# Dependency security review — 2026-10-19

## Scope and outcome

Reviewed the embedded UI (`ui/package.json` and its lockfile), the launcher Go
module, and the Go build image. This is not a full source-code security audit;
the independent `web_site` dependency tree is outside this review.

**The UI audit is clean. The Go audit is not yet clean.** Do not interpret passing
tests or the UI's zero-alert result as security clearance for the entire launcher.
Audit results are time-sensitive and do not establish absence of vulnerabilities.

## Embedded UI

`npm audit` originally reported 27 affected packages: 1 critical, 22 high,
2 moderate, and 2 low. After compatible updates, both the complete audit and
`npm audit --omit=dev` report **zero known vulnerabilities**.

Direct dependencies with changed versions:

| Package | Before | After |
| --- | --- | --- |
| `react-router-dom` / `react-router` | 7.6.1 | 7.18.4 |
| `vite` | 6.3.5 | 6.4.4 |
| `typescript-eslint` | 8.33.0 | 8.72.0 |

Minimum direct dependency ranges now require these fixed versions. The lockfile
also updates affected transitive packages, including `tar` (7.4.3 → 7.5.22),
`postcss` (8.5.4 → 8.5.31), and `rollup` (4.41.1 → 4.64.10).
No `--force`, major-version upgrade, or override was needed.

### Exposure notes

- The critical `tar` advisory arrived through `@tailwindcss/vite` →
  `@tailwindcss/oxide` → `tar`. This is build-time tooling; it is not a tar
  extractor exposed by the embedded browser application. It still matters for
  development/build environments and was updated.
- Vite's dev-server file disclosure issues concern development/preview servers,
  not static assets served by the Go binary. Do not expose a dev server publicly.
- React Router is shipped in the browser bundle, but this project uses
  `BrowserRouter`/`Routes`, not React Router's SSR/framework server endpoints.
  Advisories involving those server endpoints are not evidence that this SPA
  exposes them. Client-side link/redirect fixes are nevertheless included.
- ESLint/parser and glob-related findings primarily affected lint/build tooling.
  npm's `dependencies`/`devDependencies` classification is not the same as the
  code actually shipped to the browser (Tailwind tooling was in `dependencies`).

## Go changes without a framework migration

| Component | Before | After |
| --- | --- | --- |
| Recommended toolchain and Docker builder | Go 1.24.5 | Go 1.24.13 |
| `github.com/go-acme/lego/v4` | 4.25.1 | 4.25.2 |
| `github.com/go-jose/go-jose/v4` | 4.1.1 | 4.1.4 |

The Go language directive remains `go 1.24.0`. The dependency patches address
[GO-2025-3847](https://pkg.go.dev/vuln/GO-2025-3847) (ACME HTTP handling) and
[GO-2026-4945](https://pkg.go.dev/vuln/GO-2026-4945) (JOSE panic).
The toolchain patch removes 16 symbol-level standard-library findings compared
with the same dependency graph analyzed under Go 1.24.5.

### Remaining backend findings

`govulncheck` under **Go 1.24.13**, after these changes, reports **46 distinct
advisories at symbol level**. Counts below overlap where one advisory affects
both the standard library and an external module; they must not be summed.

| Component | Symbol-level advisories | Remediation constraint |
| --- | ---: | --- |
| Go standard library | 29 | Go 1.24 is no longer a supported security line; migrate to a supported patched toolchain |
| `golang.org/x/net` | 8 | Full fixes require newer releases; e.g. 0.60.0 requires Go 1.26 |
| `golang.org/x/image` | 8 | Current listed fixes include 0.45.0, requiring Go 1.25 |
| `golang.org/x/text` | 1 | Fix is in 0.39.0, requiring Go 1.25 |
| PocketBase | 1 | [GO-2026-6489](https://pkg.go.dev/vuln/GO-2026-6489), fixed in 0.39.7, requiring Go 1.25 |

Symbol-level reachability is a static analysis result, not a verified exploit
against this deployment. Some paths are conservative interface-dispatch results
or platform-specific. Nevertheless these are unresolved findings, not ignored
or suppressed alerts.

Next security work should migrate the launcher to a supported patched Go line
(currently 1.26.9 or 1.27.2), update the affected `golang.org/x` modules, and
separately validate a patched PocketBase release against hooks, authentication,
database migrations and existing data backups. Do not silently jump from
PocketBase 0.29 to 0.39+ or apply migrations to production data during this review.
The managed instances' PocketBase versions are separate from the launcher module
and have not been changed.

## Reproducing checks

Run from `ui/`:

```sh
npm ci
npm audit
npm audit --omit=dev
npm run lint
npm test
npm run build
```

Then, from the repository root:

```sh
GOTOOLCHAIN=go1.24.13 go mod verify
GOTOOLCHAIN=go1.24.13 go test ./...
GOTOOLCHAIN=go1.24.13 go vet ./...
GOTOOLCHAIN=go1.24.13 govulncheck ./...
```

`govulncheck` is an external audit tool, not a new application dependency.
Its text-mode command is expected to fail while the backend findings remain.
Set `GOTOOLCHAIN` explicitly for reproducible audits: the `toolchain` directive
selects a recommended minimum, not an exact pin when a newer Go is installed.
