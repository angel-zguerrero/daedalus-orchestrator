# Daedalus Node.js Worker SDK 👷⚙️

Node.js/TypeScript SDK for interacting with the **Daedalus Orchestrator**.

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

To install the SDK as a dependency in your project:

```bash
npm install @omicron-x/daedalus-sdk
```

### 🏷️ Monorepo Versioning & Git Tags
While this SDK is primarily distributed and installed via the npm registry, we follow the same monorepo tagging convention as the Go SDK to track releases and allow direct Git installations:

To tag a version of this SDK (for example, `v0.1.0`):
```bash
git tag sdk/nodejs-sdk/v0.1.0
git push origin sdk/nodejs-sdk/v0.1.0
```

This allows installing the package directly from the Git repository at a specific tag:
```bash
npm install github:angel-zguerrero/daedalus-orchestrator#sdk/nodejs-sdk/v0.1.0
```

---

## 🚀 Quick Start (Usage Example)

Here is a basic example of how to connect and create a worker to process messages using the SDK:

```typescript
import { DaedalusSDK } from '@omicron-x/daedalus-sdk';

async function main() {
    // 1. Initialize the SDK
    const sdk = new DaedalusSDK({
        uri: 'http://localhost:4000', // Orchestrator gRPC/HTTP endpoint
        username: 'admin',
        password: 'admin'
    });

    // 2. Connect to the Orchestrator
    await sdk.connect();

    // 3. Register a worker to consume tasks
    await sdk.createWorker({
        workerName: 'my-custom-worker',
        intervalMs: 500,
        capacityPolicies: [
            {
                maxQueueMessages: 10,
                claimWorkFilter: {}
            }
        ],
        onMessage: async (message, ack) => {
            console.log('👷 Processing message:', message.message.content);
            
            // Your message processing logic here...

            // Acknowledge the message when finished
            await ack();
        }
    });

    console.log('✅ Worker is running. Press Ctrl+C to stop.');
}

main().catch(err => {
    console.error('💥 Fatal error:', err);
});
```

> [!NOTE]
> **Protobuf Dependency Note:** Currently, the SDK resolves gRPC protobuf definitions relative to the monorepo directory structure (looking for `server/internal/infrastructure/server/grpc/proto/definitions` three levels up from the package folder). If you use this package standalone outside the monorepo, make sure to place or symlink the protobuf definitions at the expected relative path, or install the package within the monorepo context.

---

## 🛠️ Monorepo Development & Contribution

If you are developing inside the [Daedalus Orchestrator Monorepo](file:///Users/angel/Documents/daedalus-orchestrator-project/daedalus-orchestrator), follow these steps to build and run the SDK.

### Prerequisites

To execute build tasks and run examples inside the monorepo, you must have **Nx** installed globally (or run using `npx nx`):

```bash
npm install -g nx
```

### How to Build the SDK

Compile the SDK from the project root directory:

```bash
nx run server:build-sdk-nodejs
```

Alternatively, you can compile manually within the `sdk/nodejs-sdk` folder:

```bash
cd sdk/nodejs-sdk
npm install
npm run build
```

---

## 📚 Examples Reference

We provide fully functional examples in the [examples/](file:///Users/angel/Documents/daedalus-orchestrator-project/daedalus-orchestrator/sdk/nodejs-sdk/examples) folder:

- **[Simple Worker](file:///Users/angel/Documents/daedalus-orchestrator-project/daedalus-orchestrator/sdk/nodejs-sdk/examples/simple-worker/index.ts)**: A basic worker showing connection and message consumption.
- **[Assert Resources](file:///Users/angel/Documents/daedalus-orchestrator-project/daedalus-orchestrator/sdk/nodejs-sdk/examples/assert-resources/index.ts)**: A comprehensive example demonstrating how to upsert a tenant, exchange, queue, and binding, plus publishing and enqueueing messages.

### Running the Examples

#### Option A: Running from the Monorepo Root (Using Nx)

Run the examples directly using Nx:

- **Simple Worker**:
  ```bash
  nx run server:run-nodejs-simple-worker
  ```

- **Assert Resources**:
  ```bash
  nx run server:run-nodejs-assert-resources
  ```

#### Option B: Running from the SDK Directory

Navigate to `sdk/nodejs-sdk` first, then run:

- **Simple Worker**:
  ```bash
  npm run example:simple-worker
  # or manually:
  npx ts-node examples/simple-worker/index.ts
  ```

- **Assert Resources**:
  ```bash
  npm run example:assert-resources
  # or manually:
  npx ts-node examples/assert-resources/index.ts
  ```
