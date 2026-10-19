import { joinUrls } from "../utils/url";
import { HttpError } from "./client/errors";
import { pb } from "./client/pb";
export interface ProxyConfigsResponse {
  use_https?: boolean;
  http_port?: string;
  https_port?: string;
  base_domain?: string;
  public_url?: string;
}

export const configService = {
  fetchProxyConfigs: async (
    signal: AbortSignal,
  ): Promise<ProxyConfigsResponse> => {
    const url = joinUrls(pb.baseURL, "/x-api/proxy_configs");
    const response = await fetch(url, { signal });
    const json = await response.json();

    if (!response.ok) {
      throw new HttpError(
        response.status,
        json?.message || "Unexpected error",
        json,
      );
    }
    return json as ProxyConfigsResponse;
  },
};
