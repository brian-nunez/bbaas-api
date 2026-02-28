FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags='-s -w' -o /out/bbaas-api ./cmd/main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/* \
  && groupadd --gid 10001 app \
  && useradd --uid 10001 --gid app --shell /usr/sbin/nologin --create-home app

WORKDIR /app

COPY --from=builder /out/bbaas-api /usr/local/bin/bbaas-api
COPY --from=builder /src/assets ./assets

RUN mkdir -p /data && chown -R app:app /app /data

ENV PORT=8080
ENV DB_DRIVER=sqlite
ENV DB_DSN=file:/data/bbaas.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)
ENV CDP_MANAGER_BASE_URL=http://127.0.0.1:8081
ENV CDP_MANAGER_HTTP_TIMEOUT=60s
ENV CDP_PUBLIC_BASE_URL=

EXPOSE 8080

USER app

ENTRYPOINT ["/usr/local/bin/bbaas-api"]
