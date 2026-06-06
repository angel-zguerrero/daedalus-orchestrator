# DaedalusOrchestrator 🧠⚙️

**Welcome to the brain of your distributed system.**
DaedalusOrchestrator is not your typical task runner — this is where *you* call the shots, and the system listens.

Forget throwing tasks into a black box and *hoping* the right worker catches it.
This is deterministic orchestration. Precision. Order. No chaos allowed.

---

## 🧬 What is it?

**DaedalusOrchestrator** is a distributed task orchestration system designed for ultimate control.
You get centralized scheduling with decentralized execution — perfect for multi-tenant setups where noise is not welcome.

Built on top of [Dragonboat](https://github.com/lni/dragonboat) (Raft consensus), gRPC persistent connections, and a pluggable storage layer, it gives you a rock-solid foundation for orchestrating work at scale.

---

## 🧰 Key Features

🔄 **Custom Load Balancing**
No shared queues. No random workers grabbing tasks. The orchestrator decides — every time.

🕸️ **Persistent Connections**
Long-lived gRPC or TCP. Your workers stay visible. You always know who's online.

🧩 **Cluster Aware by Design**
Nodes declare their roles at startup: `consensus`, `connector`, or `admin`. No guesswork. Everyone knows their place.

⚖️ **Resilient & Reactive**
When a node drops out? The system pauses, breathes, and rebalances gracefully.

🧭 **Consensus-based Leadership**
Consensus nodes elect a Raft leader who calls the shots. No split brains here.

💡 **Built for Multi-Tenancy**
Keep tenants in check. Avoid noisy neighbors. Deliver predictable performance.

🗄️ **PebbleDB by Default**
The primary storage engine is [PebbleDB](https://github.com/cockroachdb/pebble) — fast, embeddable, and CGO-free. RocksDB is supported as an optional alternative for advanced use cases (see below).

---

## 📦 Structure

```
├── 🖥️  server/                  # Go backend — the core of the orchestrator
│   ├── 🚀 cmd/                  # Entry point (main.go)
│   ├── 🧩 internal/             # Internal logic organized by layer
│   │   ├── ⚙️  app/             # Application coordinators / bootstrapping
│   │   ├── 🧠 domain/           # Entities, interfaces, business rules
│   │   ├── 🏗️  infrastructure/  # External integrations (DB, gRPC, REST, Dragonboat)
│   │   ├── 🧪 usecase/          # Application use cases
│   │   ├── 📚 pkg/              # Reusable internal packages (config, utils, otel…)
│   │   └── 📡 telemetry/        # OpenTelemetry setup
│   └── 🐳 Dockerfile            # Server container image
│
├── 🌐 web-admin/                # Angular admin dashboard (built and served by the server)
│
├── 🔌 sdk/
│   └── nodejs-sdk/              # Node.js SDK for connecting workers
│
└── 🔄 shared/                   # Code shared across packages
    ├── 📦 models/               # Shared DTOs and domain models
    └── 🏷️  constants/           # Shared constants (env var names, config keys…)
```

---

## 🚀 Running the Server

### Prerequisites

- **Go 1.19+**
- **Node.js / npm** (only required if building the web-admin)
- **NX** (`npm install` in the project root to install workspace tooling)
- A **C/C++ compiler** (GCC or Clang) — only required when building with RocksDB support

---

### Option 1 — Quickstart: Single node (zero config)

If no cluster flags are provided, the server bootstraps a single-node cluster automatically using sensible defaults (`127.0.0.1:5000`, replica `1`, PebbleDB).

```bash
cd server
go run cmd/main.go
```

---

### Option 2 — Run with NX (recommended for development)

These commands are the easiest way to get started. NX handles building dependencies and wiring everything together.

**Run the server only** (no admin UI):
```bash
nx run server:serve
```

**Build the admin UI and run the server** (full stack):
```bash
nx run server:serve-admin
```

---

### Option 3 — Run a local multi-node cluster (manual)

For a realistic cluster setup, start three consensus nodes (they form the Raft quorum) and then join scheduler/connector nodes.

From the `server/` directory, open separate terminals for each node:

**Consensus nodes (form the Raft quorum — start these first):**
```bash
go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 5000 \
  -initial-members=127.0.0.1:r1,127.0.0.1:r2,127.0.0.1:r3 \
  -replica 1 -rest-port 3001 -grpc-port 4001 --role=admin,consensus

go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 5000 \
  -initial-members=127.0.0.1:r1,127.0.0.1:r2,127.0.0.1:r3 \
  -replica 2 -rest-port 3002 -grpc-port 4002 --role=admin,consensus

go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 5000 \
  -initial-members=127.0.0.1:r1,127.0.0.1:r2,127.0.0.1:r3 \
  -replica 3 -rest-port 3003 -grpc-port 4003 --role=admin,consensus
```

**Scheduler/Connector nodes (join the existing cluster):**
```bash
go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 5000 \
  -replica 4 -rest-port 3004 -grpc-port 4004 -join --role=scheduler,connector

go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 5000 \
  -replica 5 -rest-port 3005 -grpc-port 4005 -join --role=scheduler,connector

go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 5000 \
  -replica 6 -rest-port 3006 -grpc-port 4006 -join --role=scheduler,connector

go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 5000 \
  -replica 7 -rest-port 3007 -grpc-port 4007 -join --role=scheduler,connector
```

---

### Option 4 — Advanced: RocksDB storage engine

> ⚠️ **Advanced use case.** PebbleDB is the default and recommended engine. Use RocksDB only if you have a specific reason.

RocksDB requires native libraries and CGO. Install it first (macOS example):

```bash
brew install rocksdb
```

Then set the CGO flags and build with the `rocksdb` tag:

```bash
export CGO_CFLAGS="-I/opt/homebrew/include"
export CGO_LDFLAGS="-L/opt/homebrew/lib -lrocksdb"
go build -tags rocksdb ./cmd/main.go
```

Or use NX (CGO flags are configured automatically):

```bash
# Run with RocksDB (server only):
nx run server:serve-rocksdb

# Run with RocksDB + web admin:
nx run server:serve-rocksdb-admin
```

---

## 🧪 Running Tests

**Default build (PebbleDB):**
```bash
cd server
LOGGER_FORMAT=pretty go test -v ./...
```

**With RocksDB** (ensure CGO flags are set and rocksdb is installed):
```bash
cd server
go test -tags rocksdb -v ./...
```

---

## ⚙️ Configuration

Configuration is loaded from three sources, in order of precedence (highest → lowest):

1. **Command-line flags** — take precedence over everything
2. **Environment variables** — override the config file
3. **Configuration file** — specified via `--config` flag or `CONFIG_PATH` env var (`key=value` format, `#` for comments)

---

## 🌐 Environment Variables

| Variable | Description | Default |
|---|---|---|
| `ENV` | Application environment. Affects log level and limits. | `development` |
| `CONFIG_PATH` | Path to the configuration file. | *(none)* |
| `DEFAULT_ROOT_USER` | Username for the default root user created on first bootstrap. | `admin` |
| `DEFAULT_ROOT_PASSWORD` | Password for the default root user. | `admin` |
| `LOGGER_FORMAT` | Log output format. Use `pretty` for dev, `json` for production. | `pretty` |
| `REPLICA_ID` | Unique numeric ID for this node in the cluster. | *(required)* |
| `ROLES` | Comma-separated roles for this node (e.g., `consensus,scheduler,connector`). | *(all roles)* |
| `SELF_MEMBER_HOST` | IP/hostname this node advertises to cluster peers. | `127.0.0.1` |
| `CLUSTER_BASE_PORT` | Base port for Raft cluster communication. | `5000` |
| `INITIAL_MEMBERS` | Comma-separated list of initial member addresses for cluster bootstrap (e.g., `127.0.0.1:r1,127.0.0.1:r2`). | *(auto-derived)* |
| `JOIN` | Set to `true` to join an existing cluster instead of bootstrapping. | `false` |
| `CONNECTOR_PORT` | Port for the connector service (worker connections). | *(optional)* |
| `TTL_INTERNAL_ERROR` | TTL in seconds for internal error entries stored in the state machine. | *(optional)* |
| `MASTER_DB_ENGINE` | Storage engine for the master database. | `pebble` |
| `TENANT_DB_ENGINE` | Storage engine for tenant databases. | `pebble` |
| `REST_LISTEN_ADDR_HOST` | Host address for the REST API server. | `0.0.0.0` |
| `REST_LISTEN_ADDR_PORT` | Port for the REST API server. | `3000` |
| `REST_API_JWT_SECRET` | JWT signing secret for the REST API. **Change this in production!** | *(insecure default)* |
| `REST_API_JWT_EXPIRATION_HOURS` | JWT token expiration time in hours. | `3` |
| `GRPC_SERVER_LISTEN_ADDR_HOST` | Host address for the gRPC server. | `0.0.0.0` |
| `GRPC_SERVER_LISTEN_ADDR_PORT` | Port for the gRPC server. | `4000` |
| `API_RAFT_TIMEOUT` | Timeout for API → Raft node requests (e.g., `5s`, `1m`). | `30s` |
| `MAX_SHARDS` | Maximum number of Raft shards. Capped by environment limits. | `10` |
| `MAX_COLUMN_FAMILIES` | Maximum number of column families per shard. | `10` |
| `NODE_SCHEDULER_HEARTBEAT_TIMEOUT` | Heartbeat timeout for scheduler nodes (e.g., `15s`, `1m`). | `15s` |
| `NODE_SCHEDULER_TTL` | TTL for node scheduler entries in minutes. Minimum 60. | `1440` (24h) |
| `NODE_SCHEDULER_BALANCING_WAIT_TIME` | Seconds to wait after last scheduler node appears before rebalancing. | `30` |
| `TENANT_SUMMARY_WORKER_INTERVAL` | Interval in seconds for the tenant summary background worker. | `30` |
| `MAX_HEADERS` | Maximum number of custom headers allowed. Range: 5–1000. | `100` |
| `DEPLOYMENT_ID` | Cluster isolation ID — useful when multiple clusters share infrastructure. | `0` |
| `MESSAGE_LEASE_DURATION` | Seconds a dequeued message is locked to a worker before lease expiry. | `30` |
| `OTEL_ACTIVED` | Enable/disable OpenTelemetry tracing (`true` or `false`). | `true` |
| `OTEL_ENDPOINT` | OpenTelemetry collector endpoint. | `localhost:4317` |
| `OTEL_TRACER_SERVICE_NAME` | Service name reported to the OpenTelemetry collector. | `deadalus-server` |

---

## 🏷️ Command-Line Flags

| Flag | Type | Description | Default |
|---|---|---|---|
| `--config` | string | Path to the configuration file. | *(none)* |
| `--self-member-host` | string | IP/hostname for cluster peer communication. | *(required for cluster)* |
| `--cluster-base-port` | int | Base port for Raft communication. | *(required for cluster)* |
| `--initial-members` | string | Comma-separated member list for new cluster bootstrap. | *(required for new cluster)* |
| `--replica` | uint64 | Unique replica ID for this node. | *(required)* |
| `--join` | bool | Join an existing cluster instead of bootstrapping. | `false` |
| `--role` | string | Comma-separated node roles (e.g., `consensus,scheduler,connector`). | *(all roles)* |
| `--rest-host` | string | REST API listen host. | `0.0.0.0` |
| `--rest-port` | int | REST API listen port. | `3000` |
| `--grpc-host` | string | gRPC server listen host. | `0.0.0.0` |
| `--grpc-port` | int | gRPC server listen port. | `4000` |
| `--master-db-engine` | string | Storage engine for master DB (`pebble` or `rocksdb`). | `pebble` |
| `--tenant-db-engine` | string | Storage engine for tenant DBs (`pebble` or `rocksdb`). | `pebble` |
| `--connector-port` | int | Port for the connector service. | *(optional)* |
| `--rest-api-jwt-secret` | string | JWT secret for the REST API. | *(insecure default)* |
| `--rest-api-jwt-expiration-hours` | int | JWT token TTL in hours. | `3` |
| `--api-raft-timeout` | duration | Timeout for API → Raft requests. | `30s` |
| `--max-shards` | int | Max number of Raft shards. | `10` |
| `--max-column-families` | int | Max column families per shard. | `10` |
| `--node-scheduler-heartbeat-timeout` | duration | Heartbeat timeout for scheduler nodes. | `15s` |
| `--node-scheduler-ttl` | int64 | TTL for scheduler entries in minutes. | `1440` |
| `--node-scheduler-balancing-wait-time` | int64 | Seconds to wait before rebalancing. | `30` |
| `--tenant-summary-worker-interval` | int64 | Tenant summary worker interval in seconds. | `30` |
| `--max-headers` | int | Max allowed custom headers. | `100` |
| `--deployment-id` | uint64 | Cluster isolation identifier. | `0` |
| `--message-lease-duration` | int64 | Message lease duration in seconds. | `30` |
| `--ttl-internal-error` | uint64 | TTL in seconds for internal error entries. | *(optional)* |
| `--help` | bool | Print full help message and exit. | — |

---

## 🗄️ Storage Engines

| Engine | Default | Requires CGO | Notes |
|---|---|---|---|
| **PebbleDB** | ✅ Yes | ❌ No | Pure Go, fast, recommended for all environments. |
| **RocksDB** | ❌ No | ✅ Yes | Optional alternative. Requires native libs and `rocksdb` build tag. |

To switch engines at runtime, set `MASTER_DB_ENGINE` and `TENANT_DB_ENGINE` (or the equivalent CLI flags). If you configure `rocksdb` without the build tag, the application will fall back to PebbleDB and log an error.

---

## 🔧 NX Commands Reference

| Command | Description |
|---|---|
| `nx run server:serve` | Run the server only (PebbleDB, no admin UI) |
| `nx run server:serve-admin` | Build the admin UI then run the server |
| `nx run server:serve-rocksdb` | Run the server with RocksDB (advanced) |
| `nx run server:serve-rocksdb-admin` | Build the admin UI then run the server with RocksDB (advanced) |
| `nx run daedalus-web-admin:build:development` | Build the admin UI only |

---

## 🧪 Status

🚧 Active development. We're building the foundation of something sharp.
Got ideas? Pull requests? War stories from other orchestrators? Hit us up.

---

## 🩺 Troubleshooting

### "failed to lock data directory"

**Error:**
```
FTL ❌ Staring node host error="failed to lock data directory" func=Run package=app
exit status 1
```

**Cause:** A previous server instance did not shut down cleanly and is still holding a lock on the data directory (e.g., `dragonboat-data-dir`).

**Solution:**

1. Find the zombie process:
   ```bash
   ps aux | grep "go run"
   # or
   lsof | grep dragonboat
   ```

2. Kill it:
   ```bash
   kill -9 <PID>
   ```

3. Retry: run the start command again.

---

## 📜 License

Licensed under Apache 2.0.

Use it in production, modify it, contribute to it, or build businesses on top of it.

See the [LICENSE](LICENSE) file for details.