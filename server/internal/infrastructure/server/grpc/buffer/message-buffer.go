package buffer

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/shared/models"
	"fmt"
	"sync"
	"time"

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
}

func (p PublishBufferedMessage) GetGroupKey() string {
	return fmt.Sprintf("%d-%d-%s-%s", p.TenantNode.ShardID, p.TenantNode.ReplicaID, p.CF, p.CFS)
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
}

func (e EnqueueBufferedMessage) GetGroupKey() string {
	return fmt.Sprintf("%d-%d-%s-%s", e.TenantNode.ShardID, e.TenantNode.ReplicaID, e.CF, e.CFS)
}

type PublishConfirmation struct {
	QueueMessages map[string]string
	Error         error
}

type EnqueueConfirmation struct {
	MessageID string
	Error     error
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
		flushSem:      make(chan struct{}, 3), // Limit to 3 concurrent flushes
	}
	mb.timer = time.AfterFunc(mb.flushInterval, mb.flushTrigger)
	return mb
}

func (mb *MessageBuffer[T]) Stop() {
	close(mb.stopChan)
	if mb.timer != nil {
		mb.timer.Stop()
	}
	mb.flush(context.Background())
}

func (mb *MessageBuffer[T]) Add(ctx context.Context, item T) {
	mb.mu.Lock()
	mb.messages = append(mb.messages, item)
	needsFlush := len(mb.messages) >= mb.maxBufferSize
	l := len(mb.messages)
	m := mb.maxBufferSize
	mb.mu.Unlock()

	// LOG JUST ONCE for the first message
	if l == 1 {
		fmt.Printf("MessageBuffer Add: len=%d max=%d\n", l, m)
	}

	if needsFlush {
		mb.timer.Stop()
		mb.flush(ctx)
		mb.timer.Reset(mb.flushInterval)
	}
}

func (mb *MessageBuffer[T]) flushTrigger() {
	mb.flush(context.Background())
	mb.timer.Reset(mb.flushInterval)
}

func (mb *MessageBuffer[T]) flush(ctx context.Context) {
	mb.mu.Lock()
	if len(mb.messages) == 0 {
		mb.mu.Unlock()
		return
	}
	itemsToFlush := mb.messages
	mb.messages = make([]T, 0, mb.maxBufferSize)
	mb.mu.Unlock()

	go func() {
		// Acquire flush semaphore to limit concurrency
		mb.flushSem <- struct{}{}
		defer func() { <-mb.flushSem }()
		
		mb.flushFunc(ctx, itemsToFlush)
	}()
}

