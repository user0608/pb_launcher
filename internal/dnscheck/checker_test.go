package dnscheck

import (
	"context"
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"
)

type fakeResolver struct {
	ips   func(context.Context, string) ([]net.IPAddr, error)
	alias func(context.Context, string) (string, error)
}

func (f fakeResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	if f.ips != nil {
		return f.ips(ctx, host)
	}
	return []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}}, nil
}

func (f fakeResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	if f.alias != nil {
		return f.alias(ctx, host)
	}
	return host, nil
}

func TestCheckerCheck(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		ips        []string
		expected   []string
		status     string
		unexpected []string
	}{
		{"resolution only is not verification", []string{"203.0.113.10"}, nil, "resolved", []string{}},
		{"IPv4 matches", []string{"203.0.113.10"}, []string{"203.0.113.10"}, "match", []string{}},
		{"mapped IPv4 matches", []string{"::ffff:203.0.113.10"}, []string{"203.0.113.10"}, "match", []string{}},
		{"canonical IPv6 matches", []string{"2001:db8::10"}, []string{"2001:0db8:0:0:0:0:0:10"}, "match", []string{}},
		{"wrong destination", []string{"203.0.113.11"}, []string{"203.0.113.10"}, "mismatch", []string{"203.0.113.11"}},
		{"stale AAAA is partial match", []string{"203.0.113.10", "2001:db8::11"}, []string{"203.0.113.10"}, "partial_match", []string{"2001:db8::11"}},
		{"round robin requires all destinations", []string{"203.0.113.10", "203.0.113.11"}, []string{"203.0.113.10"}, "partial_match", []string{"203.0.113.11"}},
		{"no addresses", nil, nil, "not_found", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			checker := NewChecker()
			checker.resolver = fakeResolver{ips: func(ctx context.Context, host string) ([]net.IPAddr, error) {
				if host != "app.example.test." {
					t.Errorf("not an absolute normalized hostname: %q", host)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Error("lookup has no deadline")
				}
				var addresses []net.IPAddr
				for _, ip := range tc.ips {
					addresses = append(addresses, net.IPAddr{IP: net.ParseIP(ip)})
				}
				return addresses, nil
			}}
			result, err := checker.Check(t.Context(), Request{Domain: " APP.Example.Test. ", ExpectedIPs: tc.expected})
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != tc.status || !reflect.DeepEqual(result.UnexpectedIPs, tc.unexpected) {
				t.Fatalf("result=%+v", result)
			}
			if result.CheckedAt.IsZero() || result.Domain != "app.example.test" || len(checker.slots) != 0 {
				t.Fatalf("incomplete result or slot leak: %+v", result)
			}
		})
	}
}

func TestCheckerDNSErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		err    error
		status string
	}{
		{"NXDOMAIN", &net.DNSError{IsNotFound: true}, "not_found"},
		{"resolver timeout", &net.DNSError{IsTimeout: true}, "timeout"},
		{"deadline", context.DeadlineExceeded, "timeout"},
		{"canceled during resolution", context.Canceled, "timeout"},
		{"SERVFAIL", &net.DNSError{IsTemporary: true}, "error"},
		{"other resolver failure", errors.New("internal resolver details"), "error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			checker := NewChecker()
			checker.resolver = fakeResolver{ips: func(context.Context, string) ([]net.IPAddr, error) { return nil, tc.err }}
			result, err := checker.Check(t.Context(), Request{Domain: "app.test"})
			if err != nil || result.Status != tc.status {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			if result.Addresses == nil || result.ExpectedIPs == nil || result.UnexpectedIPs == nil {
				t.Fatal("empty lists must be JSON arrays, not null")
			}
		})
	}
}

func TestCheckerAliasAndDeduplication(t *testing.T) {
	t.Parallel()
	for _, aliasError := range []bool{false, true} {
		name := "canonical name"
		if aliasError {
			name = "alias error preserves resolved addresses"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			checker := NewChecker()
			checker.resolver = fakeResolver{
				ips: func(context.Context, string) ([]net.IPAddr, error) {
					return []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}, {IP: net.ParseIP("203.0.113.10")}, {IP: nil}}, nil
				},
				alias: func(context.Context, string) (string, error) {
					if aliasError {
						return "", errors.New("alias unavailable")
					}
					return "TARGET.Example.Test.", nil
				},
			}
			result, err := checker.Check(t.Context(), Request{Domain: "app.test"})
			if err != nil || result.Status != "resolved" || len(result.Addresses) != 1 {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			if !aliasError && result.CanonicalName != "target.example.test" {
				t.Fatalf("alias=%q", result.CanonicalName)
			}
		})
	}
}

func TestCheckerValidationAndCapacity(t *testing.T) {
	t.Parallel()
	checker := NewChecker()
	checker.resolver = fakeResolver{ips: func(context.Context, string) ([]net.IPAddr, error) {
		t.Fatal("invalid or busy request reached DNS")
		return nil, nil
	}}
	if _, err := checker.Check(t.Context(), Request{Domain: "https://app.test"}); !errors.Is(err, ErrInvalidDomain) {
		t.Fatalf("error=%v", err)
	}
	if _, err := checker.Check(t.Context(), Request{Domain: "app.test", ExpectedIPs: []string{"not-an-IP"}}); !errors.Is(err, ErrInvalidIPs) {
		t.Fatalf("error=%v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := checker.Check(ctx, Request{Domain: "app.test"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	for range cap(checker.slots) {
		checker.slots <- struct{}{}
	}
	if _, err := checker.Check(t.Context(), Request{Domain: "app.test"}); !errors.Is(err, ErrBusy) {
		t.Fatalf("error=%v", err)
	}
}

func TestCheckerHonorsCancellation(t *testing.T) {
	t.Parallel()
	checker := NewChecker()
	started := make(chan struct{})
	checker.resolver = fakeResolver{ips: func(ctx context.Context, _ string) ([]net.IPAddr, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan Result, 1)
	go func() { result, _ := checker.Check(ctx, Request{Domain: "app.test"}); done <- result }()
	<-started
	cancel()
	result := <-done
	if result.Status != "timeout" || len(checker.slots) != 0 {
		t.Fatalf("result=%+v, slots=%d", result, len(checker.slots))
	}
}

func TestCheckerDeadline(t *testing.T) {
	t.Parallel()
	checker := NewChecker()
	checker.timeout = 0
	checker.resolver = fakeResolver{ips: func(ctx context.Context, _ string) ([]net.IPAddr, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	result, err := checker.Check(t.Context(), Request{Domain: "app.test"})
	if err != nil || result.Status != "timeout" || len(checker.slots) != 0 {
		t.Fatalf("result=%+v error=%v slots=%d", result, err, len(checker.slots))
	}
}

func TestNormalizeDomain(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, value string
		valid       bool
	}{
		{"hostname", "app.example.test", true},
		{"punycode", "xn--bcher-kva.example", true},
		{"local hostname", "localhost", true},
		{"scheme", "https://app.test", false},
		{"port", "app.test:443", false},
		{"path", "app.test/path", false},
		{"wildcard", "*.app.test", false},
		{"query", "app.test?x=1", false},
		{"empty label", "app..test", false},
		{"underscore", "_service.test", false},
		{"hyphen boundary", "-app.test", false},
		{"IPv4", "127.0.0.1", false},
		{"IPv6", "::1", false},
		{"long label", strings.Repeat("a", 64) + ".test", false},
		{"long hostname", strings.Repeat("a.", 127) + "test", false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := normalizeDomain(tc.value)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
}

func TestNormalizeIPs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		values []string
		valid  bool
	}{
		{"empty is optional", nil, true},
		{"IPv4 and IPv6", []string{"203.0.113.10", "2001:db8::10"}, true},
		{"zone identifier", []string{"fe80::1%eth0"}, false},
		{"port", []string{"203.0.113.10:80"}, false},
		{"hostname", []string{"app.test"}, false},
		{"too many", make([]string, 9), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := normalizeIPs(tc.values)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
}
