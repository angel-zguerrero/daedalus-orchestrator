# DaedalusOrchestrator 🧠⚙️

before execute the example execute this command line 

export CGO_CFLAGS="-I/opt/homebrew/include"                
export CGO_LDFLAGS="-L/opt/homebrew/lib -lrocksdb"

# Env Vars

```
LOGGER_FORMAT= pretty | json 
ENV= development | production | staging
DB_NAME = dabase_name
DEFAULT_ROOT_USER = an username
DEFAULT_ROOT_PASSWORD = a password 
PORT = a port
OTEL_ACTIVE = "true" | "false"
OTEL_ENDPOINT =  url:port // default localhost:4317
OTEL_TRACER_SERVICE_NAME = "otl tracer service name" // default deadalus-server
```

# Parameters

config = path/configuration.conf

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

go run . -self-member-addr 127.0.0.1:5000 -initial-members=127.0.0.1:5000,127.0.0.1:5001,127.0.0.1:5002 -replica 1
go run . -self-member-addr 127.0.0.1:5001 -initial-members=127.0.0.1:5000,127.0.0.1:5001,127.0.0.1:5002 -replica 2
go run . -self-member-addr 127.0.0.1:5002 -initial-members=127.0.0.1:5000,127.0.0.1:5001,127.0.0.1:5002 -replica 3 -role consensus

go run . -self-member-addr 127.0.0.1:5003 -join true -replica 4 -role connector