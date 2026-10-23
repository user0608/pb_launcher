import toast from "react-hot-toast";
import { useMemo, useState } from "react";
import { Plus, RefreshCcw, Upload } from "lucide-react";
import { useModal } from "../../components/modal/hook";
import { ServiceForm } from "./forms/ServiceForm";
import { UpgradeServiceForm } from "./forms/UpgradeServiceForm";
import { RestoreBackupForm } from "./forms/RestoreBackupForm";
import { CloneServiceForm } from "./forms/CloneServiceForm";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useLocalStorage } from "@uidotdev/usehooks";
import { serviceService, type ServiceDto } from "../../services/services";
import { ServiceCard } from "./components/ServiceCard";
import { useConfirmModal } from "../../hooks/useConfirmModal";
import { getErrorMessage } from "../../utils/errors";
import classNames from "classnames";
import { useNavigate } from "react-router-dom";
import { useProxyConfigs } from "../../hooks/useProxyConfigs";
import { backupService } from "../../services/backup";
import { ErrorFallback } from "../../components/helpers/ErrorFallback";
import {
  filterServices,
  getServiceFilterOptions,
  readStoredStatusFilter,
  serviceStatusOptions,
  type ServiceStatusFilter,
} from "../../utils/serviceFilters";

const STATUS_FILTER_KEY = "pb-dashboard-status-filter";
export const ServicesPage = () => {
  const navigate = useNavigate();
  const { openModal } = useModal();
  const confirm = useConfirmModal();

  const proxyInfo = useProxyConfigs();
  const servicesQuery = useQuery({
    queryKey: ["services"],
    queryFn: serviceService.fetchAllServices,
    refetchInterval: 3000,
  });

  const [query, setQuery] = useState("");
  const [storedStatus, setStoredStatus] = useLocalStorage<unknown>(
    STATUS_FILTER_KEY,
    { value: "all" },
  );

  const status = readStoredStatusFilter(storedStatus);
  const [repositoryID, setRepositoryID] = useState("");
  const [version, setVersion] = useState("");
  const options = useMemo(
    () => getServiceFilterOptions(servicesQuery.data ?? [], repositoryID),
    [servicesQuery.data, repositoryID],
  );
  const filtered = useMemo(
    () =>
      filterServices(servicesQuery.data ?? [], {
        query,
        status,
        repositoryID,
        version,
      }),
    [servicesQuery.data, query, status, repositoryID, version],
  );
  const hasFilters =
    query !== "" || status !== "all" || repositoryID !== "" || version !== "";
  const clearFilters = () => {
    setQuery("");
    setStoredStatus({ value: "all" });
    setRepositoryID("");
    setVersion("");
  };

  const deleteMutation = useMutation({
    mutationFn: serviceService.deleteServiceInstance,
    onSuccess: () => setTimeout(() => servicesQuery.refetch()),
    onError: error => toast.error(getErrorMessage(error)),
  });

  const backupMutation = useMutation({
    mutationFn: backupService.downloadBackup,
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleDeleteService = async (id: string) => {
    const ok = await confirm(
      "Delete service",
      "Are you sure you want to delete this service?",
    );
    if (ok) {
      deleteMutation.mutate(id);
    }
  };

  const serviceCommandMutation = useMutation({
    mutationFn: serviceService.executeServiceCommand,
    onSuccess: () => setTimeout(() => servicesQuery.refetch()),
    onError: error => toast.error(getErrorMessage(error)),
  });

  const handleStartService = async (id: string) => {
    serviceCommandMutation.mutate({ service_id: id, action: "start" });
  };

  const handleStopService = async (id: string) => {
    const ok = await confirm(
      "Stop service",
      "Are you sure you want to stop this service?",
    );
    if (ok) {
      serviceCommandMutation.mutate({ service_id: id, action: "stop" });
    }
  };

  const handleRestartService = async (id: string) => {
    const ok = await confirm(
      "Restart service",
      "Are you sure you want to restart this service?",
    );
    if (ok) {
      serviceCommandMutation.mutate({ service_id: id, action: "restart" });
    }
  };

  const handleUpgradeService = (service: ServiceDto) => {
    openModal(
      <UpgradeServiceForm
        service={service}
        onUpgrade={() => setTimeout(() => servicesQuery.refetch())}
      />,
      { title: "Upgrade Service", width: 420 },
    );
  };

  const handleBackupService = async (service: ServiceDto) => {
    backupMutation.mutate(service.id);
  };

  const handleCloneService = (service: ServiceDto) => {
    openModal(
      <CloneServiceForm
        service={service}
        onClone={() => setTimeout(() => servicesQuery.refetch())}
      />,
      { title: "Clone Service", width: 420 },
    );
  };

  const openRestoreBackupModal = () => {
    openModal(
      <RestoreBackupForm
        onRestore={() => setTimeout(() => servicesQuery.refetch())}
      />,
      { title: "Import Backup", width: 420 },
    );
  };

  const openCreateServiceModal = () => {
    openModal(
      <ServiceForm
        onSaveRecord={() => setTimeout(() => servicesQuery.refetch())}
        width={360}
      />,
      {
        title: "Create Service",
      },
    );
  };

  const openDetailsService = (service: ServiceDto) =>
    navigate(`/services/${service.id}`);

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold">Instances</h2>
          {servicesQuery.data && (
            <p className="text-sm text-base-content/70" role="status">
              {filtered.length} of {servicesQuery.data.length} instances
            </p>
          )}
        </div>

        <div className="flex flex-col md:flex-row gap-4 select-none">
          <button
            className="btn btn-sm btn-secondary gap-2 w-full sm:w-auto"
            onClick={openRestoreBackupModal}
          >
            <Upload className="w-4 h-4" />
            Import backup
          </button>
          <button
            className="btn btn-sm btn-primary gap-2 w-full sm:w-auto"
            onClick={openCreateServiceModal}
          >
            <Plus className="w-4 h-4" />
            New instance
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 items-end gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <div className="space-y-1">
          <label
            htmlFor="instance-search"
            className="block text-sm font-medium"
          >
            Search
          </label>
          <div className="flex gap-2">
            <input
              id="instance-search"
              type="search"
              placeholder="Search by name or ID…"
              className="input input-sm input-bordered w-full min-w-0"
              value={query}
              onChange={e => setQuery(e.target.value)}
            />
            <button
              type="button"
              aria-label="Refresh instances"
              title="Refresh instances"
              onClick={() => {
                void servicesQuery.refetch();
              }}
              className="btn btn-sm btn-ghost"
            >
              <RefreshCcw
                className={classNames("w-4 h-4", {
                  "animate-spin": servicesQuery.isFetching,
                })}
                aria-hidden="true"
              />
            </button>
          </div>
        </div>
        <div className="space-y-1">
          <label
            htmlFor="instance-status"
            className="block text-sm font-medium"
          >
            Status
          </label>
          <select
            id="instance-status"
            className="select select-sm select-bordered w-full"
            value={status}
            onChange={e =>
              setStoredStatus({ value: e.target.value as ServiceStatusFilter })
            }
          >
            {serviceStatusOptions.map(option => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>
        <div className="space-y-1">
          <label
            htmlFor="instance-repository"
            className="block text-sm font-medium"
          >
            Repository
          </label>
          <select
            id="instance-repository"
            className="select select-sm select-bordered w-full"
            value={repositoryID}
            onChange={e => {
              setRepositoryID(e.target.value);
              setVersion("");
            }}
          >
            <option value="">All repositories</option>
            {repositoryID &&
              !options.repositories.some(
                repository => repository.id === repositoryID,
              ) && (
                <option value={repositoryID}>
                  Selected repository (no current instances)
                </option>
              )}
            {options.repositories.map(repository => (
              <option key={repository.id} value={repository.id}>
                {repository.name || "Unnamed repository"}
              </option>
            ))}
          </select>
        </div>
        <div className="space-y-1">
          <label
            htmlFor="instance-version"
            className="block text-sm font-medium"
          >
            Version
          </label>
          <select
            id="instance-version"
            className="select select-sm select-bordered w-full"
            value={version}
            onChange={e => setVersion(e.target.value)}
          >
            <option value="">All versions</option>
            {version && !options.versions.includes(version) && (
              <option value={version}>{version} (no current instances)</option>
            )}
            {options.versions.map(value => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="flex justify-end">
        <button
          type="button"
          className="btn btn-sm btn-ghost"
          disabled={!hasFilters}
          onClick={clearFilters}
        >
          Clear filters
        </button>
      </div>

      {servicesQuery.isPending && <p role="status">Loading instances...</p>}
      {servicesQuery.isError && !servicesQuery.data && (
        <ErrorFallback
          error={servicesQuery.error}
          onRetry={() => {
            void servicesQuery.refetch();
          }}
        />
      )}
      {servicesQuery.isError && servicesQuery.data && (
        <p role="alert" className="text-error">
          Refresh failed. Showing the last successful instance list.
        </p>
      )}
      {servicesQuery.data && servicesQuery.data.length === 0 && (
        <p>
          No instances yet. Create an instance or import a backup to get
          started.
        </p>
      )}
      {servicesQuery.data &&
        servicesQuery.data.length > 0 &&
        filtered.length === 0 && (
          <p>
            No instances match your filters. Change or clear the filters to see
            more instances.
          </p>
        )}

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        {filtered.map(service => (
          <ServiceCard
            proxyInfo={proxyInfo}
            key={service.id}
            service={service}
            refreshData={() => setTimeout(() => servicesQuery.refetch())}
            onDetails={() => openDetailsService(service)}
            onDelete={() => handleDeleteService(service.id)}
            onStart={() => handleStartService(service.id)}
            onStop={() => handleStopService(service.id)}
            onRestart={() => handleRestartService(service.id)}
            onUpgrade={() => handleUpgradeService(service)}
            onBackup={() => handleBackupService(service)}
            onClone={() => handleCloneService(service)}
          />
        ))}
      </div>
    </div>
  );
};
