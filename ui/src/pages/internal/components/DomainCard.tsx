import classNames from "classnames";
import {
  ExternalLink,
  Globe,
  Info,
  Pencil,
  ShieldCheck,
  Trash2,
} from "lucide-react";
import { useMemo, type FC } from "react";
import type { DomainDto } from "../../../services/services_domain";
import { joinUrls } from "../../../utils/url";
import { useModal } from "../../../components/modal/hook";

type Props = {
  domain: DomainDto;
  url?: string;
  port?: string;
  readonly?: boolean;
  suffix: string;
  onEdit?: () => void;
  onDelete?: () => void;
  onValidate?: () => void;
  onDnsSetup?: () => void;
};

export const DomainCard: FC<Props> = ({
  domain,
  port,
  readonly,
  url,
  suffix,
  onEdit,
  onDelete,
  onValidate,
  onDnsSetup,
}) => {
  const { openModal } = useModal();
  const fmtdomain = useMemo(() => {
    let status = domain.x_cert_request_state;
    if (status == "failed" && !domain.x_reached_max_attempt) {
      status = "pending";
    }
    return {
      name: domain.domain,
      protocol: domain.use_https === "yes" ? "https" : "http",
      status: status,
      has_valid_ssl_cert: !!domain.x_has_valid_ssl_cert,
      reached_max_attempt: !!domain.x_reached_max_attempt,
      failed_error_message: domain.x_failed_error_message,
    };
  }, [domain]);

  const strUrl = useMemo(
    () =>
      joinUrls(url ? url : `${fmtdomain.protocol}://${fmtdomain.name}`, suffix),
    [fmtdomain.name, fmtdomain.protocol, suffix, url],
  );

  const showErrorModal = (message: string) => {
    if (!message.trim()) return;

    openModal(
      <div className="p-4 w-full max-w-md sm:max-w-lg md:max-w-xl">
        <div className="alert alert-error">
          <span>{message}</span>
        </div>
      </div>,
      { title: "Certificate Generation Failed" },
    );
  };

  return (
    <div
      className={classNames(
        "rounded-xl border p-4 flex flex-col gap-2 hover:shadow-sm transition",
        "border-base-300 dark:border-base-100",
        "bg-base-100 dark:bg-base-200",
      )}
    >
      <div className="flex justify-between items-center mb-2">
        <div className="flex items-center gap-1 truncate pl-2">
          <span className="text-sm font-medium text-base-content dark:text-neutral-content truncate">
            {fmtdomain.name}
          </span>
          <a
            href={strUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
          >
            <ExternalLink className="w-4 h-4" />
          </a>
        </div>
        {fmtdomain.status && fmtdomain.protocol === "https" && (
          <span
            className={classNames("text-xs font-medium flex gap-2 ", {
              "text-warning":
                fmtdomain.status === "pending" && !fmtdomain.has_valid_ssl_cert,
              "text-success":
                fmtdomain.status === "approved" || fmtdomain.has_valid_ssl_cert,
              "text-error cursor-pointer":
                fmtdomain.status === "failed" && !fmtdomain.has_valid_ssl_cert,
            })}
            onClick={() => {
              if (fmtdomain.status === "failed") {
                showErrorModal(fmtdomain.failed_error_message ?? "");
              }
            }}
          >
            {fmtdomain.status}
            {fmtdomain.status === "failed" && <Info className="w-4 h-4" />}
          </span>
        )}
      </div>
      <div className="flex justify-between items-center text-xs text-zinc-500 dark:text-zinc-400 mt-2">
        <div className="flex items-center gap-6">
          <div className="flex">
            <span className="badge badge-ghost badge-xs">
              {fmtdomain.protocol.toUpperCase()}
            </span>
            <span className="badge badge-ghost badge-xs">Port: {port}</span>
          </div>
          {!readonly &&
            fmtdomain.reached_max_attempt &&
            fmtdomain.status !== "pending" &&
            fmtdomain.protocol === "https" &&
            !fmtdomain.has_valid_ssl_cert && (
              <button
                onClick={onValidate}
                className={classNames(
                  "btn btn-xs gap-1 border",
                  "text-zinc-700 dark:text-zinc-200",
                  "bg-white dark:bg-zinc-800",
                  "hover:bg-zinc-100 dark:hover:bg-zinc-700",
                  "border-zinc-300 dark:border-zinc-700",
                )}
              >
                <ShieldCheck className="w-3 h-3 text-inherit" />
                Retry certificate
              </button>
            )}
        </div>
        {!readonly && (
          <div className="flex gap-6">
            <button
              className={classNames(
                "btn-xs gap-1 cursor-pointer",
                "text-zinc-700 dark:text-zinc-200",
              )}
              onClick={() => onEdit?.()}
            >
              <Pencil className="w-4 h-4" />
            </button>
            <button
              className={classNames(
                "btn-xs gap-1 cursor-pointer",
                "text-zinc-700 dark:text-zinc-200",
              )}
              onClick={() => onDelete?.()}
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        )}
      </div>
      {onDnsSetup && (
        <div className="mt-2">
          <button
            type="button"
            className="btn btn-sm btn-outline gap-2"
            onClick={onDnsSetup}
          >
            <Globe className="h-4 w-4" aria-hidden="true" />
            DNS setup
          </button>
        </div>
      )}
    </div>
  );
};
