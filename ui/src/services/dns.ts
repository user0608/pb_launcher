import { joinUrls } from "../utils/url";
import { HttpError } from "./client/errors";
import { pb } from "./client/pb";

export type DNSCheckStatus =
  | "resolved"
  | "match"
  | "partial_match"
  | "mismatch"
  | "not_found"
  | "timeout"
  | "error";

export type DNSCheckResult = {
  domain: string;
  status: DNSCheckStatus;
  addresses: string[];
  canonical_name?: string;
  expected_ips: string[];
  unexpected_ips: string[];
  checked_at: string;
};

export const dnsService = {
  check: async (
    domain: string,
    expectedIPs: string[],
    signal: AbortSignal,
  ): Promise<DNSCheckResult> => {
    const response = await fetch(joinUrls(pb.baseURL, "/x-api/dns/check"), {
      method: "POST",
      signal,
      headers: {
        Authorization: pb.authStore.token,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ domain, expected_ips: expectedIPs }),
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to check DNS",
        json,
      );
    }
    return json as DNSCheckResult;
  },
};
