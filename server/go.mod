module deadalus-orch/server

go 1.23.2

replace deadalus-orch/shared => ../shared

require (
	deadalus-orch/shared v0.0.0-00010101000000-000000000000
	github.com/linxGnu/grocksdb v1.9.9
	github.com/stretchr/testify v1.10.0
	golang.org/x/crypto v0.37.0
	google.golang.org/grpc v1.71.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/protobuf v1.36.4 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
