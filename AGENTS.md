# AGENTS.md

## Commands
- Go toolchain is pinned by `go.mod` to Go `1.24.13` (`go 1.24.0`).
- Run the app with `make run`; it executes `env TZ=UTC go run *.go -c config.yml`.
- Generate a local config with `make gen-config > config.yml`; `config.yml` is ignored and may contain secrets.
- Build embedded UI before any root Go build/test from a clean tree: `cd ui && npm run build` or `make build-ui`. The `ui` package uses `//go:embed all:dist`, so `go test ./...` fails if `ui/dist` is missing.
- Full verification order: `cd ui && npm run lint && npm run build`, then from repo root `go test ./...`.
- Focused Go tests: `go test ./path/to/package -run TestName`.
- UI scripts run from `ui/`: `npm run dev`, `npm run lint`, `npm run fmt`, `npm run build`, `npm run build-embed`.
- Release build: `make build`; it runs `build-ui`, builds `build/pblauncher`, and zips using the `version` file.

## Architecture
- `main.go` is the Cobra entrypoint; normal execution builds an Uber Fx app and invokes the API server, embedded UI, certificate jobs, release sync, launcher runner, proxy, and serial executor.
- PocketBase is the backing app/database. Local runtime data is under ignored dirs such as `pb_data`, `data`, `downloads`, `.certificates`, and `.accounts`.
- Major Fx modules live under `internal/download`, `internal/launcher`, `internal/proxy`, `internal/certmanager`, and `internal/hooks`.
- Custom PocketBase routes and collection hooks are registered in `internal/hooks`; custom API endpoints use the `/x-api/...` prefix.
- The React UI lives in `ui/src`, talks to PocketBase through `ui/src/services/client/pb.ts`, and reads `VITE_API_URL` with `/` as fallback.
- Embedded UI builds use `ui/.env.embed` (`VITE_API_URL=/`) and are served by `ui.go` as a catch-all SPA route from `ui/dist`.
- For deeper AI/contributor context, read `docs/AI_GUIDE.md`. When handing work to another model, use `docs/AI_HANDOFF_TEMPLATE.md`.

## Config And Local Runtime
- `config.yaml.example` is embedded into the binary and is the source for `gen-config`.
- Default local domain is `pb.labenv.test`; the README documents the CoreDNS wildcard setup needed for local proxy/domain testing.
- If `https: true`, `acme_email` must be a valid email; cert provider defaults to `selfsigned` when `cert.provider` is empty.
- Config durations are clamped to minimums in `configs/configs.go` (for example release sync `5m`, command check `10s`, cert check `1m`).

## Migrations
- PocketBase migrations live in `migrations/` and are imported by `main.go` with `_ "pb_launcher/migrations"`.
- Create migrations with `make new-migrate name_here`; it writes `migrations/<unix>_<snake_name>.go`.
- Apply/revert migrations with `make upgrade` and `make downgrade`; these run the app commands against the default PocketBase data location.

## Gotchas
- Do not delete or ignore `ui/dist` during Go verification unless you rebuild it before running root `go test ./...` or `go build`.
- Some package/file names intentionally contain typos (`comand`, `sever`, `iouitls`, `githug`); match existing names instead of renaming opportunistically.
- `helpers/process` tests spawn `sleep` and include a force-kill timeout path, so that package can take around 10-12 seconds.
