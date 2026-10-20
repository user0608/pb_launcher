# DNS setup guide

Open the **Domains** section of a service or proxy entry and select **DNS setup**
on the generated hostname or a custom-domain card. This is a read-only assistant:
it does not edit DNS records, request certificates, start services, or contact
the returned IP addresses over HTTP/TLS.

## Configure records

At your DNS provider, create an **A** record for a reachable IPv4 address and/or
an **AAAA** record if IPv6 is actually routed to your server or external proxy.
Do not publish a stale AAAA record: some clients will prefer IPv6 even when the
A record points to the correct server.

A **CNAME** to the launcher's base domain is an alternative for a custom hostname,
provided that base domain already resolves to the correct destination. A CNAME
target is a hostname, not an IP address or a URL. Do not create a CNAME alongside
A/AAAA records for the same name. At a zone apex, use A/AAAA unless the provider
explicitly supports aliasing/flattening.

The guide shows full DNS names; providers may instead expect a relative label
or `@`. For generated instance URLs you may configure `*.your-base-domain`.
Wildcard records do not cover the base domain itself. The assistant checks a
concrete hostname, not a literal `*` wildcard.

Behind NAT, use the reachable public address rather than a container or private
LAN address. For local-only installations use the project's local DNS setup.
If a proxy/CDN terminates incoming traffic, its public addresses may be the
expected destination rather than the origin server's address.

## Check resolution

Optionally enter up to **8 expected IPv4/IPv6 addresses**, separated by commas
or whitespace, then select **Check DNS**. Do not include ports or IPv6 zone IDs.
The checker compares normalized addresses, including IPv4-mapped IPv6 addresses.

| Result | Meaning |
| --- | --- |
| Resolved | A/AAAA addresses were returned, but no expected IPs were supplied; the destination has not been verified |
| Match | Every returned address belongs to the expected set |
| Partial match | Some addresses match but others do not; review all A/AAAA and round-robin destinations |
| Mismatch | None of the returned addresses belong to the expected set |
| Not found | No usable A/AAAA addresses were returned |
| Timeout | The resolver did not complete within the deadline |
| Error | Another DNS resolution failure, such as SERVFAIL |

The guide also displays a canonical name when available and different from the
queried hostname. Alias lookup is best effort and cannot invalidate an otherwise
successful address lookup.

The source is the **launcher's configured OS resolver**. Local hosts entries,
split DNS, caches, propagation, and CDNs can produce different answers from the
browser or public resolvers. A match is a snapshot, not proof of global DNS
propagation or of HTTP/TLS readiness. It also does not require every expected IP
to appear in that particular DNS response.

Firewall rules, port forwarding, external proxy Host routing, certificate issuance,
and ACME challenge requirements must be checked separately. The old certificate
retry button is now labelled **Retry certificate** rather than **Validate DNS**
to distinguish its state-changing action from this read-only check.

## API

Authenticated endpoint: `POST /x-api/dns/check`

```json
{
  "domain": "app.example.com",
  "expected_ips": ["203.0.113.10", "2001:db8::10"]
}
```

The example IPs are documentation addresses; replace them with real destinations.
`expected_ips` may be omitted. The domain must contain only ASCII/punycode DNS
labels, optionally ending with a dot, without a scheme, path, port, wildcard,
or IP literal. Names are normalized to lowercase and resolved as absolute DNS
names to avoid OS search suffix expansion.

Responses include `domain`, `status`, `addresses`, `expected_ips`,
`unexpected_ips`, `checked_at`, and optionally `canonical_name`. DNS failures are
reported as a diagnostic result with HTTP 200; invalid input returns 400 and
unauthenticated requests return 401.

Requests are limited to 8 KiB and four simultaneous checks per launcher. Excess
concurrent checks return 429 with `Retry-After`. Resolution shares a five-second
deadline across address and alias queries and honors client cancellation. There
is no automatic polling or request retry, and responses carry `Cache-Control:
no-store`. No external DNS service, IP-discovery service, or new dependency is
required.

Automated tests use fake resolvers, not public DNS, and cover invalid inputs,
IPv4/IPv6 normalization, aliases, partial/missing results, resolver errors,
timeouts/cancellation, capacity limits, authentication and payload size.
