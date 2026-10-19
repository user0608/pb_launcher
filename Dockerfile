# syntax=docker/dockerfile:1
FROM node:22-bookworm-slim AS ui-builder
WORKDIR /src/ui
COPY ui/package*.json ./
RUN npm ci
COPY ui/ ./
RUN npm run build-embed

FROM golang:1.24.5-bookworm AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=ui-builder /src/ui/dist ./ui/dist
ARG COMMIT=none
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.commit=${COMMIT}" -o /out/pblauncher .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 pblauncher \
    && useradd --uid 10001 --gid 10001 --no-create-home pblauncher \
    && mkdir -p /app/pb_data /app/data /app/downloads /app/.certificates /app/.accounts \
    && chown -R pblauncher:pblauncher /app
WORKDIR /app
# PocketBase locates its launcher database next to the executable.
COPY --from=go-builder /out/pblauncher /app/pblauncher
USER 10001:10001
ENV TZ=UTC HEALTHCHECK_URL=http://127.0.0.1:7080/healthz
EXPOSE 7080 8443
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD curl --fail --silent --show-error --max-time 4 "$HEALTHCHECK_URL" || exit 1
ENTRYPOINT ["/app/pblauncher"]
CMD ["-c", "/app/config.yml"]
