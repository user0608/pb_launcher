import { useMemo, useRef, type FC } from "react";
import type { ServiceDto } from "../../../services/services";
import {
  Copy,
  MoreVertical,
  Pencil,
  Power,
  Save,
  ShieldAlert,
  Trash2,
  Upload,
} from "lucide-react";
import classNames from "classnames";
import { useModal } from "../../../components/modal/hook";
import { DefaultCredentialsCard } from "./DefaultCredentialsCard";
import type { ProxyConfigsResponse } from "../../../services/config";
import { formatProxyUrl } from "../../../utils/url";
import { CopyableField } from "./CopyableField";

type Props = {
  proxyInfo: ProxyConfigsResponse;
  service: ServiceDto;
  onDetails: () => void;
  onDelete: () => void;
  onStart: () => void;
  onStop: () => void;
  onRestart: () => void;
  onUpgrade: () => void;
  onBackup: () => void;
  onClone: () => void;
  refreshData: () => void;
};

export const ServiceCard: FC<Props> = ({
  proxyInfo,
  service,
  onDetails,
  onDelete,
  onRestart,
  onStart,
  onStop,
  onUpgrade,
  onBackup,
  onClone,
  refreshData,
}) => {
  const dropdownRef = useRef<HTMLDivElement>(null);

  const { openModal } = useModal();

  const serviceUrls = useMemo((): string[] => {
    const domains: string[] = [];
    if (proxyInfo.base_domain) {
      domains.push(`${service.id}.${proxyInfo.base_domain}`);
    }
    domains.push(...(service.domains ?? []).map(d => d.domain));
    return domains.map(domain => {
      const urlStr = formatProxyUrl(
        proxyInfo.use_https ? "https" : "http",
        domain,
        proxyInfo.use_https ? proxyInfo.https_port : proxyInfo.http_port,
        proxyInfo.public_url,
      );
      if (service._pb_install)
        return `${urlStr}/_/#/pbinstal/${service._pb_install}`;
      return `${urlStr}/_/`;
    });
  }, [proxyInfo, service]);

  const executeAfterBlur = (fn: () => void) => {
    (document.activeElement as HTMLElement)?.blur?.();
    fn();
  };

  const showDefaultCredentials = () => {
    openModal(
      <DefaultCredentialsCard
        service_id={service.id}
        username={service.boot_user_email}
        password={service.boot_user_password}
        onResetCredentials={refreshData}
      />,
    );
  };

  return (
    <div
      key={service.id}
      className="card bg-base-100 shadow border border-base-300"
    >
      <div className="card-body">
        <div className="flex justify-between items-start">
          <h2 className="card-title text-base-content">{service.name}</h2>
          <div className="flex gap-4 items-center">
            <ShieldAlert
              className="w-4 h-4 active:translate-[0.5px] relative -right-3 -top-2 text-gray-300"
              onClick={showDefaultCredentials}
            />

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
                  <details className="group">
                    <summary className="flex items-center gap-2 text-base-content cursor-pointer select-none py-1 px-2 hover:bg-base-200 rounded-md">
                      <Power className="w-4 h-4" />
                      <span className="font-medium">Power</span>
                    </summary>
                    <ul className="mt-1 space-y-1">
                      <li>
                        <button
                          disabled={
                            service.status === "running" ||
                            service.status === "pending"
                          }
                          className={classNames(
                            "flex items-center gap-2 w-full px-2 py-1 rounded-md text-left",
                            service.status === "running" ||
                              service.status === "pending"
                              ? "text-base-content/60 cursor-not-allowed"
                              : "text-success hover:bg-success/10 hover:text-success",
                          )}
                          onClick={() => executeAfterBlur(onStart)}
                        >
                          Start
                        </button>
                      </li>
                      <li>
                        <button
                          disabled={service.status !== "running"}
                          className={classNames(
                            "flex items-center gap-2 w-full px-2 py-1 rounded-md text-left",
                            service.status !== "running"
                              ? "text-base-content/60 cursor-not-allowed"
                              : "text-warning hover:bg-warning/10 hover:text-warning",
                          )}
                          onClick={() => executeAfterBlur(onRestart)}
                        >
                          Restart
                        </button>
                      </li>
                      <li>
                        <button
                          disabled={service.status !== "running"}
                          className={classNames(
                            "flex items-center gap-2 w-full px-2 py-1 rounded-md text-left",
                            service.status !== "running"
                              ? "text-base-content/60 cursor-not-allowed"
                              : "text-error hover:bg-error/10 hover:text-error",
                          )}
                          onClick={() => executeAfterBlur(onStop)}
                        >
                          Stop
                        </button>
                      </li>
                    </ul>
                  </details>
                </li>
                {service.status === "stopped" && (
                  <>
                    <li>
                      <button
                        onClick={() => executeAfterBlur(onUpgrade)}
                        className="flex items-center gap-2 w-full justify-start text-info hover:bg-base-200"
                      >
                        <Upload className="w-4 h-4" />
                        Upgrade
                      </button>
                    </li>
                    <li>
                      <button
                        onClick={() => executeAfterBlur(onClone)}
                        className="flex items-center gap-2 w-full justify-start text-accent hover:bg-base-200"
                      >
                        <Copy className="w-4 h-4" />
                        Clone
                      </button>
                    </li>
                    <li>
                      <button
                        onClick={() => executeAfterBlur(onBackup)}
                        className="flex items-center gap-2 w-full justify-start text-secondary hover:bg-base-200"
                      >
                        <Save className="w-4 h-4" />
                        Backup
                      </button>
                    </li>
                  </>
                )}
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
            <span className="font-medium">Version:</span>
            <span>{`${service.repository} v${service.release_version}`}</span>
          </div>
          <div className="flex justify-between">
            <span className="font-medium">Status:</span>
            <span
              className={classNames("badge badge-sm", {
                "badge-success": service.status === "running",
                "badge-warning":
                  service.status === "pending" || service.status === "idle",
                "badge-error": service.status === "failure",
                "badge-neutral": ![
                  "running",
                  "pending",
                  "idle",
                  "failure",
                ].includes(service.status),
              })}
            >
              {service.status.charAt(0).toUpperCase() + service.status.slice(1)}
            </span>
          </div>

          <div className="flex justify-between">
            <span className="font-medium">Started:</span>
            <span>{new Date(service.created).toLocaleString()}</span>
          </div>
          <div className="flex justify-between text-xs">
            <span className="font-medium">Restart Policy:</span>
            <span className="capitalize">{service.restart_policy}</span>
          </div>
        </div>
        {serviceUrls.map(serviceUrl => (
          <CopyableField key={serviceUrl} value={serviceUrl} isUrl />
        ))}
      </div>
    </div>
  );
};
