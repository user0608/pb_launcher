import assert from "node:assert/strict";
import { test } from "node:test";
import { formatProxyUrl } from "../src/utils/url.ts";

for (const scenario of [
  { name: "direct HTTP retains 7080", scheme: "http", port: "7080", publicUrl: undefined, expected: "http://service.pb.test:7080" },
  { name: "direct HTTPS retains 8443", scheme: "https", port: "8443", publicUrl: undefined, expected: "https://service.pb.test:8443" },
  { name: "external TLS uses public port", scheme: "http", port: "7080", publicUrl: "https://pb.test", expected: "https://service.pb.test" },
  { name: "external custom port preserved", scheme: "http", port: "7080", publicUrl: "https://pb.test:9443", expected: "https://service.pb.test:9443" },
]) {
  test(scenario.name, () => {
    assert.equal(formatProxyUrl(scenario.scheme, "service.pb.test", scenario.port, scenario.publicUrl), scenario.expected);
  });
}

test("custom domain hostname is retained behind external TLS", () => {
  assert.equal(formatProxyUrl("http", "custom.test", "7080", "https://pb.test"), "https://custom.test");
});
