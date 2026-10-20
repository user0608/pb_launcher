package dnscheck

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalidDomain = errors.New("domain must be a DNS hostname without a scheme, port, path or wildcard; use ASCII/punycode labels")
	ErrInvalidIPs    = errors.New("expected_ips must contain at most 8 IPv4 or IPv6 addresses, without ports or zone identifiers")
	ErrBusy          = errors.New("DNS checks are busy; try again shortly")
)

type resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
	LookupCNAME(context.Context, string) (string, error)
}

type Request struct {
	Domain      string   `json:"domain"`
	ExpectedIPs []string `json:"expected_ips"`
}

type Result struct {
	Domain        string    `json:"domain"`
	Status        string    `json:"status"`
	Addresses     []string  `json:"addresses"`
	CanonicalName string    `json:"canonical_name,omitempty"`
	ExpectedIPs   []string  `json:"expected_ips"`
	UnexpectedIPs []string  `json:"unexpected_ips"`
	CheckedAt     time.Time `json:"checked_at"`
}

type Checker struct {
	resolver resolver
	timeout  time.Duration
	slots    chan struct{}
}

func NewChecker() *Checker {
	return &Checker{resolver: net.DefaultResolver, timeout: 5 * time.Second, slots: make(chan struct{}, 4)}
}

// Check only performs DNS resolution. It never connects to a returned address,
// probes HTTP/TLS, writes records or changes certificate/service state.
func (c *Checker) Check(ctx context.Context, request Request) (Result, error) {
	domain, err := normalizeDomain(request.Domain)
	if err != nil {
		return Result{}, err
	}
	expected, err := normalizeIPs(request.ExpectedIPs)
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	default:
		return Result{}, ErrBusy
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	result := Result{
		Domain: domain, ExpectedIPs: expected, Addresses: []string{},
		UnexpectedIPs: []string{}, Status: "error",
	}
	// A trailing dot prevents resolver search suffixes from changing the name.
	addresses, lookupErr := c.resolver.LookupIPAddr(ctx, domain+".")
	if lookupErr != nil {
		var dnsErr *net.DNSError
		switch {
		case errors.Is(lookupErr, context.DeadlineExceeded), errors.Is(lookupErr, context.Canceled):
			result.Status = "timeout"
		case errors.As(lookupErr, &dnsErr):
			if dnsErr.IsTimeout {
				result.Status = "timeout"
			} else if dnsErr.IsNotFound {
				result.Status = "not_found"
			}
		}
	} else {
		seen := make(map[string]bool)
		for _, item := range addresses {
			if ip, ok := netip.AddrFromSlice(item.IP); ok && item.Zone == "" {
				seen[ip.Unmap().String()] = true
			}
		}
		for ip := range seen {
			result.Addresses = append(result.Addresses, ip)
		}
		sort.Strings(result.Addresses)
		result.Status, result.UnexpectedIPs = compareAddresses(result.Addresses, expected)
		if len(result.Addresses) > 0 {
			// Alias information is best effort, within the same deadline. Failure
			// here must not discard a successful A/AAAA result.
			if name, err := c.resolver.LookupCNAME(ctx, domain+"."); err == nil {
				if name, err = normalizeDomain(name); err == nil && name != domain {
					result.CanonicalName = name
				}
			}
		}
	}
	result.CheckedAt = time.Now().UTC()
	return result, nil
}

func normalizeDomain(raw string) (string, error) {
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(raw), "."))
	if len(domain) == 0 || len(domain) > 253 || net.ParseIP(domain) != nil {
		return "", ErrInvalidDomain
	}
	for _, label := range strings.Split(domain, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", ErrInvalidDomain
		}
		for _, ch := range label {
			if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' {
				return "", ErrInvalidDomain
			}
		}
	}
	return domain, nil
}

func normalizeIPs(raw []string) ([]string, error) {
	if len(raw) > 8 {
		return nil, ErrInvalidIPs
	}
	seen := make(map[string]bool)
	for _, value := range raw {
		ip, err := netip.ParseAddr(strings.TrimSpace(value))
		if err != nil || ip.Zone() != "" {
			return nil, ErrInvalidIPs
		}
		seen[ip.Unmap().String()] = true
	}
	result := make([]string, 0, len(seen))
	for ip := range seen {
		result = append(result, ip)
	}
	sort.Strings(result)
	return result, nil
}

func compareAddresses(addresses, expected []string) (string, []string) {
	unexpected := []string{}
	if len(addresses) == 0 {
		return "not_found", unexpected
	}
	if len(expected) == 0 {
		return "resolved", unexpected
	}
	allowed := make(map[string]bool, len(expected))
	for _, ip := range expected {
		allowed[ip] = true
	}
	for _, ip := range addresses {
		if !allowed[ip] {
			unexpected = append(unexpected, ip)
		}
	}
	switch len(unexpected) {
	case 0:
		return "match", unexpected
	case len(addresses):
		return "mismatch", unexpected
	default:
		return "partial_match", unexpected
	}
}
