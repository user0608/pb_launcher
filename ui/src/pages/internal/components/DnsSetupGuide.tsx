import { useId, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import classNames from "classnames";
import { dnsService } from "../../../services/dns";
import { dnsStatusMessage, parseExpectedIPs } from "../../../utils/dns";
import { getErrorMessage } from "../../../utils/errors";

type Props = { domain: string; baseDomain?: string };

export const DnsSetupGuide = ({ domain, baseDomain }: Props) => {
  const inputId = useId();
  const [expected, setExpected] = useState("");
  const expectedIPs = parseExpectedIPs(expected);
  const query = useQuery({
    queryKey: ["dns-check", domain, expectedIPs],
    queryFn: ({ signal }) => dnsService.check(domain, expectedIPs, signal),
    enabled: false,
    retry: false,
    gcTime: 0,
  });
  const result = query.data;

  return (
    <div className="w-full min-w-0 max-w-2xl space-y-5">
      <p className="break-all font-medium">{domain}</p>
      <section className="space-y-2">
        <h3 className="font-semibold">1. Choose the destination</h3>
        <p className="text-sm">
          Use the address of the server or external proxy that receives traffic
          for this domain. Behind NAT, use the reachable public address, not the
          container or LAN address. For local testing, use your local DNS setup.
        </p>
      </section>

      <section className="space-y-2">
        <h3 className="font-semibold">
          2. Create DNS records at your DNS provider
        </h3>
        <div className="overflow-x-auto">
          <table className="table table-sm">
            <thead>
              <tr>
                <th scope="col">Type</th>
                <th scope="col">Name</th>
                <th scope="col">Value</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>A</td>
                <td className="break-all">{domain}</td>
                <td>Reachable server/proxy IPv4 address</td>
              </tr>
              <tr>
                <td>AAAA</td>
                <td className="break-all">{domain}</td>
                <td>
                  IPv6 address, only if IPv6 traffic reaches this server/proxy
                </td>
              </tr>
              {baseDomain && domain !== baseDomain && (
                <tr>
                  <td>
                    CNAME
                    <br />
                    (alternative)
                  </td>
                  <td className="break-all">{domain}</td>
                  <td className="break-all">{baseDomain}</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <ul className="list-disc space-y-1 pl-5 text-sm">
          <li>
            Use A/AAAA records, or a CNAME alias—not both for the same name.
          </li>
          <li>
            CNAME targets are hostnames, never URLs, paths or IP addresses. The
            target must already resolve to the correct destination.
          </li>
          <li>
            At the zone apex, use A/AAAA unless your provider explicitly
            supports alias/flattening.
          </li>
          <li>
            Some providers expect a relative name or @ instead of the full
            hostname above.
          </li>
          {baseDomain && (
            <li className="break-words">
              For generated instance URLs, you can configure *.{baseDomain}. A
              wildcard does not cover {baseDomain} itself.
            </li>
          )}
          <li>
            Remove stale AAAA records if IPv6 is not configured. A correct A
            record alone does not fix incorrect IPv6 routing.
          </li>
        </ul>
      </section>

      <section className="space-y-3">
        <h3 className="font-semibold">3. Check DNS resolution</h3>
        <form
          className="space-y-3"
          onSubmit={event => {
            event.preventDefault();
            void query.refetch();
          }}
        >
          <div className="space-y-1">
            <label className="block text-sm font-medium" htmlFor={inputId}>
              Expected server/proxy IPs (optional)
            </label>
            <input
              id={inputId}
              value={expected}
              onChange={event => setExpected(event.target.value)}
              className="input input-bordered w-full"
              maxLength={512}
              autoComplete="off"
              spellCheck={false}
              placeholder="203.0.113.10, 2001:db8::10"
              aria-describedby={`${inputId}-help`}
            />
            <p id={`${inputId}-help`} className="text-sm text-base-content/70">
              Separate IPv4/IPv6 addresses with commas or spaces (maximum 8). No
              ports. Without these, we only report resolution—not whether the
              destination is correct.
            </p>
          </div>
          <button
            className="btn btn-primary btn-sm"
            type="submit"
            disabled={query.isFetching}
          >
            {query.isFetching ? "Checking DNS…" : "Check DNS"}
          </button>
        </form>
        <div aria-live="polite" aria-atomic="true">
          {query.isError && (
            <p role="alert" className="text-error">
              {getErrorMessage(query.error)}
            </p>
          )}
          {!query.isFetching && !query.isError && result && (
            <div className="space-y-2 rounded-lg border border-base-300 p-3">
              <p
                className={classNames("text-sm font-medium", {
                  "text-success": result.status === "match",
                  "text-warning":
                    result.status === "partial_match" ||
                    result.status === "mismatch",
                  "text-error": ["not_found", "timeout", "error"].includes(
                    result.status,
                  ),
                })}
              >
                {dnsStatusMessage(result.status)}
              </p>
              <dl className="space-y-1 text-sm">
                <dt className="font-medium">Resolved addresses</dt>
                <dd className="break-all">
                  {result.addresses.length
                    ? result.addresses.join(", ")
                    : "None reported"}
                </dd>
                {result.canonical_name && (
                  <>
                    <dt className="font-medium">
                      Canonical name (alias target)
                    </dt>
                    <dd className="break-all">{result.canonical_name}</dd>
                  </>
                )}
                {result.unexpected_ips.length > 0 && (
                  <>
                    <dt className="font-medium">Unexpected addresses</dt>
                    <dd className="break-all">
                      {result.unexpected_ips.join(", ")}
                    </dd>
                  </>
                )}
                <dt className="font-medium">Checked at</dt>
                <dd>{new Date(result.checked_at).toLocaleString()}</dd>
              </dl>
            </div>
          )}
        </div>
        <p className="text-sm text-base-content/70">
          This uses the launcher's resolver. Your browser or public DNS may see
          different cached answers. Changes can take time to propagate. A CDN
          may intentionally return its own IPs.
        </p>
      </section>

      <section className="space-y-2">
        <h3 className="font-semibold">
          4. Check routing and certificates separately
        </h3>
        <p className="text-sm">
          DNS does not configure firewall rules, port forwarding, proxy host
          routing or TLS certificates. Ensure traffic reaches the correct
          HTTP/HTTPS port. Only request a certificate when the relevant DNS and
          challenge requirements are met.
        </p>
        <p className="text-sm text-base-content/70">
          This guide never changes DNS records, starts an instance, or requests
          a certificate.
        </p>
      </section>
    </div>
  );
};
