import type { DNSCheckStatus } from "../services/dns";

export const parseExpectedIPs = (value: string): string[] =>
  value
    .trim()
    .split(/[\s,]+/)
    .filter(Boolean);

// DNS resolution is not an HTTP/TLS or certificate validation result.
export const dnsStatusMessage = (status: DNSCheckStatus): string => {
  switch (status) {
    case "match":
      return "All resolved addresses match the expected IPs. HTTP and TLS have not been checked.";
    case "resolved":
      return "The domain resolves. Enter expected IPs to check whether it points to your server or proxy.";
    case "partial_match":
      return "Only some resolved addresses match. Check all A/AAAA records; clients may reach a different server.";
    case "mismatch":
      return "None of the resolved addresses match the expected IPs. Check your records or whether a CDN/proxy is in use.";
    case "not_found":
      return "No usable A/AAAA addresses were found. Check the hostname, records and DNS propagation.";
    case "timeout":
      return "The DNS lookup timed out. Try again or check the launcher's DNS resolver.";
    case "error":
      return "DNS resolution failed. Check your DNS provider and the launcher's resolver, then retry.";
  }
};
