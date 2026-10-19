import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { ErrorFallback } from "../../components/helpers/ErrorFallback";
import { statusService } from "../../services/status";

const unavailable = "Not available";

const formatBytes = (bytes: number | null | undefined): string => {
  if (bytes == null) return unavailable;
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  const index =
    bytes > 0 ? Math.min(4, Math.floor(Math.log(bytes) / Math.log(1024))) : 0;
  return `${(bytes / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
};

const formatPercent = (percent: number | null | undefined): string =>
  percent == null ? unavailable : `${percent.toFixed(1)}%`;

const Metric = ({
  title,
  value,
  detail,
  percent,
}: {
  title: string;
  value: string;
  detail: string;
  percent?: number | null;
}) => (
  <section className="card bg-base-100 border border-base-300 shadow-sm">
    <div className="card-body gap-2">
      <h3 className="font-medium text-base-content/70">{title}</h3>
      <p className="text-2xl font-semibold">{value}</p>
      <p className="text-sm text-base-content/70 break-words">{detail}</p>
      {percent != null && (
        <progress
          className="progress progress-primary w-full"
          aria-label={title}
          value={Math.min(100, Math.max(0, percent))}
          max={100}
        />
      )}
    </div>
  </section>
);

export const StatusPage = () => {
  const query = useQuery({
    queryKey: ["system-status"],
    queryFn: ({ signal }) => statusService.fetch(signal),
    refetchInterval: 5000,
  });
  const namesQuery = useQuery({
    queryKey: ["system-status", "service-names"],
    queryFn: ({ signal }) => statusService.fetchNames(signal),
    refetchInterval: 10000,
  });

  if (!query.data) {
    if (query.isError)
      return (
        <ErrorFallback
          error={query.error}
          onRetry={() => {
            void query.refetch();
          }}
        />
      );
    return (
      <div className="p-4" role="status">
        Loading system resources...
      </div>
    );
  }

  const status = query.data;
  const names = new Map(
    (namesQuery.data ?? []).map(service => [service.id, service.name]),
  );
  const memoryPercent = status.memory
    ? (100 * status.memory.used_bytes) / status.memory.total_bytes
    : null;
  const diskPercent = status.disk
    ? (100 * status.disk.used_bytes) / status.disk.total_bytes
    : null;
  const uptime = status.uptime_seconds;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold">System Resources</h2>
          <p className="text-sm text-base-content/70">
            Refreshes every 5 seconds while this page is visible.
          </p>
        </div>
        <button
          className="btn btn-sm btn-outline"
          disabled={query.isFetching}
          onClick={() => {
            void query.refetch();
          }}
        >
          Refresh
        </button>
      </div>
      {query.isError && (
        <p role="alert" className="text-error">
          Refresh failed. Showing the last successful sample.
        </p>
      )}
      <p className="text-sm text-base-content/70">
        OS: {status.os} · Last sample:{" "}
        {new Date(status.sampled_at).toLocaleTimeString()}
      </p>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric
          title="System CPU"
          value={formatPercent(status.cpu.usage_percent)}
          detail={`${status.cpu.cores} logical cores · first usage sample may be unavailable`}
          percent={status.cpu.usage_percent}
        />
        <Metric
          title="System Memory"
          value={formatBytes(status.memory?.used_bytes)}
          detail={
            status.memory
              ? `of ${formatBytes(status.memory.total_bytes)}`
              : unavailable
          }
          percent={memoryPercent}
        />
        <Metric
          title="Data Filesystem"
          value={formatBytes(status.disk?.used_bytes)}
          detail={
            status.disk
              ? `of ${formatBytes(status.disk.total_bytes)} · ${status.disk.path}`
              : unavailable
          }
          percent={diskPercent}
        />
        <Metric
          title="Active Instances"
          value={String(status.active_instances)}
          detail={
            uptime == null
              ? `System uptime: ${unavailable}`
              : `System uptime: ${Math.floor(uptime / 86400)}d ${Math.floor(uptime / 3600) % 24}h ${Math.floor(uptime / 60) % 60}m`
          }
        />
      </div>
      <p className="text-sm text-base-content/70">
        System metrics describe the OS visible to the launcher, not container
        CPU or memory quotas. Unsupported or inaccessible measurements are shown
        as “{unavailable}”, not zero.
      </p>
      <section className="card bg-base-100 border border-base-300 shadow-sm">
        <div className="card-body">
          <h3 className="card-title text-base">Running Instances</h3>
          <p className="text-sm text-base-content/70">
            CPU: 100% equals one logical core. Memory is resident process memory
            (RSS).
          </p>
          {namesQuery.isError && (
            <p role="alert" className="text-warning">
              Service names could not be loaded.
            </p>
          )}
          {status.instances.length === 0 ? (
            <p>No instances are running.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="table table-sm">
                <thead>
                  <tr>
                    <th scope="col">Service</th>
                    <th scope="col">CPU</th>
                    <th scope="col">Memory (RSS)</th>
                  </tr>
                </thead>
                <tbody>
                  {status.instances.map(instance => (
                    <tr key={instance.service_id}>
                      <td>
                        <Link
                          className="link link-primary"
                          to={`/services/${instance.service_id}`}
                        >
                          {names.get(instance.service_id) ?? "Service"}
                        </Link>
                      </td>
                      <td>{formatPercent(instance.cpu_percent)}</td>
                      <td>{formatBytes(instance.memory_bytes)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </section>
    </div>
  );
};
