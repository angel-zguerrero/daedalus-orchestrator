package buffer

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/shared/models"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	configPkg "deadalus-orch/server/internal/pkg/config"
	pbExchange "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/exchange"
	pbQueue "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/queue"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	"github.com/rs/zerolog"
)

// BufferedItem is the interface that any buffered item must implement
type BufferedItem interface {
	GetGroupKey() string // for grouping by (TenantNode+CF+CFS)
}

// PublishBufferedMessage for PublishStream
type PublishBufferedMessage struct {
	ClientMessageID string
	ExchangeCode    string
	RoutingKey      string
	VNamespace      string
	Message         models.QueueMessage
	CF              string
	CFS             string
	Tenant          *models.TenantInMaster
	TenantNode      *dragonboat.RaftNode
	ResponseChan    chan PublishConfirmation
	SendChan        chan<- *pbExchange.PublishStreamResponse
	StreamCtx       context.Context
}

func (p PublishBufferedMessage) GetGroupKey() string {
	if p.TenantNode == nil {
		return p.CF + "-" + p.CFS
	}
	return strconv.FormatUint(p.TenantNode.ShardID, 10) + "-" + strconv.FormatUint(p.TenantNode.ReplicaID, 10) + "-" + p.CF + "-" + p.CFS
}

// EnqueueBufferedMessage for EnqueueStream
type EnqueueBufferedMessage struct {
	ClientMessageID string
	QueueCode       string
	VNamespace      string
	Message         models.QueueMessage
	CF              string
	CFS             string
	Tenant          *models.TenantInMaster
	TenantNode      *dragonboat.RaftNode
	ResponseChan    chan EnqueueConfirmation
	SendChan        chan<- *pbQueue.EnqueueStreamResponse
	StreamCtx       context.Context
}

func (e EnqueueBufferedMessage) GetGroupKey() string {
	if e.TenantNode == nil {
		return e.CF + "-" + e.CFS
	}
	return strconv.FormatUint(e.TenantNode.ShardID, 10) + "-" + strconv.FormatUint(e.TenantNode.ReplicaID, 10) + "-" + e.CF + "-" + e.CFS
}

type PublishConfirmation struct {
	QueueMessages map[string]string
	Error         error
}

type EnqueueConfirmation struct {
	MessageID string
	Error     error
}

type AckBufferedMessage struct {
	LeaseID    string
	CF         string
	CFS        string
	Tenant     *models.TenantInMaster
	TenantNode *dragonboat.RaftNode
	ResponseChan chan AckConfirmation
}

func (a AckBufferedMessage) GetGroupKey() string {
	if a.TenantNode == nil {
		return a.CF + "-" + a.CFS
	}
	return strconv.FormatUint(a.TenantNode.ShardID, 10) + "-" + strconv.FormatUint(a.TenantNode.ReplicaID, 10) + "-" + a.CF + "-" + a.CFS
}

type AckConfirmation struct {
	Success             bool
	Message             string
	QueueCode           string
	VNamespace          string
	ProcessingLatencyMs float64
	QueueLatencyMs      float64
	Pending             uint64
	InProcess           uint64
	Error               error
}

type DeliveredBufferedMessage struct {
	LeaseID    string
	CF         string
	CFS        string
	TenantNode *dragonboat.RaftNode
	ResponseChan chan DeliveredConfirmation
}

func (d DeliveredBufferedMessage) GetGroupKey() string {
	if d.TenantNode == nil {
		return d.CF + "-" + d.CFS
	}
	return strconv.FormatUint(d.TenantNode.ShardID, 10) + "-" + strconv.FormatUint(d.TenantNode.ReplicaID, 10) + "-" + d.CF + "-" + d.CFS
}

type DeliveredConfirmation struct {
	Success bool
	Error   error
}

type DequeueBufferedMessage struct {
	QueueID                      string
	JobWorkerID                  string
	LeaseDuration                time.Duration
	JobWorkerCapacityPolicyIndex int
	CF                           string
	CFS                          string
	TenantNode                   *dragonboat.RaftNode
	ResponseChan                 chan DequeueConfirmation
}

func (d DequeueBufferedMessage) GetGroupKey() string {
	return fmt.Sprintf("%d-%d-%s-%s", d.TenantNode.ShardID, d.TenantNode.ReplicaID, d.CF, d.CFS)
}

type DequeueConfirmation struct {
	Result *queue_command.DequeueResult
	Error  error
}

type JobWorkerHeartbeatBufferedMessage struct {
	JobWorker    models.JobWorker
	MasterNode   *dragonboat.RaftNode
	ResponseChan chan HeartbeatConfirmation
}

func (h JobWorkerHeartbeatBufferedMessage) GetGroupKey() string {
	if h.MasterNode == nil {
		return "nil-master"
	}
	return fmt.Sprintf("%d-%d", h.MasterNode.ShardID, h.MasterNode.ReplicaID)
}

type HeartbeatConfirmation struct {
	Success bool
	Error   error
}

type MessageBuffer[T BufferedItem] struct {
	mu            sync.Mutex
	messages      []T
	flushInterval time.Duration
	maxBufferSize int
	flushFunc     func(ctx context.Context, items []T)
	logger        zerolog.Logger
	timer         *time.Timer
	stopChan      chan struct{}
	firstAddLogged uint64 // atomic counter for one-time log
	flushSem      chan struct{}
}

func NewMessageBuffer[T BufferedItem](flushInterval time.Duration, maxBufferSize int, logger zerolog.Logger, flushFunc func(ctx context.Context, items []T)) *MessageBuffer[T] {
	mb := &MessageBuffer[T]{
		messages:      make([]T, 0, maxBufferSize),
		flushInterval: flushInterval,
		maxBufferSize: maxBufferSize,
		flushFunc:     flushFunc,
		logger:        logger,
		stopChan:      make(chan struct{}),
		flushSem:      make(chan struct{}, configPkg.GlobalConfiguration.PublishBufferFlushConcurrency), // Configurable concurrency
	}
	mb.timer = time.AfterFunc(mb.flushInterval, mb.flushTrigger)
	return mb
}

func (mb *MessageBuffer[T]) Stop() {
	close(mb.stopChan)
	if mb.timer != nil {
		mb.timer.Stop()
	}
	mb.triggerFlush(context.Background())
}

func (mb *MessageBuffer[T]) Add(ctx context.Context, item T) {
	mb.mu.Lock()
	mb.messages = append(mb.messages, item)
	l := len(mb.messages)
	m := mb.maxBufferSize
	mb.mu.Unlock()

	// Log once using atomic counter
	if atomic.AddUint64(&mb.firstAddLogged, 1) == 1 {
		mb.logger.Info().Int("len", l).Int("max", m).Msg("MessageBuffer Add")
	}

	mb.triggerFlush(ctx)
}

func (mb *MessageBuffer[T]) flushTrigger() {
	mb.triggerFlush(context.Background())
}

func (mb *MessageBuffer[T]) triggerFlush(ctx context.Context) {
	select {
	case mb.flushSem <- struct{}{}:
		go func() {
			defer func() { <-mb.flushSem }()

			for {
				mb.mu.Lock()
				if len(mb.messages) == 0 {
					mb.mu.Unlock()
					return
				}
				itemsToFlush := mb.messages
				mb.messages = make([]T, 0, mb.maxBufferSize)
				mb.mu.Unlock()

				mb.flushFunc(ctx, itemsToFlush)
			}
		}()
	default:
		// An active flush worker is already running and will drain mb.messages in its loop
	}
}

