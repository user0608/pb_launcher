import assert from "node:assert/strict";
import { test } from "node:test";
import {
  filterServices,
  getServiceFilterOptions,
  readStoredStatusFilter,
  serviceStatusOptions,
  type ServiceFilters,
} from "../src/utils/serviceFilters.ts";

const services = [
  {
    id: "IDAlpha",
    name: "Alpha API",
    status: "running",
    repository: "PocketBase",
    repository_id: "repo-a",
    release_version: "0.9.0",
  },
  {
    id: "IDBeta",
    name: "Beta API",
    status: "stopped",
    repository: "Custom build",
    repository_id: "repo-b",
    release_version: "0.9.0",
  },
  {
    id: "IDGamma",
    name: "Gamma",
    status: "failure",
    repository: "PocketBase",
    repository_id: "repo-a",
    release_version: "0.10.0",
  },
  {
    id: "IDDelta",
    name: "Delta",
    status: "idle",
    repository: "PocketBase",
    repository_id: "repo-a",
    release_version: "0.10.0",
  },
  {
    id: "IDEpsilon",
    name: "Epsilon",
    status: "pending",
    repository: "Custom build",
    repository_id: "repo-b",
    release_version: "0.11.0",
  },
  {
    id: "IDZeta",
    name: "Zeta",
    status: "restoring",
    repository: "PocketBase",
    repository_id: "repo-a",
    release_version: "0.10.0",
  },
] as const;

const defaults: ServiceFilters = {
  query: "",
  status: "all",
  repositoryID: "",
  version: "",
};

test("no filters preserve all instances and their order", () => {
  assert.deepEqual(filterServices(services, defaults), services);
});

for (const scenario of [
  {
    name: "case-insensitive name",
    filters: { query: "ALPHA API" },
    ids: ["IDAlpha"],
  },
  { name: "case-insensitive ID", filters: { query: "idb" }, ids: ["IDBeta"] },
  {
    name: "whitespace trimmed",
    filters: { query: "  ALPHA  " },
    ids: ["IDAlpha"],
  },
  {
    name: "whitespace-only query",
    filters: { query: " \t " },
    ids: services.map(service => service.id),
  },
  {
    name: "repository uses ID not display name",
    filters: { repositoryID: "repo-b" },
    ids: ["IDBeta", "IDEpsilon"],
  },
  {
    name: "version shared across repositories",
    filters: { version: "0.9.0" },
    ids: ["IDAlpha", "IDBeta"],
  },
  { name: "exact version", filters: { version: "0.9" }, ids: [] },
  {
    name: "repository and version",
    filters: { repositoryID: "repo-b", version: "0.9.0" },
    ids: ["IDBeta"],
  },
  {
    name: "all filters combine with AND",
    filters: {
      query: "api",
      status: "stopped",
      repositoryID: "repo-b",
      version: "0.9.0",
    },
    ids: ["IDBeta"],
  },
  {
    name: "contradictory filters",
    filters: { query: "alpha", status: "stopped" },
    ids: [],
  },
  {
    name: "unavailable repository remains a real filter",
    filters: { repositoryID: "removed-repository" },
    ids: [],
  },
  {
    name: "unavailable version remains a real filter",
    filters: { version: "99.0.0" },
    ids: [],
  },
] satisfies {
  name: string;
  filters: Partial<ServiceFilters>;
  ids: readonly string[];
}[]) {
  test(scenario.name, () => {
    assert.deepEqual(
      filterServices(services, { ...defaults, ...scenario.filters }).map(
        service => service.id,
      ),
      scenario.ids,
    );
  });
}

for (const option of serviceStatusOptions.filter(
  option => option.value !== "all",
)) {
  test(`status ${option.value} is supported`, () => {
    const result = filterServices(services, {
      ...defaults,
      status: option.value,
    });
    assert.equal(result.length, 1);
    assert.equal(result[0].status, option.value);
  });
}

test("empty service lists have no matches or options", () => {
  assert.deepEqual(filterServices([], defaults), []);
  assert.deepEqual(getServiceFilterOptions([], ""), {
    repositories: [],
    versions: [],
  });
});

test("filter options are deduplicated, sorted and scoped by repository", () => {
  const options = getServiceFilterOptions(services, "");
  assert.deepEqual(options.repositories, [
    { id: "repo-b", name: "Custom build" },
    { id: "repo-a", name: "PocketBase" },
  ]);
  assert.deepEqual(options.versions, ["0.9.0", "0.10.0", "0.11.0"]);
  const selected = getServiceFilterOptions(services, "repo-b");
  assert.deepEqual(selected.versions, ["0.9.0", "0.11.0"]);
  assert.deepEqual(selected.repositories, options.repositories);
  assert.deepEqual(getServiceFilterOptions(services, "missing").versions, []);
});

test("filtering does not mutate services or their data", () => {
  const frozen = Object.freeze(
    services.map(service => Object.freeze({ ...service })),
  );
  const before = JSON.stringify(frozen);
  const result = filterServices(frozen, { ...defaults, query: "alpha" });
  assert.equal(result[0], frozen[0]);
  getServiceFilterOptions(frozen, "repo-b");
  assert.equal(JSON.stringify(frozen), before);
});

test("filtering reflects refreshed status data", () => {
  const filters = { ...defaults, status: "running" as const };
  assert.equal(filterServices(services, filters).length, 1);
  const refreshed = services.map(service =>
    service.id === "IDAlpha"
      ? { ...service, status: "stopped" as const }
      : service,
  );
  assert.equal(filterServices(refreshed, filters).length, 0);
});

test("existing stored status preferences are preserved", () => {
  for (const { value } of serviceStatusOptions)
    assert.equal(readStoredStatusFilter({ value }), value);
});

test("invalid or obsolete stored preferences fall back to all statuses", () => {
  for (const value of [
    undefined,
    null,
    "running",
    [],
    {},
    { value: null },
    { value: "sleeping" },
  ]) {
    assert.equal(readStoredStatusFilter(value), "all");
  }
});
