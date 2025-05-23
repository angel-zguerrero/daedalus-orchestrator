package dragonboat

import "deadalus-orch/server/internal/infrastructure/db"

type NodeRole string

type Member struct {
	IP   string
	Port int
}
type PagedResultKV struct {
	Data       []db.KeyValuePair
	NextCursor []byte
}
