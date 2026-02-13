# DaedalusOrchestrator 🧠⚙️


# Install the necessary Angular and NX libraries

npm install

## Building the Application

The application uses Go modules for dependency management.

### Prerequisites

- Go (version 1.19 or later recommended)
- A C/C++ compiler toolchain (e.g., GCC or Clang) for Cgo support.

### Standard Build

To build the application with the default database engine (PebbleDB), run:

```bash
go build ./server/cmd/main.go
```
This will create a `main` executable in the current directory.

### Building with RocksDB Support (Optional)

DaedalusOrchestrator can use RocksDB as an alternative storage backend. To enable RocksDB support, you need to have the RocksDB library and headers installed on your system.

**Installation (macOS Example using Homebrew):**
```bash
# Ensure you are installing a compatible version (tested with 10.9.1)
brew install rocksdb
```
For other operating systems, please refer to the official RocksDB installation documentation.

**Environment Configuration:**
If you are running the application via `go run` or `go build` directly, you must set the following environment variables so CGO can find the RocksDB headers and libraries installed by Homebrew:
```bash
export CGO_CFLAGS="-I/opt/homebrew/include"
export CGO_LDFLAGS="-L/opt/homebrew/lib -lrocksdb"
```
*Note: If you use the NX commands (e.g., `nx run server:serve-rocksdb`), these variables are configured automatically.*

Then, build the application with the `rocksdb` build tag:
```bash
go build -tags rocksdb ./server/cmd/main.go
```

If you build with the `rocksdb` tag, you can then configure the application to use RocksDB via the `MASTER_DB_ENGINE` and `TENANT_DB_ENGINE` environment variables or corresponding configuration file settings (see "Configuration Sources and Precedence" below). If you attempt to configure RocksDB without building with the tag, the application will default to PebbleDB and log an error.

### Running Tests

To run tests for the default build:
```bash
cd server
go test ./...
```

To run tests including RocksDB-specific tests (ensure RocksDB is installed and CGO flags are set as above):
```bash
cd server
go test -tags rocksdb ./...
```


# Environment Variables

The application can be configured using the following environment variables. Please note that some of these might be checked by the `utils.ValidateEnvVar()` function, and others might be loaded via the `config.LoadDefaultConfiguration()` function (details of which are not fully available for this documentation update).

| Variable                 | Description                                                                                                | Possible Values                      | Default Value        |
| ------------------------ | ---------------------------------------------------------------------------------------------------------- | ------------------------------------ | -------------------- |
| `LOGGER_FORMAT`          | Defines the log output format. `pretty` is recommended for development, `json` for production environments.  | `pretty`, `json`                     | `pretty` (assumed)   |
| `ENV`           | Specifies the application environment. Affects logging levels and potentially other behaviors.             | `development`, `production`, `staging` | `development` (assumed) |
| `DEFAULT_ROOT_USER`      | The username for the default root user, created on initial bootstrap if not present.                       | string                               | `root` (example)     |
| `DEFAULT_ROOT_PASSWORD`  | The password for the default root user.                                                                    | string                               | (none, should be set)|
| `PORT`                   | The network port on which the main application services might listen (specific usage depends on config).   | integer (e.g., `8080`)               | (none, configurable) |
| `OTEL_ACTIVED`            | Enables or disables OpenTelemetry tracing.                                                                 | `true`, `false`                      | `false` (assumed)    |
| `OTEL_ENDPOINT`          | The endpoint (host:port) for the OpenTelemetry collector.                                                  | string (e.g., `localhost:4317`)      | `localhost:4317`     |
| `OTEL_TRACER_SERVICE_NAME` | The service name used for OpenTelemetry traces.                                                            | string                               | `deadalus-server`    |


# Command-Line Parameters

The application accepts command-line parameters for configuration, often related to cluster setup and node identity. These parameters are typically parsed by the `config.LoadDefaultConfiguration()` function.

| Flag                 | Type   | Description                                                                                                   | Default Value                                  | Required / Optional |
| -------------------- | ------ | ------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- | ------------------- |
| `-config`            | string | Path to a configuration file. If provided, other command-line flags might be overridden by the file's content. | (none)                                         | Optional            |
| `-self-member-host`  | string | The IP address or hostname for this node to listen on for cluster communication (e.g., `127.0.0.1`).             | (none)                                         | Required            |
| `-cluster-base-port` | int    | The base port for this node to listen on for cluster communication (e.g., `5000`).                             | `0` (must be specified)                        | Required            |
| `-initial-members`   | string | A comma-separated list of member addresses (IP:port) for bootstrapping a new cluster.                         | (none)                                         | Required for new cluster |
| `-replica`           | uint64 | The unique ID for this replica within its Raft shard.                                                         | `0` (must be specified if creating/joining)    | Required            |
| `-join`              | bool   | If `true`, the node will attempt to join an existing cluster specified by `-initial-members` (or other means). | `false`                                        | Optional            |
| `-role`              | string | Comma-separated list of roles for this node (e.g., `consensus,scheduler,connector`).                           | `consensus,scheduler,connector` (all roles)    | Optional            |
| `-connector-port`    | int    | The port on which the connector role (if active) might listen for external connections.                       | (implementation specific, e.g. `0` or a default) | Optional            |
| `-ttl-internal-error`| uint64 | Time-To-Live (in seconds) for internal error messages stored in the database by the state machine.            | (implementation specific, e.g. `3600`)         | Optional            |

**Notes on Parameters:**

*   The flag names and types are based on their definitions in `server/internal/pkg/config/loader.go`.
*   The "Cluster example" section further illustrates the usage of these flags.

# Configuration Sources and Precedence

The application loads its configuration from multiple sources. The order of precedence, from highest to lowest, is as follows:

1.  **Command-Line Flags**: These provide the highest level of override.
2.  **Environment Variables**: Values set as environment variables will override those from the configuration file.
3.  **Configuration File**: Specified by `--config` flag or `CONFIG_PATH` env var (e.g., `daedalus.conf`). This is the source with the lowest precedence.

If a setting is specified in multiple places, the value from the source with higher precedence will be used.

**Welcome to the brain of your distributed system.**  
DaedalusOrchestrator ain’t your typical task runner — this is where *you* call the shots, and the system listens.  

Forget throwing tasks into a black box and *hoping* the right worker catches it.  
This is deterministic orchestration, baby. Precision. Order. No more chaos in the dojo.

---

## 🧬 What is it?

**DaedalusOrchestrator** is a distributed task orchestration system designed for ultimate control.  
You get centralized scheduling with decentralized execution — perfect for multi-tenant setups where noise is not welcome.

It’s like if Raft, gRPC, and a Greek architect walked into a server room and built something glorious.

---

## 🧰 Key Features

🔄 **Custom Load Balancing**  
No shared queues. No rando workers grabbing tasks. The orchestrator decides — every time.

🕸️ **Persistent Connections**  
We’re talkin’ long-lived gRPC or TCP. Your workers stay visible. You always know who’s online.

🧩 **Cluster Aware by Design**  
Nodes register themselves as `main` or `follower`. No guesswork. Everyone knows their role.

⚖️ **Resilient & Reactive**  
When a follower dips out? The system pauses, breathes, and rebalances with grace.

🧭 **Consensus-based Leadership**  
Main nodes elect a leader (Raft style) who calls the shots. No split brains here.

💡 **Built for Multi-Tenancy**  
Keep tenants in check. Avoid noisy neighbors. Deliver predictable performance.


## 📦 Structure

```
├── 🖥️ server/                  # Server code (Go, gRPC, etc.)
│   ├── 🚀 cmd/                 # Main entry point
│   │   └── 🧠 main.go
│   ├── 🧩 internal/            # Internal logic and layered organization
│   │   ├── ⚙️ app/             # Use case coordinators
│   │   ├── 🧠 domain/          # Entities, interfaces, business rules
│   │   ├── 🏗️ infrastructure/  # External integrations (DB, gRPC, etc.)
│   │   ├── 🧪 usecase/         # Application use cases
│   │   └── 📚 pkg/             # Reusable internal utilities
│   ├── 🛠️ scripts/             # Deployment or maintenance scripts
│   └── 🐳 Dockerfile           # Server Dockerfile
│
├── 🧑‍💻 client/                  # Client application (Buffalo)
│
└── 🔄 shared/                  # Code shared between client and server
    └── 📦 models/              # Shared models or DTOs
```
---

## 🧪 Status

🚧 Early development. We're building the foundation of something sharp.  
Got ideas? Pull requests? War stories from other orchestrators? Hit us up.

---

## 📜 License

MIT — because control shouldn’t come with chains. 

## Cluster example
go to folder server/cmd and run:

go run . -self-member-host 127.0.0.1 -cluster-base-port 5000 -initial-members=127.0.0.1:r1,127.0.0.1:r2,127.0.0.1:r3 -replica 1 -rest-port 3001 -grpc-port 4001 --role=consensus,admin

go run . -self-member-host 127.0.0.1 -cluster-base-port 5000 -initial-members=127.0.0.1:r1,127.0.0.1:r2,127.0.0.1:r3 -replica 2  -rest-port 3002 -grpc-port 4002 --role=consensus

go run . -self-member-host 127.0.0.1 -cluster-base-port 5000 -initial-members=127.0.0.1:r1,127.0.0.1:r2,127.0.0.1:r3 -replica 3  -rest-port 3003  -grpc-port 4003 -master-db-engine pebble --role=consensus

go run . -self-member-host 127.0.0.1 -initial-members=127.0.0.1:r4 -cluster-base-port 5000 -replica 4  -rest-port 3004 -grpc-port 4004 -join true --role=scheduler


go run . -self-member-host 127.0.0.1 -initial-members=127.0.0.1:r5 -cluster-base-port 5000 -replica 5  -rest-port 3005 -grpc-port 4005 -join true --role=scheduler

## With NX:

* build web-admin and run server:
```bash
nx run server:serve-admin
```

* run only server:
```bash
nx run server:serve
```

* run server with RocksDB engine:
```bash
nx run server:serve-rocksdb
```

* build only web-admin:
```bash
nx run daedalus-web-admin:build:development
```
Tests:

LOGGER_FORMAT=pretty go test -v  ./... 

# Troubleshooting / Known Errors

## "failed to lock data directory"

**Error:**
```
FTL ❌ Staring node host error="failed to lock data directory" func=Run package=app
exit status 1
```

**Cause:**
This error occurs when a previous instance of the server did not shut down correctly and is still holding a lock on the data directory (e.g., `dragonboat-data-dir`).

**Solution:**
1.  **Find the zombie process:**
    ```bash
    ps aux | grep "go run"
    # or
    lsof | grep dragonboat
    ```
2.  **Kill the process:**
    ```bash
    kill -9 <PID>
    ```
    Replace `<PID>` with the process ID found in the previous step.
3.  **Retry:** Run the start command again. 