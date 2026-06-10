# DaedalusOrchestrator 🧠⚙️

**Consensus-Driven, Fair-Queued, Disk-Backed Multi-Tenant Messaging.**

Control. Fairness. Isolation. No compromises.

- **🎯 Deterministic Control**: Raft-based consensus decides task assignments, not randomness
- **⚖️ Fair Queues**: Multiple priorities in one queue without starvation
- **🏝️ Noise Isolation**: Per-tenant queues mean one customer's spike doesn't starve another
- **💾 Disk-Backed**: Thousands of queues without RAM penalties
- **🚀 Flexible Workers**: One worker handles many queues via smart policies

---

## 🧬 What is it?

**DaedalusOrchestrator** is a consensus-driven, fair-queued message orchestrator for multi-tenant systems.

Unlike traditional brokers that struggle with noisy neighbors and starvation, Daedalus gives you:
- **Complete Noise Isolation**: Each tenant has separate queues per resource
- **Fair Queue Scheduling**: Priorities without starvation (threshold-based fairness)
- **Disk-Backed Scalability**: Thousands of queues on modest hardware
- **Deterministic Control**: Raft consensus, not random distribution
- **Consensus-based Leadership**: A Raft-elected leader determines all task assignments
- **Deterministic Task Routing**: Workers declare their capabilities via `ClaimWorkCapacityPolicies`, and the system matches tasks to workers based on explicit rules, not luck
- **Policy-driven Filtering**: Specify exactly which tenants, namespaces, and queues a worker can handle — down to pattern matching and exclusions

Built on top of [Dragonboat](https://github.com/lni/dragonboat) (Raft consensus), gRPC persistent connections, and a pluggable storage layer, it gives you a rock-solid foundation for orchestrating work at scale.

---

### 🏗️ True Multi-Tenancy with Noise Isolation
**The Noisy Neighbor Problem: Solved.**

Most brokers use **one queue per resource with all tenants inside** — meaning if one tenant floods the queue, it starves everyone else. **Daedalus is different**:

❌ **Traditional Broker:**
   Queue: "orders"
   ├─ Tenant A: 1000 msgs
   ├─ Tenant B: 1 msg ← Starved
   └─ Tenant C: 500 msgs

✅ **Daedalus:**
   Queue: "orders|tenant-A" → 1000 msgs
   Queue: "orders|tenant-B" → 1 msg (isolated!)
   Queue: "orders|tenant-C" → 500 msgs


**Result**: Each tenant has its own queue per resource. One tenant's traffic spike doesn't affect another's SLA. Complete noise isolation.

---

### ⚖️ Fair Queues: Beyond Priority
**Most message brokers give you priority or nothing. Daedalus gives you fairness.**

Traditional systems with multiple priorities suffer from **starvation**: higher-priority tasks consume all CPU, lower-priority tasks never run. **Daedalus implements Fair Queues with threshold-based scheduling**:

```go
// Example: Queue with 3 priority levels
Thresholds: {
    Priority 3 (highest): 4 tasks,
    Priority 2 (medium):  3 tasks,
    Priority 1 (low):     2 tasks
}

// Dequeue order:
// Cycle 1: P3×4, P2×3, P1×2
// Cycle 2: P3×4, P2×3, P1×2  (repeats fairly)
```

How it works:

- Higher priorities are always served first
- But once a priority threshold is met, the scheduler moves to the next priority level
- This prevents starvation: low-priority tasks always get a turn
- Thresholds can be 0 (drain all) or any positive number (fairness ratio).

**Why others don’t do this**: Requires persistent state on disk to track scheduling position across crashes. Daedalus has it built-in.

---

### 💾 Disk-Backed Queues by Default
**RAM doesn’t scale. Disk does.**

Most brokers require explicit configuration to persist queues to disk (and warn you it's "expensive"). **Daedalus inverts this**:

- All queues live on disk (PebbleDB or RocksDB) by default
- RAM footprint is minimal: Daedalus only caches metadata, not message data
- Minimal memory overhead per shard: MemTableSize tuned to 32KB (vs. 4MB default)

**What this means:**

- You can create thousands of queues without paying a RAM penalty
- No "queue creation limits" due to memory constraints
- Scales to massive deployments without requiring massive servers

---
### 🚀 Workers as Flexible Consumers
**You don't need 1 worker per queue. You can't afford it.**

With thousands of queues per tenant across hundreds of tenants, dedicated workers-per-queue becomes impractical. **Daedalus workers are smart consumers**:


```typescript

// One worker handles MANY queues via policies
await sdk.createWorker({
    capacityPolicies: [
        {
            maxQueueMessages: 50,
            claimWorkFilter: {
                tenantPatterns: ['prod-*'],           // Match tenant prefixes
                excludeTenantCodes: ['prod-debug'],  // But not this one
                queueCodes: ['orders', 'payments'],  // Focus on these
                vnamespaces: ['primary']             // Within this namespace
            }
        },
        {
            maxQueueMessages: 100,
            claimWorkFilter: {
                tenantCodes: ['staging-demo']        // Different policy for staging
            }
        }
    ]
});

```
**Result**: One worker can intelligently handle 100+ queues by declaring what it can handle. The orchestrator dispatches work based on the policy, not arbitrary distribution.

---

### 📊 Comparison Table

| Feature | Traditional Broker | Daedalus |
|---|---|---|
| Multi-tenant isolation | Shared queue → noisy neighbors | Per-tenant queue per resource → isolated|
| Priority handling | Starvation-prone | Fair Queues with thresholds → no starvation|
| Queue persistence | RAM-based, optional disk | Disk-backed by default, minimal RAM|
| Scalability | Limited by queue memory | Thousands of queues, same RAM footprint|
| Worker assignment | Random/round-robin | Explicit policies + consensus|

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

## 🏗️ Architecture
```
┌─────────────────────────────────────────┐
│   Consensus Layer (Raft)                │
│   ├─ Leader election                    │
│   └─ Deterministic task assignment      │
├─────────────────────────────────────────┤
│   Fair Queue System                     │
│   ├─ Threshold-based priority           │
│   ├─ Starvation prevention              │
│   └─ Stateful scheduling                │
├─────────────────────────────────────────┤
│   Disk Storage (PebbleDB/RocksDB)       │
│   ├─ Queue metadata                     │
│   ├─ Messages (on-disk, not in-memory)  │
│   └─ Fair queue state (persistent)      │
├─────────────────────────────────────────┤
│   Worker Orchestration                  │
│   ├─ Policy-based assignment            │
│   ├─ Capacity-aware distribution        │
│   └─ Flexible multi-queue handling      │
└─────────────────────────────────────────┘
```

---

## 🔌 Worker Integration (SDK)

To connect workers, register task handlers, publish/enqueue messages, and interact with the **Daedalus Orchestrator Server**, you need to use a client SDK.

Currently, we provide:
- **[Node.js / TypeScript SDK](sdk/nodejs-sdk/README.md)**: A client library built to establish persistent gRPC connections with the orchestrator, manage topologies (tenants, exchanges, queues, bindings), and process queued tasks. For installation, usage examples, and configuration guides, refer to the [Node.js SDK README](sdk/nodejs-sdk/README.md).

---

## 🚀 Running the Server

### Prerequisites

- **Go 1.19+**
- **Node.js / npm** (only required if building the web-admin)
- **NX** (`npm install` in the project root to install workspace tooling)
- A **C/C++ compiler** (GCC or Clang) — only required when building with RocksDB support

---

### Option 1 — Quickstart: Single node (zero config)

If no cluster flags are provided, the server bootstraps a single-node cluster automatically using sensible defaults (`127.0.0.1:17000`, replica `1`, PebbleDB).

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

For a realistic cluster setup, start three consensus nodes (they form the Raft quorum) and then join connector nodes.

From the `server/` directory, open separate terminals for each node:

**Consensus nodes (form the Raft quorum — start these first):**
```bash
go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 17000 \
  -initial-members=127.0.0.1:r1,127.0.0.1:r2,127.0.0.1:r3 \
  -replica 1 -rest-port 3001 -grpc-port 4001 --role=admin,consensus

go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 17000 \
  -initial-members=127.0.0.1:r1,127.0.0.1:r2,127.0.0.1:r3 \
  -replica 2 -rest-port 3002 -grpc-port 4002 --role=admin,consensus

go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 17000 \
  -initial-members=127.0.0.1:r1,127.0.0.1:r2,127.0.0.1:r3 \
  -replica 3 -rest-port 3003 -grpc-port 4003 --role=admin,consensus
```

**Connector nodes (join the existing cluster):**
```bash
go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 17000 \
  -replica 4 -rest-port 3004 -grpc-port 4004 -join --role=connector

go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 17000 \
  -replica 5 -rest-port 3005 -grpc-port 4005 -join --role=connector

go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 17000 \
  -replica 6 -rest-port 3006 -grpc-port 4006 -join --role=connector

go run cmd/main.go -self-member-host 127.0.0.1 -cluster-base-port 17000 \
  -replica 7 -rest-port 3007 -grpc-port 4007 -join --role=connector
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

### Option 5 — Run with Docker

You can run Daedalus Orchestrator using our pre-built multi-architecture Docker images.

**Quickstart (Single Node):**
```bash
docker run -d \
  -p 3000:3000 \
  -p 4000:4000 \
  -p 17000:17000 \
  --name daedalus \
  ghcr.io/angel-zguerrero/daedalus-orchestrator:latest
```

Once started, the Web UI is available at `http://localhost:3000/admin/`.

**Persistent Storage:**
By default, the container writes data to `/var/lib/daedalus/data`. Use a Docker volume to persist this data:
```bash
docker run -d \
  -p 3000:3000 \
  -p 4000:4000 \
  -p 17000:17000 \
  -v daedalus-data:/var/lib/daedalus/data \
  --name daedalus \
  ghcr.io/angel-zguerrero/daedalus-orchestrator:latest
```

You can override the data directory path inside the container using the `DAEDALUS_DATA_DIR` environment variable:
```bash
docker run -d \
  -p 3000:3000 \
  -p 4000:4000 \
  -p 17000:17000 \
  -e DAEDALUS_DATA_DIR=/custom/data/path \
  -v daedalus-data:/custom/data/path \
  --name daedalus \
  ghcr.io/angel-zguerrero/daedalus-orchestrator:latest
```

**Environment Variables Configuration:**
The container is fully configurable using environment variables. Any configuration parameter can be overridden.
```bash
docker run -d \
  -p 3000:3000 \
  -p 4000:4000 \
  -p 17000:17000 \
  -e ENV=production \
  -e DEFAULT_ROOT_USER=admin \
  -e DEFAULT_ROOT_PASSWORD=secret \
  -e REPLICA_ID=1 \
  -e DEPLOYMENT_ID=0 \
  -v daedalus-data:/var/lib/daedalus/data \
  --name daedalus \
  ghcr.io/angel-zguerrero/daedalus-orchestrator:latest
```

**Docker Compose:**
Here is a complete `docker-compose.yml` example:
```yaml
services:
  daedalus:
    image: ghcr.io/angel-zguerrero/daedalus-orchestrator:latest
    container_name: daedalus-orchestrator
    ports:
      - "3000:3000"   # Web UI & REST API
      - "4000:4000"   # gRPC Server
      - "17000:17000"   # Cluster Communication
    environment:
      - ENV=production
      - DEFAULT_ROOT_USER=admin
      - DEFAULT_ROOT_PASSWORD=secret
      - REPLICA_ID=1
      - DEPLOYMENT_ID=0
    volumes:
      - daedalus-data:/var/lib/daedalus/data

volumes:
  daedalus-data:
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
| `ROLES` | Comma-separated roles for this node (e.g., `consensus,connector,admin`). | *(all roles)* |
| `SELF_MEMBER_HOST` | IP/hostname this node advertises to cluster peers. | `127.0.0.1` |
| `CLUSTER_BASE_PORT` | Base port for Raft cluster communication. | `17000` |
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
| `--role` | string | Comma-separated node roles (e.g., `consensus,connector,admin`). | *(all roles)* |
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
