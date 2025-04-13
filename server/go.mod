module deadalus-orch/server

go 1.23.2

replace deadalus-orch/shared => ../shared

require (
	deadalus-orch/shared v0.0.0-00010101000000-000000000000
	github.com/linxGnu/grocksdb v1.9.9
	golang.org/x/crypto v0.37.0
)
