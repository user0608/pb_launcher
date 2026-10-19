import { type FC, useMemo, useRef } from "react";

import classNames from "classnames";

import type { ProxyConfigsResponse } from "../../../services/config";
import { formatProxyUrl } from "../../../utils/url";
import { CopyableField } from "./CopyableField";
import type { ProxyEntryDto } from "../../../services/proxy";
import { MoreVertical, Pencil, Trash2 } from "lucide-react";

type Props = {
  proxyInfo: ProxyConfigsResponse;
  entry: ProxyEntryDto;
  onDetails: () => void;
  onDelete: () => void;
};

export const ProxyCard: FC<Props> = ({
  proxyInfo,
  entry,
  onDelete,
  onDetails,
}) => {
  const dropdownRef = useRef<HTMLDivElement>(null);
  const urls = useMemo(() => {
    const domains: string[] = [];

    if (proxyInfo.base_domain) {
      domains.push(`${entry.id}.${proxyInfo.base_domain}`);
    }

    if (entry.domains) {
      domains.push(...entry.domains.map(d => d.domain));
    }

    return domains.map(domain =>
      formatProxyUrl(
        proxyInfo.use_https ? "https" : "http",
        domain,
        proxyInfo.use_https ? proxyInfo.https_port : proxyInfo.http_port,
        proxyInfo.public_url,
      ),
    );
  }, [proxyInfo, entry]);

  const executeAfterBlur = (fn: () => void) => {
    (document.activeElement as HTMLElement)?.blur?.();
    fn();
  };

  return (
    <div className="card bg-base-100 shadow border border-base-300">
      <div className="card-body">
        <div className="flex justify-between items-start">
          <h2 className="card-title text-base-content">{entry.name}</h2>
          <div className="flex gap-4 items-center">
            <div
              ref={dropdownRef}
              className="dropdown dropdown-end select-none relative -right-3 -top-2"
            >
              <label
                tabIndex={0}
                className="btn btn-sm btn-ghost btn-circle text-base-content"
              >
                <MoreVertical className="w-4 h-4" />
              </label>
              <ul
                tabIndex={0}
                className="dropdown-content menu p-2 shadow-lg bg-base-100 text-base-content rounded-box min-w-[10rem] z-[1] space-y-1"
              >
                <li>
                  <button
                    className="flex items-center gap-2 w-full justify-start hover:bg-base-200 text-primary"
                    onClick={() => executeAfterBlur(onDetails)}
                  >
                    <Pencil className="w-4 h-4" />
                    Details
                  </button>
                </li>
                <li>
                  <button
                    onClick={() => executeAfterBlur(onDelete)}
                    className="flex items-center gap-2 w-full justify-start text-error hover:bg-base-200"
                  >
                    <Trash2 className="w-4 h-4" />
                    Delete
                  </button>
                </li>
              </ul>
            </div>
          </div>
        </div>
        <div className="text-sm space-y-1 text-base-content/80">
          <div className="flex justify-between">
            <span className="font-medium">Target URL:</span>
            <span className="truncate text-right">{entry.target_url}</span>
          </div>
          <div className="flex justify-between">
            <span className="font-medium">Enabled:</span>
            <span
              className={classNames("badge badge-sm", {
                "badge-success": entry.enabled === "yes",
                "badge-neutral": entry.enabled === "no",
              })}
            >
              {entry.enabled.charAt(0).toUpperCase() + entry.enabled.slice(1)}
            </span>
          </div>
        </div>

        {urls.map(url => {
          return <CopyableField key={url} value={url} isUrl />;
        })}
      </div>
    </div>
  );
};
