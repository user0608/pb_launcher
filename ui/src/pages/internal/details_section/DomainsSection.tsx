import { useMemo, type FC } from "react";
import { DomainCard } from "../components/DomainCard";
import { useProxyConfigs } from "../../../hooks/useProxyConfigs";
import { formatProxyUrl } from "../../../utils/url";
import { Plus } from "lucide-react";
import { useModal } from "../../../components/modal/hook";
import { DomainForm } from "../forms/DomainForm";
import {
  domainsService,
  type DomainDto,
} from "../../../services/services_domain";
import { useMutation, useQuery } from "@tanstack/react-query";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { getErrorMessage } from "../../../utils/errors";
import toast from "react-hot-toast";
import { useConfirmModal } from "../../../hooks/useConfirmModal";

type Props = {
  service_id: string;
  proxy_id: string;
  url_route_suffix: string;
};

export const DomainsSection: FC<Props> = ({
  service_id,
  proxy_id,
  url_route_suffix,
}) => {
  const confirm = useConfirmModal();
  const { openModal } = useModal();
  const proxy = useProxyConfigs();
  const queryKey = useMemo(() => {
    if (service_id !== "") return ["services", service_id, "domains"];
    if (proxy_id !== "") return ["proxy_entries", proxy_id, "domains"];
    return ["unknown"];
  }, [service_id, proxy_id]);

  const domainsQuery = useQuery({
    queryKey,
    queryFn: async () => {
      if (service_id !== "")
        return domainsService.fetchAllByServiceID(service_id!);
      if (proxy_id !== "")
        return domainsService.fetchAllDomainsByProxyID(proxy_id!);
      return [];
    },
  });

  const proxyDomain = useMemo((): DomainDto => {
    const strid = service_id !== "" ? service_id : proxy_id;
    return {
      id: "__",
      service: "",
      proxy_entry: "",
      domain: proxy.base_domain ? `${strid}.${proxy.base_domain}` : "--",
      use_https: proxy.use_https ? "yes" : "no",
    };
  }, [proxy.base_domain, proxy.use_https, service_id, proxy_id]);

  const openCreateModal = () => {
    openModal(
      <DomainForm
        service_id={service_id}
        proxy_id={proxy_id}
        onSaveRecord={() => setTimeout(domainsQuery.refetch)}
        use_https={proxy.use_https ? "yes" : "no"}
        width={360}
      />,
      {
        title: "Create Domain",
      },
    );
  };

  const openEditModal = (record: DomainDto) => {
    openModal(
      <DomainForm
        service_id={service_id}
        proxy_id={proxy_id}
        width={360}
        record={record}
        onSaveRecord={() => setTimeout(domainsQuery.refetch)}
        use_https={proxy.use_https ? "yes" : "no"}
      />,
      {
        title: "Edit Domain",
      },
    );
  };

  const deleteMutation = useMutation({
    mutationFn: domainsService.deleteDomain,
    onSuccess: () => setTimeout(() => domainsQuery.refetch()),
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleDelete = async (id: string) => {
    const ok = await confirm(
      "Delete domain",
      "Are you sure you want to delete this domain?",
    );
    if (ok) {
      deleteMutation.mutate(id);
    }
  };

  const requestSSLCertificate = useMutation({
    mutationFn: domainsService.createSSLRequest,
    onSuccess: () => setTimeout(() => domainsQuery.refetch()),
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleCreateSSLRequest = async (domain: string) =>
    requestSSLCertificate.mutate(domain);

  if (domainsQuery.isFetching) {
    return <div className="p-4">Loading...</div>;
  }

  if (domainsQuery.isError)
    return (
      <ErrorFallback
        error={domainsQuery.error}
        onRetry={() => setTimeout(domainsQuery.refetch)}
      />
    );

  return (
    <div className="space-y-6">
      <div className="flex justify-end">
        <div className="flex gap-2">
          <button
            className="btn btn-sm btn-primary gap-2 w-full sm:w-auto"
            onClick={openCreateModal}
          >
            <Plus className="w-4 h-4" />
            New instance
          </button>
        </div>
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4">
        <DomainCard
          readonly
          url={formatProxyUrl(
            proxyDomain.use_https === "yes" ? "https" : "http",
            proxyDomain.domain,
            proxy.use_https ? proxy.https_port : proxy.http_port,
            proxy.public_url,
          )}
          port={proxy.use_https ? proxy.https_port : proxy.http_port}
          domain={proxyDomain}
          suffix={url_route_suffix}
        />
        {(domainsQuery.data ?? []).map(domain => (
          <DomainCard
            key={domain.id}
            domain={domain}
            url={formatProxyUrl(
              domain.use_https === "yes" ? "https" : "http",
              domain.domain,
              domain.use_https === "yes" ? proxy.https_port : proxy.http_port,
              proxy.public_url,
            )}
            port={proxy.use_https ? proxy.https_port : proxy.http_port}
            onEdit={() => openEditModal(domain)}
            onDelete={() => handleDelete(domain.id)}
            onValidate={() => handleCreateSSLRequest(domain.domain)}
            suffix={url_route_suffix}
          />
        ))}
      </div>
    </div>
  );
};
