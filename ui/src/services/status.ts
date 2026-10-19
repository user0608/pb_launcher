import { joinUrls } from "../utils/url";
import { HttpError } from "./client/errors";
import { pb } from "./client/pb";

export type SystemStatus = {
  sampled_at: string;
  os: string;
  cpu: { usage_percent: number | null; cores: number };
  memory: { total_bytes: number; used_bytes: number } | null;
  disk: { total_bytes: number; used_bytes: number; path: string } | null;
  uptime_seconds: number | null;
  active_instances: number;
  instances: {
    service_id: string;
    cpu_percent: number | null;
    memory_bytes: number | null;
  }[];
};

export const statusService = {
  fetch: async (signal: AbortSignal): Promise<SystemStatus> => {
    const response = await fetch(joinUrls(pb.baseURL, "/x-api/system/status"), {
      signal,
      headers: { Authorization: pb.authStore.token },
    });
    const json = await response.json().catch(() => null);
    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Failed to fetch system status",
        json,
      );
    }
    return json as SystemStatus;
  },
  fetchNames: (signal: AbortSignal) =>
    pb.collection("services").getFullList<{ id: string; name: string }>({
      fields: "id,name",
      filter: 'deleted=""',
      signal,
      requestKey: "system-status-service-names",
    }),
};
