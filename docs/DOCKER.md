# Docker deployment

The image builds the embedded React UI and Go binary, runs as UID/GID `10001`,
and does not generate or overwrite your configuration. No Docker socket,
privileged mode, or systemd is needed.

## First installation

1. If you do not already have `config.yml`, copy `config.yaml.example` to it.
   Keep `listen_address: 0.0.0.0`, `http_port: "7080"`, and the default relative
   data paths. Inside the image these paths resolve under `/app`.
2. Set `domain` to the hostname you will use. DNS for that domain and
   `*.your-domain` must point to the host. For local testing, follow the README's
   wildcard DNS instructions and use `http://pb.labenv.test:7080`.
3. Build, apply migrations, and start:

   ```sh
   docker compose build
   docker compose run --rm pblauncher upgrade
   docker compose up -d
   docker compose logs -f pblauncher
   ```

4. Open the configured domain on port 7080 and complete the existing admin setup.

Compose deliberately binds published ports to `127.0.0.1` by default. To expose
them on the network, set `HTTP_BIND_ADDRESS=0.0.0.0` and configure your firewall.
Do not expose the initial setup screen on an untrusted network before setup.

The config is mounted read-only and must be readable by UID 10001. New named
volumes inherit the image directories' ownership. When replacing them with bind
mounts or importing existing data, ensure that UID/GID 10001 can write those
directories. Do not change permissions on unrelated host directories.

## External TLS-terminating reverse proxy

Use the Dockerfile or Compose service with internal HTTP port **7080**. Mount
your config at `/app/config.yml`, and preserve all five data volumes from
`docker-compose.yml`. Apply migrations before starting a new installation.

Example configuration (keep the rest of `config.yaml.example`):

```yaml
domain: pb.example.com
public_url: https://pb.example.com
listen_address: 0.0.0.0
bind_address: 127.0.0.1
http_port: "7080"
https: false
disable_https_redirect: true
```

`public_url` changes the **scheme and port advertised in UI links**, including
service/custom-domain links, without changing internal listener ports or enabling
internal TLS. It must be an HTTP(S) origin matching `domain`, with no path,
credentials, query, or fragment. Omit it for direct access; nonstandard ports
such as 7080 and 8443 are never silently removed. All domains using this override
are expected to share that external scheme/port.

Configure the external proxy to:

- Forward the original `Host` header unchanged, including service subdomains.
- Route the base domain, wildcard service domains, and each custom domain to
  this container's port 7080. Configuring only the dashboard domain is not enough.
- Terminate TLS and provision certificates for **all** those names externally.
- Support WebSocket upgrades and long-lived connections as required by your apps.

Leave launcher-managed HTTPS disabled when TLS is terminated externally; do not
use the launcher's SSL request action to manage external proxy certificates.
The application does not infer the public origin from untrusted forwarding headers.

If the proxy runs in Docker, attach it and PBLauncher to the same Docker network.
It can reach the service by name on port 7080; host port publication is not needed.

## Health and persistence

The image checks `http://127.0.0.1:7080/healthz` every 30 seconds. Override the
`HEALTHCHECK_URL` environment variable if you change the internal HTTP port.
This is **launcher HTTP liveness**, not readiness of the database, release sync,
or managed instances. The route works without authentication or redirects and
returns only `ok`. Health paths on managed service/custom-domain hosts remain
untouched.

Persistent volumes contain:

| Mount | Contents |
| --- | --- |
| `/app/pb_data` | Launcher database and service logs |
| `/app/data` | Managed instances, hooks, public assets, migrations and snapshots |
| `/app/downloads` | Downloaded release binaries |
| `/app/.certificates` | Launcher-managed certificates |
| `/app/.accounts` | ACME accounts |

Back up all volumes and the configuration before updating. Configuration and
backup data may contain credentials. `docker compose down` preserves named
volumes; **do not use `down -v`** unless you intend to delete persistent data.

## Updating

After backing up and updating the source:

```sh
docker compose build
docker compose stop
docker compose run --rm pblauncher upgrade
docker compose up -d
```

Existing data directories are unchanged. Custom configuration paths must be
mounted separately if they are outside the five default persistent directories.
The image builds for the selected Docker platform; downloaded instance binaries
must match that platform. Custom releases requiring extra runtime libraries may
need a derived image.

## Resource dashboard

Open **Resources** (`/status`) after signing in. The authenticated
`GET /x-api/system/status` endpoint supplies CPU, RAM, data-filesystem usage,
system uptime, active instances, and per-instance CPU/RSS memory.

- UI refresh: 5 seconds while visible; collection is shared/cached for 2 seconds.
- Linux measurements use `/proc` and `statfs`. The first CPU sample needs a second
  sample before a percentage can be calculated. Exited/inaccessible processes
  have unavailable metrics, not fabricated zeros.
- System CPU is a percentage of the visible system's aggregate CPU time.
  Instance CPU uses **100% per logical core**, so it can exceed 100%.
- Memory is total minus `MemAvailable`; instance memory is RSS, not heap size.
- In containers, CPU, memory and uptime describe the Linux system visible via
  `/proc`, **not cgroup quotas**. Disk describes the filesystem containing
  `data_dir` (or its nearest existing parent before that directory is created).
- On non-Linux systems this first implementation reports OS, logical CPU count
  and active instances; the remaining metrics explicitly show **Not available**.
- This feature only observes processes: it adds no resource limits, autosleep,
  remote console, or changes to service lifecycle behavior.

These additions were inspired by the Docker deployment work in
[Diogo-BBDI/pb_launcher](https://github.com/Diogo-BBDI/pb_launcher) and resource
monitoring in [JDS0504/pb_launcher](https://github.com/JDS0504/pb_launcher).
