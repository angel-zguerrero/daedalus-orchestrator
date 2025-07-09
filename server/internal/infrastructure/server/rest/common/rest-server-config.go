package common

import (
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type RestServerConfing struct {
	MasterNode            *dragonboat.RaftNode
	TenantNodes           []*dragonboat.RaftNode
	TenantNodesDictionary map[string]*dragonboat.RaftNode
	TenantNodesLock       sync.Mutex
	JwtKey                []byte
	JwtDuration           time.Duration
	Server                *http.Server
	Logger                zerolog.Logger
}
