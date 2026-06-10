PROTO_SRC_DIR = ../../server/internal/infrastructure/server/grpc/proto/definitions
PROTO_OUT_DIR = proto
PROTOS = auth jobworker tenant exchange queue binding

.PHONY: proto build examples clean

## proto: Generate Go protobuf/gRPC stubs from .proto definitions
proto:
	@echo "🔧 Generating protobuf Go stubs..."
	@for proto in $(PROTOS); do \
		mkdir -p $(PROTO_OUT_DIR)/$$proto; \
		protoc --proto_path=$(PROTO_SRC_DIR) \
			--go_out=$(PROTO_OUT_DIR)/$$proto --go_opt=paths=source_relative \
			--go-grpc_out=$(PROTO_OUT_DIR)/$$proto --go-grpc_opt=paths=source_relative \
			$(PROTO_SRC_DIR)/$$proto.proto; \
	done
	@echo "✅ Proto generation complete"

## build: Build the SDK
build:
	@echo "🔨 Building SDK..."
	go build ./...
	@echo "✅ Build complete"

## examples: Build the example binaries
examples:
	@echo "🔨 Building examples..."
	go build ./examples/simple-worker/
	go build ./examples/assert-resources/
	@echo "✅ Examples built"

## clean: Clean generated proto files
clean:
	@echo "🧹 Cleaning generated files..."
	@for proto in $(PROTOS); do \
		rm -f $(PROTO_OUT_DIR)/$$proto/*.pb.go; \
	done
	@echo "✅ Clean complete"

## tidy: Run go mod tidy
tidy:
	go mod tidy

## all: Generate protos, tidy deps, and build
all: proto tidy build examples
