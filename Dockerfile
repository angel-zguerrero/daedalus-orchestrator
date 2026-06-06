# Stage 1: Build Angular web-admin
FROM --platform=$BUILDPLATFORM node:22-alpine AS web-builder
WORKDIR /app
COPY package*.json nx.json project.json tsconfig.base.json ./
COPY web-admin/project.json web-admin/tsconfig.json web-admin/tsconfig.app.json web-admin/tsconfig.spec.json ./web-admin/
COPY web-admin/src ./web-admin/src
RUN npm ci
RUN npx nx run daedalus-web-admin:build

# Stage 2: Build Go server
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS go-builder
WORKDIR /app
COPY server/go.mod server/go.sum ./server/
COPY shared/go.mod ./shared/
RUN cd server && go mod download
COPY shared ./shared
COPY server ./server
ARG TARGETOS TARGETARCH VERSION=dev
RUN cd server && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X deadalus-orch/server/internal/pkg/utils.DefaultEnv=production" \
    -o daedalus-orchestrator \
    ./cmd/main.go

# Stage 3: Minimal runtime image
FROM alpine:3.20
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=go-builder /app/server/daedalus-orchestrator /usr/local/bin/daedalus-orchestrator
COPY --from=web-builder /app/web-admin/dist/daedalus-web-admin/browser /usr/local/bin/web-admin

# Expose Web UI & REST API (3000), gRPC (4000), Raft / Cluster communication (5000)
EXPOSE 3000 4000 5000

# Declare persistent data volume
VOLUME ["/var/lib/daedalus/data"]

ENTRYPOINT ["daedalus-orchestrator"]
