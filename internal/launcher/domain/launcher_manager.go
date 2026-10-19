package domain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path"
	"pb_launcher/configs"
	"pb_launcher/helpers/logstore"
	"pb_launcher/helpers/process"
	download "pb_launcher/internal/download/domain"
	"pb_launcher/internal/launcher/domain/models"
	"pb_launcher/internal/launcher/domain/repositories"
	"pb_launcher/internal/launcher/domain/services"
	"pb_launcher/internal/operationlog"
	"pb_launcher/utils/iouitls"
	"pb_launcher/utils/networktools"
	"regexp"
	"strings"
	"sync"

	version "github.com/hashicorp/go-version"
)

type LauncherManager struct {
	rwMtx               sync.RWMutex
	dataDir             string
	ipAddress           string
	installTokenUsecase *CleanServiceInstallTokenUsecase
	repository          repositories.ServiceRepository
	comandsRepository   repositories.CommandsRepository
	finder              services.BinaryFinder
	downloader          *download.DownloadUsecase
	lstore              *logstore.ServiceLogDB
	operationLogger     *operationlog.Logger
	//
	processMtx  sync.RWMutex
	processList map[string]*process.Process
	errChan     chan process.ProcessErrorMessage
}

func NewLauncherManager(
	installTokenUsecase *CleanServiceInstallTokenUsecase,
	repository repositories.ServiceRepository,
	comandsRepository repositories.CommandsRepository,
	finder services.BinaryFinder,
	downloader *download.DownloadUsecase,
	lstore *logstore.ServiceLogDB,
	operationLogger *operationlog.Logger,
	c configs.Config,
) *LauncherManager {
	lm := &LauncherManager{
		installTokenUsecase: installTokenUsecase,
		repository:          repository,
		comandsRepository:   comandsRepository,
		finder:              finder,
		downloader:          downloader,
		lstore:              lstore,
		operationLogger:     operationLogger,
		dataDir:             c.GetDataDir(),
		ipAddress:           c.GetBindIPAddress(),
		processList:         make(map[string]*process.Process),
		errChan:             make(chan process.ProcessErrorMessage, 10),
	}
	go lm.handleServiceErrors()
	return lm
}

func (lm *LauncherManager) handleServiceErrors() {
	for serviceErr := range lm.errChan {
		ctx := context.Background()
		var errorMessage string
		if serviceErr.Error != nil {
			errorMessage = serviceErr.Error.Error()
		}

		if err := lm.repository.MarkServiceFailure(ctx, serviceErr.ID, errorMessage); err != nil {
			slog.Error("failed to update service status",
				"serviceID", serviceErr.ID,
				"error", err,
				"originalError", errorMessage,
			)
			continue
		}

		service, err := lm.repository.FindService(ctx, serviceErr.ID)
		if err != nil {
			slog.Error("failed to find service",
				"serviceID", serviceErr.ID,
				"error", err,
			)
			continue
		}

		if service.RestartPolicy != models.OnFailure {
			continue
		}

		if err := lm.comandsRepository.PublishStartComand(ctx, service.ID); err != nil {
			slog.Error("failed to publish restart command",
				"serviceID", service.ID,
				"error", err,
			)
		}
	}
}

func (lm *LauncherManager) buildArgs(serviceID string) ([]string, error) {
	pb_data := path.Join(lm.dataDir, serviceID)
	return []string{
		"--dir", path.Join(pb_data, "pb_data"),
		"--hooksDir", path.Join(pb_data, "hooks"),
		"--publicDir", path.Join(pb_data, "public"),
		"--migrationsDir", path.Join(pb_data, "migrations"),
	}, nil
}

func (lm *LauncherManager) findOrDownloadBinary(ctx context.Context, service models.Service) (string, error) {
	executablePath, err := lm.finder.FindBinary(ctx, service.RepositoryID, service.Version, service.ExecFilePattern)
	if err == nil {
		return executablePath, nil
	}

	slog.Warn("binary not found locally, downloading release",
		"serviceID", service.ID,
		"releaseID", service.ReleaseID,
		"version", service.Version,
		"error", err,
	)
	lm.lstore.InsertLog(service.ID, logstore.StreamStdout, fmt.Sprintf("Binary v%s not found locally. Downloading release...", service.Version))

	if err := lm.downloader.EnsureReleaseDownloaded(ctx, service.ReleaseID); err != nil {
		lm.lstore.InsertLog(service.ID, logstore.StreamStderr, fmt.Sprintf("Failed to download binary v%s: %s", service.Version, err.Error()))
		return "", fmt.Errorf("failed to download release %s: %w", service.ReleaseID, err)
	}

	executablePath, err = lm.finder.FindBinary(ctx, service.RepositoryID, service.Version, service.ExecFilePattern)
	if err != nil {
		lm.lstore.InsertLog(service.ID, logstore.StreamStderr, fmt.Sprintf("Binary v%s downloaded, but executable was not found: %s", service.Version, err.Error()))
		return "", err
	}

	lm.lstore.InsertLog(service.ID, logstore.StreamStdout, fmt.Sprintf("Binary v%s downloaded successfully", service.Version))
	return executablePath, nil
}

// initializeBootUser sets up the initial boot user for the service instance.
func (lm *LauncherManager) UpsertSuperuser(ctx context.Context, serviceID, email, password string) error {
	service, err := lm.repository.FindService(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("failed to find service %s: %w", serviceID, err)
	}

	binaryPath, err := lm.findOrDownloadBinary(ctx, *service)
	if err != nil {
		slog.Error("failed to find binary", "serviceID", service.ID, "error", err)
		return err
	}
	baseArgs, err := lm.buildArgs(service.ID)
	if err != nil {
		slog.Error("failed to build args", "serviceID", service.ID, "error", err)
		return err
	}
	args := append(baseArgs, "superuser", "upsert", email, password)
	cmd := exec.CommandContext(ctx, binaryPath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("failed to initialize boot user",
			"service", service.ID,
			"email", service.BootUserEmail,
			"output", string(output),
			"error", err,
		)
		return err
	}
	return lm.repository.UpdateSuperuser(ctx, serviceID, email, password)
}

var pbInstallURLRegex = regexp.MustCompile(`https?://[^/]+/_/#/pbinstal/([a-zA-Z0-9._\-]+)`)

func (lm *LauncherManager) buildStdoutHandler(serviceID string) iouitls.WriterInterceptorHandler {
	pbpbinstal := []byte("/pbinstal/")
	return func(data []byte) {
		if !bytes.Contains(data, pbpbinstal) {
			return
		}
		searchData := data
		if len(data) > 2048 {
			searchData = data[:2048]
		}
		matched := pbInstallURLRegex.FindSubmatch(searchData)
		if len(matched) < 2 {
			return
		}
		token := strings.TrimSpace(string(matched[1]))
		go lm.installTokenUsecase.SetInstallToken(context.Background(), serviceID, token)
	}
}

func (lm *LauncherManager) startService(ctx context.Context, service models.Service) error {
	lm.processMtx.RLock()
	existingProcess, exists := lm.processList[service.ID]
	lm.processMtx.RUnlock()
	if exists {
		if existingProcess.IsRunning() {
			return fmt.Errorf("service %s is already running", service.ID)
		}
	}

	executablePath, err := lm.findOrDownloadBinary(ctx, service)
	if err != nil {
		slog.Error("failed to find binary", "serviceID", service.ID, "error", err)
		return err
	}
	ip, port, err := networktools.GetAvailablePort(lm.ipAddress)
	if err != nil {
		slog.Error("failed to find free port", "serviceID", service.ID, "error", err)
		return err
	}

	baseArgs, err := lm.buildArgs(service.ID)
	if err != nil {
		slog.Error("failed to build args", "serviceID", service.ID, "error", err)
		return err
	}

	listenIp := fmt.Sprintf("%s:%d", ip, port)
	serveArgs := append([]string{"serve"}, append(baseArgs, "--http", listenIp)...)

	stdout := iouitls.NewWriterInterceptor(
		lm.lstore.NewWriter(service.ID, logstore.StreamStdout),
		lm.buildStdoutHandler(service.ID),
	)

	newProcess := process.New(
		service.ID,
		executablePath,
		serveArgs,
		process.WithErrorChan(lm.errChan),
		process.WithStdout(stdout),
		process.WithStderr(lm.lstore.NewWriter(service.ID, logstore.StreamStderr)),
	)

	if err := newProcess.Start(); err != nil {
		slog.Error("failed to start process", "serviceID", service.ID, "error", err)

		stopErr := fmt.Errorf("failed to start process: %w", err)
		if err := lm.repository.MarkServiceFailure(ctx, service.ID, stopErr.Error()); err != nil {
			slog.Error("failed to mark service as failed", "serviceID", service.ID, "error", err)
		}
		return err
	}

	lm.processMtx.Lock()
	lm.processList[service.ID] = newProcess
	lm.processMtx.Unlock()

	if err := lm.repository.MarkServiceRunning(ctx, service.ID, ip, fmt.Sprint(port)); err != nil {
		slog.Error("failed to update service status to running",
			"serviceID", service.ID,
			"ip", ip,
			"port", port,
			"error", err,
		)
	}

	return err
}

func (lm *LauncherManager) stopService(ctx context.Context, serviceID string) error {
	lm.processMtx.RLock()
	existingProcess, exists := lm.processList[serviceID]
	lm.processMtx.RUnlock()
	if !exists {
		return fmt.Errorf("no running process found for service %s", serviceID)
	}
	if !existingProcess.IsRunning() {
		return fmt.Errorf("process for service %s is not currently running", serviceID)
	}

	if err := existingProcess.Stop(); err != nil {
		slog.Error("failed to stop existing process", "serviceID", serviceID, "error", err)
		return err
	}

	lm.processMtx.Lock()
	delete(lm.processList, serviceID)
	lm.processMtx.Unlock()

	if err := lm.repository.MarkServiceStoped(ctx, serviceID); err != nil {
		slog.Error("failed to mark service as stopped", "serviceID", serviceID, "error", err)
	}
	return nil
}

func (lm *LauncherManager) restartService(ctx context.Context, service models.Service) error {
	lm.lstore.InsertLog(service.ID, logstore.StreamStdout, "Restarting service...")

	lm.processMtx.RLock()
	p, ok := lm.processList[service.ID]
	lm.processMtx.RUnlock()
	if ok && p.IsRunning() {
		if err := lm.stopService(ctx, service.ID); err != nil {
			slog.Error("restart failed: unable to stop service", "serviceID", service.ID, "error", err)
			return err
		}
	}
	if err := lm.startService(ctx, service); err != nil {
		slog.Error("restart failed: unable to start service", "serviceID", service.ID, "error", err)
		return err
	}
	lm.lstore.InsertLog(service.ID, logstore.StreamStdout, "Service restarted successfully")
	return nil
}

func (lm *LauncherManager) upgradeService(ctx context.Context, service models.Service, targetReleaseID string) error {
	if targetReleaseID == "" {
		return fmt.Errorf("target release is required for service upgrade")
	}
	if service.Status != models.Stopped {
		return fmt.Errorf("service %s must be stopped before upgrade", service.ID)
	}

	targetRelease, err := lm.repository.FindRelease(ctx, targetReleaseID)
	if err != nil {
		return fmt.Errorf("failed to find target release %s: %w", targetReleaseID, err)
	}
	if targetRelease.RepositoryID != service.RepositoryID {
		return fmt.Errorf("target release belongs to a different repository")
	}

	currentVersion, err := version.NewVersion(service.Version)
	if err != nil {
		return fmt.Errorf("invalid current service version %q: %w", service.Version, err)
	}
	targetVersion, err := version.NewVersion(targetRelease.Version)
	if err != nil {
		return fmt.Errorf("invalid target release version %q: %w", targetRelease.Version, err)
	}
	if !targetVersion.GreaterThan(currentVersion) {
		return fmt.Errorf("target version %s must be greater than current version %s", targetVersion.String(), currentVersion.String())
	}

	validationService := service
	validationService.ReleaseID = targetRelease.ID
	validationService.Version = targetRelease.Version
	if _, err := lm.findOrDownloadBinary(ctx, validationService); err != nil {
		return fmt.Errorf("target version binary not found: %w", err)
	}

	if err := lm.repository.UpdateServiceRelease(ctx, service.ID, targetRelease.ID); err != nil {
		return fmt.Errorf("failed to update service release: %w", err)
	}

	upgradedService := service
	upgradedService.ReleaseID = targetRelease.ID
	upgradedService.Version = targetRelease.Version
	lm.lstore.InsertLog(service.ID, logstore.StreamStdout, fmt.Sprintf("Upgrading service from v%s to v%s...", service.Version, targetRelease.Version))
	if err := lm.startService(ctx, upgradedService); err != nil {
		slog.Error("upgrade failed: unable to start service", "serviceID", service.ID, "error", err)
		return err
	}
	lm.lstore.InsertLog(service.ID, logstore.StreamStdout, "Service upgraded successfully")
	return nil
}

// RecoveryLastState restores and starts all services that were active
// before pb_launcher was shut down.
func (lm *LauncherManager) RecoveryLastState(ctx context.Context) error {
	lm.rwMtx.Lock()
	defer lm.rwMtx.Unlock()
	services, err := lm.repository.RunningServices(ctx)
	if err != nil {
		slog.Error("Failed to retrieve running services", "error", err)
		return err
	}

	for _, service := range services {
		if service.Deleted != "" {
			continue
		}
		if err := lm.startService(ctx, service); err != nil {
			slog.Error("failed to start service",
				"serviceID", service.ID,
				"error", err,
			)

			continue
		}
	}

	return nil
}

func (lm *LauncherManager) evaluateCommand(ctx context.Context, cmd models.ServiceCommand) error {
	service, err := lm.repository.FindService(ctx, cmd.Service)
	if err != nil {
		return fmt.Errorf("failed to find service %s: %w", cmd.Service, err)
	}

	switch cmd.Action {
	case models.ActionStart:
		return lm.startService(ctx, *service)
	case models.ActionStop:
		return lm.stopService(ctx, service.ID)
	case models.ActionRestart:
		return lm.restartService(ctx, *service)
	case models.ActionUpgrade:
		return lm.upgradeService(ctx, *service, cmd.TargetRelease)
	default:
		return fmt.Errorf("unknown action %q for service %s", cmd.Action, cmd.Service)
	}
}

func (lm *LauncherManager) Run(ctx context.Context) error {
	comands, err := lm.comandsRepository.GetPendingCommands(ctx)
	if err != nil {
		slog.Error("failed to get pending commands", "error", err)
		return err
	}
	for _, c := range comands {
		if err := lm.evaluateCommand(ctx, c); err != nil {
			lm.operationLogger.Error(ctx, c.Service, c.Action.String(), err.Error(), map[string]any{"command_id": c.ID})
			if markErr := lm.comandsRepository.MarkCommandError(ctx, c.ID, err.Error()); markErr != nil {
				slog.Error("failed to mark command as error", "commandID", c.ID, "error", markErr)
			}
			continue
		}
		lm.operationLogger.Success(ctx, c.Service, c.Action.String(), "command executed successfully", map[string]any{"command_id": c.ID})
		if err := lm.comandsRepository.MarkCommandSuccess(ctx, c.ID); err != nil {
			slog.Error("failed to mark command as success", "commandID", c.ID, "error", err)
		}
	}
	return nil
}

// RunningProcessIDs returns a detached snapshot; callers never access the map
// owned by the launcher or hold its lock while collecting OS metrics.
func (lm *LauncherManager) RunningProcessIDs() map[string]int {
	lm.processMtx.RLock()
	defer lm.processMtx.RUnlock()
	pids := make(map[string]int)
	for id, proc := range lm.processList {
		if pid := proc.PID(); pid > 0 {
			pids[id] = pid
		}
	}
	return pids
}

func (lm *LauncherManager) Dispose() error {
	lm.rwMtx.Lock()
	defer lm.rwMtx.Unlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var combinedErr error

	collectError := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		combinedErr = errors.Join(combinedErr, err)
	}

	lm.processMtx.RLock()
	processes := make([]*process.Process, 0, len(lm.processList))
	for _, proc := range lm.processList {
		processes = append(processes, proc)
	}
	lm.processMtx.RUnlock()
	for _, proc := range processes {
		wg.Add(1)
		go func(p *process.Process) {
			defer wg.Done()
			if !p.IsRunning() {
				return
			}
			if err := p.Stop(); err != nil {
				collectError(err)
			}
		}(proc)
	}

	wg.Wait()
	close(lm.errChan)
	return combinedErr
}
