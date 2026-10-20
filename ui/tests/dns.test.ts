import assert from "node:assert/strict";
import { test } from "node:test";
import { dnsStatusMessage, parseExpectedIPs } from "../src/utils/dns.ts";

test("expected IPs are optional and can be separated by spaces/commas", () => {
  assert.deepEqual(parseExpectedIPs(""), []);
  assert.deepEqual(
    parseExpectedIPs(" 203.0.113.10, 2001:db8::10\n203.0.113.11 "),
    ["203.0.113.10", "2001:db8::10", "203.0.113.11"],
  );
});

test("resolution without expected IPs is not advertised as a verified destination", () => {
  assert.match(dnsStatusMessage("resolved"), /Enter expected IPs/);
});

test("successful DNS does not claim TLS readiness", () => {
  assert.match(dnsStatusMessage("match"), /HTTP and TLS have not been checked/);
});

test("partial matches warn about incorrect A/AAAA records", () => {
  assert.match(dnsStatusMessage("partial_match"), /Check all A\/AAAA/);
});

test("DNS failures have distinct actionable messages", () => {
  assert.match(dnsStatusMessage("not_found"), /No usable A\/AAAA/);
  assert.match(dnsStatusMessage("timeout"), /timed out/);
  assert.match(dnsStatusMessage("error"), /resolution failed/);
  assert.match(dnsStatusMessage("mismatch"), /CDN\/proxy/);
});
