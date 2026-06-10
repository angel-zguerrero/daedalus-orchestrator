# Daedalus Go Worker SDK 👷⚙️

Go SDK for interacting with the **Daedalus Orchestrator**.

> [!IMPORTANT]
> This SDK acts as a worker/client to execute tasks. In order to use it, you must have the **Daedalus Orchestrator Server** running.
> 
> ### 🐳 Running the Server with Docker (Recommended)
> The easiest way to run the Daedalus server is using Docker. You can pull the official multi-architecture image from the **GitHub Container Registry (GHCR)**:
> 
> ```bash
> docker run -d \
>   -p 3000:3000 \
>   -p 4000:4000 \
>   -p 17000:17000 \
>   --name daedalus \
>   ghcr.io/angel-zguerrero/daedalus-orchestrator:latest
> ```
> Once the container is running, the gRPC connector is available at `http://localhost:4000` (which is the default port the SDK uses) and the Web Admin UI at `http://localhost:3000/admin/`.
> 
> ### 🚀 GitHub Releases & Advanced Setup
> - **Pre-built Binaries & Source Code**: You can download compiled releases from the [GitHub Releases](https://github.com/angel-zguerrero/daedalus-orchestrator/releases) page.
> - **Advanced Configuration**: For multi-node setup, custom database engines (PebbleDB/RocksDB), or detailed environment variables, check out the [Daedalus Server Running Instructions](https://github.com/angel-zguerrero/daedalus-orchestrator/blob/main/README.md#-running-the-server) in the main project README.

---

## 📦 Installation

To install the SDK as a dependency in your Go project:

```bash
go get github.com/angel-zguerrero/daedalus-orchestrator/sdk/golang-sdk
```

### 🏷️ Monorepo Versioning & Publishing
Because this SDK resides in a subdirectory within a monorepo, Go requires Git version tags to be prefixed with the relative path of the module.

To publish a version of this SDK (for example, `v0.1.0`), you must tag the repository as follows:
```bash
git tag sdk/golang-sdk/v0.1.0
git push origin sdk/golang-sdk/v0.1.0
```

Without this prefix, the Go toolchain will not associate the tag with this sub-module. If you want to fetch the latest commit from the `main` branch without a published tag, you can run:
```bash
go get github.com/angel-zguerrero/daedalus-orchestrator/sdk/golang-sdk@main
```

---

## 🚀 Quick Start (Usage Example)

Here is a basic example of how to connect and create a worker to process messages using the SDK:

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    daedalus "github.com/angel-zguerrero/daedalus-orchestrator/sdk/golang-sdk"
)

func main() {
    // 1. Initialize the SDK
    sdk := daedalus.NewDaedalusSDK(daedalus.Config{
        URI:      "http://localhost:4000", // Orchestrator gRPC/HTTP endpoint
        Username: "admin",
        Password: "admin",
    })

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 2. Connect to the Orchestrator
    if err := sdk.Connect(ctx); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer sdk.Disconnect()

    // Graceful shutdown on Ctrl+C
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigCh
        cancel()
    }()

    // 3. Register a worker to consume tasks
    err := sdk.CreateWorker(ctx, daedalus.WorkerOptions{
        WorkerName: "my-custom-worker",
        IntervalMs: 500,
        CapacityPolicies: []daedalus.ClaimWorkCapacityPolicy{
            {
                MaxQueueMessages: 10,
                ClaimWorkFilter:  &daedalus.ClaimWorkFilter{},
            },
        },
        OnMessage: func(message daedalus.ClaimedMessage, ack daedalus.AckCallback) error {
            log.Printf("👷 Processing message: %s", message.Message.Content)

            // Your message processing logic here...

            // Acknowledge the message when finished
            return ack()
        },
    })

    if err != nil && err != context.Canceled {
        log.Fatalf("Worker error: %v", err)
    }

    log.Println("✅ Worker stopped.")
}
```

> [!NOTE]
> **Protobuf Dependency Note:** The SDK includes pre-generated Go protobuf stubs under the `proto/` directory. If the `.proto` definitions in the server change, you can regenerate them with `make proto` (requires `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc`).

---

## 🛠️ Monorepo Development & Contribution

If you are developing inside the [Daedalus Orchestrator Monorepo](https://github.com/angel-zguerrero/daedalus-orchestrator), follow these steps to build and run the SDK.

### Prerequisites

- **Go 1.21+** installed
- **protoc**, **protoc-gen-go**, and **protoc-gen-go-grpc** (only for regenerating proto stubs)

Install the protoc Go plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### How to Build the SDK

Compile the SDK from the project root directory:

```bash
nx run server:build-sdk-golang
```

Alternatively, you can compile manually within the `sdk/golang-sdk` folder:

```bash
cd sdk/golang-sdk
make all
```

Or step by step:

```bash
cd sdk/golang-sdk
make proto   # Regenerate protobuf stubs
go mod tidy  # Resolve dependencies
go build ./... # Build the SDK
```

---

## 📚 Examples Reference

We provide fully functional examples in the [examples/](examples/) folder:

- **[Simple Worker](examples/simple-worker/main.go)**: A basic worker showing connection and message consumption.
- **[Assert Resources](examples/assert-resources/main.go)**: A comprehensive example demonstrating how to upsert a tenant, exchange, queue, and binding, plus publishing and enqueueing messages.

### Running the Examples

#### Option A: Running from the Monorepo Root (Using Nx)

Run the examples directly using Nx:

- **Simple Worker**:
  ```bash
  nx run server:run-golang-simple-worker
  ```

- **Assert Resources**:
  ```bash
  nx run server:run-golang-assert-resources
  ```

#### Option B: Running from the SDK Directory

Navigate to `sdk/golang-sdk` first, then run:

- **Simple Worker**:
  ```bash
  go run examples/simple-worker/main.go
  ```

- **Assert Resources**:
  ```bash
  go run examples/assert-resources/main.go
  ```

---

## 📖 API Reference

### `NewDaedalusSDK(config Config) *DaedalusSDK`
Creates a new SDK instance.

### `sdk.Connect(ctx context.Context) error`
Establishes the gRPC connection and performs initial login.

### `sdk.Disconnect() error`
Closes the gRPC connection.

### `sdk.AssertTenant(ctx, input AssertTenantInput) (*tenant.Tenant, error)`
Upserts a tenant.

### `sdk.AssertExchange(ctx, input AssertExchangeInput) (*exchange.Exchange, error)`
Upserts an exchange.

### `sdk.AssertQueue(ctx, input AssertQueueInput) (*queue.Queue, error)`
Upserts a queue.

### `sdk.AssertBinding(ctx, input AssertBindingInput) (*binding.Binding, error)`
Upserts a binding.

### `sdk.EnqueueMessage(ctx, input EnqueueMessageInput) (string, error)`
Enqueues a message directly into a queue. Returns the message ID.

### `sdk.PublishMessage(ctx, input PublishMessageInput) (map[string]string, error)`
Publishes a message through an exchange. Returns a map of `queueCode → messageID`.

### `sdk.AckMessage(ctx, leaseID, tenantCode string) error`
Acknowledges a claimed message.

### `sdk.CreateWorker(ctx, options WorkerOptions) error`
Starts a bidirectional streaming worker loop. Blocks until the context is cancelled.
