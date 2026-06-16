package common

import (
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/metrics"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type ServerConfing struct {
	MasterNode            *dragonboat.RaftNode
	TenantNodes           []*dragonboat.RaftNode
	TenantNodesDictionary map[string]*dragonboat.RaftNode
	TenantNodesLock       sync.Mutex
	JwtKey                []byte
	JwtDuration           time.Duration
	Server                *http.Server
	Logger                zerolog.Logger
	MetricsCollector      *metrics.MetricsCollector
}
