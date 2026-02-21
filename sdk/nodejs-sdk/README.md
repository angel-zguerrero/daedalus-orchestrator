# Daedalus Node.js Worker SDK

Node.js/TypeScript SDK for interacting with the Daedalus Orchestrator.

## Installation

```bash
cd sdk/nodejs-sdk
npm install
```

## How to Build

You can compile the SDK using Nx from the project root:

```bash
nx run server:build-sdk-nodejs
```

Or manually within the `sdk/nodejs-sdk` folder:

```bash
npm run build
```

## Examples

### Simple Worker

To run the simple worker example:

```bash
nx run server:run-nodejs-simple-worker
```

Or manually:

```bash
cd sdk/nodejs-sdk
npx ts-node examples/simple-worker/index.ts
```
