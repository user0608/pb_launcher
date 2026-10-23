import type { ServiceDto } from "../services/services";

export const serviceStatusOptions = [
  { value: "all", label: "All statuses" },
  { value: "idle", label: "Idle" },
  { value: "pending", label: "Pending" },
  { value: "running", label: "Running" },
  { value: "stopped", label: "Stopped" },
  { value: "failure", label: "Failure" },
  { value: "restoring", label: "Restoring" },
] as const satisfies readonly {
  value: "all" | ServiceDto["status"];
  label: string;
}[];

export type ServiceStatusFilter =
  (typeof serviceStatusOptions)[number]["value"];
export type ServiceFilters = {
  query: string;
  status: ServiceStatusFilter;
  repositoryID: string;
  version: string;
};

type FilterableService = Pick<
  ServiceDto,
  "id" | "name" | "status" | "repository" | "repository_id" | "release_version"
>;

// Preserve the existing localStorage shape, but tolerate obsolete/invalid values.
export const readStoredStatusFilter = (
  stored: unknown,
): ServiceStatusFilter => {
  if (typeof stored === "object" && stored !== null && "value" in stored) {
    return (
      serviceStatusOptions.find(option => option.value === stored.value)
        ?.value ?? "all"
    );
  }
  return "all";
};

export const filterServices = <T extends FilterableService>(
  services: readonly T[],
  filters: ServiceFilters,
): T[] => {
  const query = filters.query.trim().toLowerCase();
  return services.filter(
    service =>
      (service.name.toLowerCase().includes(query) ||
        service.id.toLowerCase().includes(query)) &&
      (filters.status === "all" || service.status === filters.status) &&
      (!filters.repositoryID ||
        service.repository_id === filters.repositoryID) &&
      (!filters.version || service.release_version === filters.version),
  );
};

export const getServiceFilterOptions = (
  services: readonly FilterableService[],
  repositoryID: string,
) => {
  const repositories = new Map<string, string>();
  const versions = new Set<string>();
  for (const service of services) {
    if (service.repository_id)
      repositories.set(service.repository_id, service.repository);
    if (
      (!repositoryID || service.repository_id === repositoryID) &&
      service.release_version
    ) {
      versions.add(service.release_version);
    }
  }
  return {
    repositories: Array.from(repositories, ([id, name]) => ({ id, name })).sort(
      (a, b) => a.name.localeCompare(b.name),
    ),
    versions: Array.from(versions).sort((a, b) =>
      a.localeCompare(b, "en", { numeric: true }),
    ),
  };
};
