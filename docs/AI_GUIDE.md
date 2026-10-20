# AI Project Guide

This guide is for AI agents and contributors that need to work on PBLauncher without rediscovering the project structure from scratch.

## Product Summary

PBLauncher manages multiple local PocketBase instances from one Go binary and React UI. It downloads PocketBase releases, creates isolated service data folders, starts/stops instances, routes domains through a local proxy, manages certificates, and exposes launcher-specific APIs under `/x-api/...`.

PocketBase is both the application database and the runtime dependency being launched as managed services.

## Verification Commands

- UI lint: `cd ui && npm run lint`
- UI build for embedded assets: `cd ui && npm run build`
- Go tests: `go test ./...`
- Full verification order: run UI lint, UI build, then Go tests from repo root.

Root Go builds and tests require `ui/dist` because `ui/embed.go` uses `//go:embed all:dist`.

## Runtime Data

Runtime paths are local and usually ignored by git:

- Launcher PocketBase data: `pb_data/`
- Managed service data root: `data/<serviceID>/`
- Managed service PocketBase data: `data/<serviceID>/pb_data/`
- Managed service hooks: `data/<serviceID>/hooks/`
- Managed service public assets: `data/<serviceID>/public/`
- Managed service migrations: `data/<serviceID>/migrations/`
- Snapshot storage: `data/_snapshots/<serviceID>/`
- Downloaded PocketBase binaries/releases: `downloads/`
- Local certificates: `.certificates/`

Never treat those paths as source files. Avoid deleting them unless explicitly requested.

## Backend Architecture

- `main.go` is the Cobra entrypoint and wires the app with Uber Fx.
- PocketBase migrations live in `migrations/` and are side-effect imported by `main.go`.
- Collection names live in `collections/collections.go`.
- Custom PocketBase hooks live in `internal/hooks`.
- Custom API route modules usually expose `RegisterRoutes(app, manager)` and bind to `app.OnServe()`.
- Most custom APIs use the `/x-api/...` prefix.

Major modules:

- `internal/launcher`: service command execution, process lifecycle, start/stop/restart/upgrade.
- `internal/download`: GitHub release sync, repository records, download/ensure binary logic.
- `internal/repositorymanager`: repository sync/status HTTP endpoints.
- `internal/backup`: backup, restore, clone, snapshot APIs and file operations.
- `internal/hookmanager`: managed service PB Hooks import/export/list/read/save/delete.
- `internal/operationlog`: operation history logger.
- `internal/proxy`: reverse proxy and domain routing.
- `internal/certmanager`: certificate planning/execution.
- `internal/dnscheck`: authenticated, bounded, read-only DNS diagnostics; no HTTP/TLS probes or certificate changes.

## Frontend Architecture

- React UI lives in `ui/src`.
- Routes are registered in `ui/src/routes/AppRoutes.tsx`.
- Main authenticated shell is `ui/src/layouts/DashboardLayout.tsx`.
- API clients live in `ui/src/services/`.
- Service detail tabs live in `ui/src/pages/internal/ServiceDetailPage.tsx` and `ui/src/pages/internal/details_section/`.
- Forms live in `ui/src/pages/internal/forms/`.
- PocketBase client is `ui/src/services/client/pb.ts`.

Use TanStack Query for server state and `react-hot-toast` for user feedback. Keep UI additions aligned with DaisyUI/Tailwind patterns already used in the project.

## Service Lifecycle Rules

Service status values include:

- `idle`
- `pending`
- `running`
- `stopped`
- `failure`
- `restoring`

Important safety rules:

- Backup requires the service to be `stopped`.
- Clone requires the source service to be `stopped`.
- Upgrade requires the service to be `stopped`.
- Snapshot creation requires the service to be `stopped`.
- PB Hooks import/create/edit/delete require the service to be `stopped`.
- PB Hooks export can run while the service is running.
- Restore creates a new service instance and then queues a start command.
- Snapshot restore currently creates a new service instance using existing restore logic.

Do not bypass these constraints unless the user explicitly asks for a behavior change.

## Repository And Release Rules

- The default repository ID is `pb91u2l315h29a5` and points to `pocketbase/pocketbase`.
- The default repository must always exist.
- The default repository cannot be deleted.
- For the default repository, only `retention` is editable.
- Repository `retention` minimum is `1`.
- Upgrades must stay within the same repository/source. Do not mix PocketBase versions from different repositories.
- Release sync status fields are `last_sync_at`, `last_sync_status`, and `last_sync_error`.

## Backups And Snapshots

Backup ZIP format:

- `manifest.json`
- `data/` containing the full managed service data directory

Manifest format is `pblauncher-backup/v1`.

Restore validates that the release exists locally, metadata matches repository/version, ensures the binary is downloaded, creates a new service record, copies data, marks it stopped, and queues a start command.

Snapshots use the same ZIP structure as backups, but are stored locally under `data/_snapshots/<serviceID>/` with:

- `<snapshotID>.zip`
- `<snapshotID>.json`

Snapshot metadata is represented by `backup.SnapshotInfo`.

## PB Hooks

Managed service hooks are stored internally in `data/<serviceID>/hooks/` and shown in the UI as `PB Hooks`.

ZIP import behavior:

- The backend detects a common root from `.pb.js` files.
- Only `.pb.js` files are accepted under that detected root.
- Import replaces the full managed hooks folder.
- Import requires the service to be stopped.

The CodeMirror editor is lazy-loaded from `PBHooksSection.tsx` through `PBHookCodeEditor.tsx`.

## Operation History

The `operation_logs` collection records launcher actions. Use `internal/operationlog.Logger` instead of ad hoc logging for user-visible operations.

Typical operation names include:

- `start`
- `stop`
- `restart`
- `upgrade`
- `backup`
- `restore`
- `clone`
- `hooks_import`
- `hooks_export`
- `hooks_save`
- `hooks_delete`
- `snapshot_create`
- `snapshot_restore`
- `snapshot_delete`

Per-service history is shown in service details. Global operation history is shown at `/operations`.

## Adding A Backend Feature

Preferred minimal path:

1. Add or extend a manager/usecase in the appropriate `internal/...` package.
2. Register custom HTTP routes with `app.OnServe()` and `apis.RequireAuth()`.
3. Use `/x-api/...` for non-PocketBase-standard endpoints.
4. If schema changes are needed, add a PocketBase migration in `migrations/`.
5. Log user-visible outcomes through `internal/operationlog` when the action is operational.
6. Run `gofmt`, UI build if needed, and Go tests.

## Adding A Frontend Feature

Preferred minimal path:

1. Add API methods under `ui/src/services/`.
2. Add or extend a page/section under `ui/src/pages/internal/`.
3. Register routes in `ui/src/routes/AppRoutes.tsx` if it is a new page.
4. Add navigation in `DashboardLayout.tsx` only if the page is global.
5. Use TanStack Query for fetching and mutations.
6. Use existing `ErrorFallback`, `useModal`, `useConfirmModal`, and toast patterns.
7. Run `cd ui && npm run lint && npm run build`.

## Naming Gotchas

Some typos are intentional or historical. Do not rename them opportunistically:

- `comands`
- `ServicesComands`
- `sever.go`
- `iouitls`
- `githug`

Preserve these names unless a task explicitly requests a migration/rename.

## Security And Safety

- `config.yml` may contain secrets and is ignored. Do not commit it.
- Do not log boot user passwords or secret config values.
- Be careful with `pb_data`, `data`, `downloads`, `.certificates`, and `.accounts`; these can contain local state or sensitive data.
- Do not run destructive git commands or delete runtime folders unless explicitly requested.
