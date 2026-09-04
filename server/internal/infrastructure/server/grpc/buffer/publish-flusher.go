package buffer

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/exchange"
	"deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type routingCacheEntry struct {
	queues    []models.Queue
	expiresAt time.Time
}

type RoutingCache struct {
	mu      sync.RWMutex
	entries map[string]routingCacheEntry
	ttl     time.Duration
}

func NewRoutingCache(ttl time.Duration) *RoutingCache {
	return &RoutingCache{
		entries: make(map[string]routingCacheEntry),
		ttl:     ttl,
	}
}

func (rc *RoutingCache) Get(key string) ([]models.Queue, bool) {
	rc.mu.RLock()
	entry, ok := rc.entries[key]
	rc.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		rc.mu.Lock()
		delete(rc.entries, key)
		rc.mu.Unlock()
		return nil, false
	}
	return entry.queues, true
}

func (rc *RoutingCache) Set(key string, queues []models.Queue) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.entries[key] = routingCacheEntry{
		queues:    queues,
		expiresAt: time.Now().Add(rc.ttl),
	}

	if len(rc.entries) > 500 {
		now := time.Now()
		for k, e := range rc.entries {
			if now.After(e.expiresAt) {
				delete(rc.entries, k)
			}
		}
	}
}

func NewPublishFlusher(exchangeBO *bo.ExchangeBO, logger zerolog.Logger, raftTimeout time.Duration) func(ctx context.Context, items []PublishBufferedMessage) {
	globalRoutingCache := NewRoutingCache(30 * time.Second)

	return func(ctx context.Context, items []PublishBufferedMessage) {
		if len(items) == 0 {
			return
		}

		// Group by TenantNode + CF + CFS (Raft partition)
		groups := make(map[string][]PublishBufferedMessage)
		for _, item := range items {
			key := item.GetGroupKey()
			groups[key] = append(groups[key], item)
		}

		for _, group := range groups {
			processPublishGroup(ctx, group, exchangeBO, logger, raftTimeout, globalRoutingCache)
		}
	}
}

func processPublishGroup(ctx context.Context, items []PublishBufferedMessage, exchangeBO *bo.ExchangeBO, logger zerolog.Logger, raftTimeout time.Duration, routingCache *RoutingCache) {
	if len(items) == 0 {
		return
	}
	
	logger.Debug().Int("items_count", len(items)).Msg("processPublishGroup started")

	// For all items in the group, we assume they share the same TenantNode, CF, CFS, and Tenant Code.
	// We use the first item to get the common properties.
	tenantNode := items[0].TenantNode
	cf := items[0].CF
	cfs := items[0].CFS

	var allMessages []models.QueueMessage
	// Maps original index in 'items' to the indices of its resolved messages in 'allMessages'
	itemToMessageIndices := make(map[int][]int)
	// Maps original index in 'items' to any error encountered during routing
	itemErrors := make(map[int]error)

	msgIDToQueueCode := make(map[string]string)
	publishedCounts := make(map[string]int)

	for i, item := range items {
		var cacheKey string
		if len(item.Message.Headers) == 0 {
			cacheKey = cf + "|" + cfs + "|" + item.ExchangeCode + "|" + item.RoutingKey + "|" + item.VNamespace
		} else {
			var headerStr strings.Builder
			keys := make([]string, 0, len(item.Message.Headers))
			for k := range item.Message.Headers {
				keys = append(keys, k)
			}
			if len(keys) > 1 {
				sort.Strings(keys)
			}
			for _, k := range keys {
				headerStr.WriteString(k)
				headerStr.WriteString("=")
				headerStr.WriteString(item.Message.Headers[k])
				headerStr.WriteString(";")
			}
			cacheKey = cf + "|" + cfs + "|" + item.ExchangeCode + "|" + item.RoutingKey + "|" + item.VNamespace + "|" + headerStr.String()
		}

		var queuesList []models.Queue
		var err error

		if cachedQueues, ok := routingCache.Get(cacheKey); ok {
			queuesList = cachedQueues
		} else {
			// Resolve the exchange routing
			queuesList, err = exchangeBO.GetQueuesFromExchange(ctx, item.ExchangeCode, item.RoutingKey, item.Message, item.VNamespace, cf, cfs, item.Tenant, tenantNode)
			if err == nil && len(queuesList) > 0 {
				routingCache.Set(cacheKey, queuesList)
			}
		}

		if err != nil {
			itemErrors[i] = fmt.Errorf("failed to get queues from exchange: %w", err)
			continue
		}

		if len(queuesList) == 0 {
			itemErrors[i] = fmt.Errorf("no queues bound to exchange %s with routing key %s", item.ExchangeCode, item.RoutingKey)
			continue
		}

		var indices []int
		for _, q := range queuesList {
			msg := item.Message // Copy
			msg.QueueID = q.ID
			u := uuid.New()
			msg.ID = hex.EncodeToString(u[:])
			if msg.MessageID == "" {
				msg.MessageID = msg.ID
			}

			msgIDToQueueCode[msg.ID] = q.Code
			publishedCounts[q.Code+"|"+q.VNamespace]++

			indices = append(indices, len(allMessages))
			allMessages = append(allMessages, msg)
		}
		itemToMessageIndices[i] = indices
	}

	// If there are no messages to enqueue, notify any items that had errors
	if len(allMessages) == 0 {
		for i, item := range items {
			if err, ok := itemErrors[i]; ok {
				notifyPublishError(item, err)
			} else {
				// Should not happen, but just in case
				notifyPublishError(item, fmt.Errorf("unknown routing error"))
			}
		}
		return
	}

	const MaxMicroBatchSize = 250

	var combinedRes queue.EnqueueResult
	var lastErr error

	for batchStart := 0; batchStart < len(allMessages); batchStart += MaxMicroBatchSize {
		batchEnd := batchStart + MaxMicroBatchSize
		if batchEnd > len(allMessages) {
			batchEnd = len(allMessages)
		}

		chunkMessages := allMessages[batchStart:batchEnd]
		enqueueCmd := queue.EnqueueCommand{
			CF:       cf,
			CFS:      cfs,
			Messages: chunkMessages,
		}

		startEnqueue := time.Now()
		execCtx, execCancel := context.WithTimeout(context.Background(), raftTimeout)
		res, err := dragonboat.ExecuteScheduledRepositoryCommand[queue.EnqueueResult](
			dragonboat.KindEnqueue,
			tenantNode,
			execCtx,
			&enqueueCmd,
			raftTimeout,
			logger,
			"EnqueueCommand (Batch Publish)",
		)
		execCancel()
		logger.Debug().Dur("duration", time.Since(startEnqueue)).Int("messages", len(chunkMessages)).Msg("EnqueueCommand executed in publish-flusher micro-batch")

		if err != nil {
			lastErr = err
			break
		}

		combinedRes.Gauges = append(combinedRes.Gauges, res.Gauges...)
	}

	// Notify results back
	if lastErr != nil {
		for _, item := range items {
			notifyPublishError(item, lastErr)
		}
		return
	}

	if exchangeBO.Config.MetricsCollector != nil {
		for _, gauge := range combinedRes.Gauges {
			exchangeBO.Config.MetricsCollector.UpdateGauges(items[0].Tenant.Code, gauge.QueueCode, gauge.VNamespace, gauge.Pending, gauge.InProcess)
		}
		for key, count := range publishedCounts {
			parts := strings.Split(key, "|")
			exchangeBO.Config.MetricsCollector.RecordPublish(items[0].Tenant.Code, parts[0], parts[1], uint64(count))
		}
	}

	for i, item := range items {
		if routingErr, ok := itemErrors[i]; ok {
			notifyPublishError(item, routingErr)
			continue
		}

		queueMessages := make(map[string]string)
		for _, msgIdx := range itemToMessageIndices[i] {
			msg := allMessages[msgIdx]
			queueCode := msgIDToQueueCode[msg.ID]
			queueMessages[queueCode] = msg.ID
		}

		if item.SendChan != nil {
			item.SendChan <- &pb.PublishStreamResponse{
				ClientMessageId: item.ClientMessageID,
				Confirmed:       true,
				QueueMessages:   queueMessages,
			}
		} else if item.ResponseChan != nil {
			item.ResponseChan <- PublishConfirmation{
				QueueMessages: queueMessages,
				Error:         nil,
			}
		}
	}
}

func notifyPublishError(item PublishBufferedMessage, err error) {
	if item.SendChan != nil {
		item.SendChan <- &pb.PublishStreamResponse{
			ClientMessageId: item.ClientMessageID,
			Confirmed:       false,
			Error:           err.Error(),
		}
	} else if item.ResponseChan != nil {
		item.ResponseChan <- PublishConfirmation{
			Error: err,
		}
	}
}
