package buffer

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	pbQueue "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/queue"
	"deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func NewEnqueueFlusher(queueBO *bo.QueueBO, logger zerolog.Logger, raftTimeout time.Duration) func(ctx context.Context, items []EnqueueBufferedMessage) {
	return func(ctx context.Context, items []EnqueueBufferedMessage) {
		if len(items) == 0 {
			return
		}

		// Group by TenantNode + CF + CFS (Raft partition)
		groups := make(map[string][]EnqueueBufferedMessage)
		for _, item := range items {
			key := item.GetGroupKey()
			groups[key] = append(groups[key], item)
		}

		for _, group := range groups {
			processEnqueueGroup(ctx, group, queueBO, logger, raftTimeout)
		}
	}
}

func processEnqueueGroup(ctx context.Context, items []EnqueueBufferedMessage, queueBO *bo.QueueBO, logger zerolog.Logger, raftTimeout time.Duration) {
	if len(items) == 0 {
		return
	}

	tenantNode := items[0].TenantNode
	cf := items[0].CF
	cfs := items[0].CFS

	var allMessages []models.QueueMessage
	itemToMessageIndices := make(map[int]int)
	itemErrors := make(map[int]error)

	publishedCounts := make(map[string]int)
	
	queueCache := make(map[string]models.Queue)

	for i, item := range items {
		cacheKey := fmt.Sprintf("%s|%s", item.QueueCode, item.VNamespace)
		
		var q models.Queue
		var err error
		
		if cachedQueue, ok := queueCache[cacheKey]; ok {
			q = cachedQueue
		} else {
			q, err = queueBO.GetQueue(ctx, item.QueueCode, item.VNamespace, false, cf, cfs, item.Tenant, tenantNode)
			if err == nil {
				queueCache[cacheKey] = q
			}
		}

		if err != nil {
			itemErrors[i] = fmt.Errorf("failed to get queue %s: %w", item.QueueCode, err)
			continue
		}
		if q.ID == "" {
			itemErrors[i] = fmt.Errorf("queue not found: %s", item.QueueCode)
			continue
		}

		msg := item.Message
		msg.QueueID = q.ID
		u := uuid.New()
		msg.ID = hex.EncodeToString(u[:])
		if msg.MessageID == "" {
			msg.MessageID = msg.ID
		}

		publishedCounts[q.Code+"|"+item.VNamespace]++

		itemToMessageIndices[i] = len(allMessages)
		allMessages = append(allMessages, msg)
	}

	if len(allMessages) == 0 {
		for i, item := range items {
			if err, ok := itemErrors[i]; ok {
				notifyEnqueueError(item, err)
			} else {
				notifyEnqueueError(item, fmt.Errorf("unknown routing error"))
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

		execCtx, execCancel := context.WithTimeout(context.Background(), raftTimeout)
		res, err := dragonboat.ExecuteScheduledRepositoryCommand[queue.EnqueueResult](
			dragonboat.KindEnqueue,
			tenantNode,
			execCtx,
			&enqueueCmd,
			raftTimeout,
			logger,
			"EnqueueCommand (Batch Enqueue)",
		)
		execCancel()

		if err != nil {
			lastErr = err
			break
		}

		combinedRes.Gauges = append(combinedRes.Gauges, res.Gauges...)
	}

	if lastErr != nil {
		for _, item := range items {
			notifyEnqueueError(item, lastErr)
		}
		return
	}

	if queueBO.Config.MetricsCollector != nil {
		for _, gauge := range combinedRes.Gauges {
			queueBO.Config.MetricsCollector.UpdateGauges(items[0].Tenant.Code, gauge.QueueCode, gauge.VNamespace, gauge.Pending, gauge.InProcess)
		}
		for key, count := range publishedCounts {
			parts := strings.Split(key, "|")
			queueBO.Config.MetricsCollector.RecordPublish(items[0].Tenant.Code, parts[0], parts[1], uint64(count))
		}
	}

	for i, item := range items {
		if routingErr, ok := itemErrors[i]; ok {
			notifyEnqueueError(item, routingErr)
			continue
		}

		msgIdx := itemToMessageIndices[i]
		msg := allMessages[msgIdx]

		if item.SendChan != nil {
			if item.StreamCtx != nil {
				select {
				case <-item.StreamCtx.Done():
				case item.SendChan <- &pbQueue.EnqueueStreamResponse{
					ClientMessageId: item.ClientMessageID,
					Confirmed:       true,
					MessageId:       msg.MessageID,
				}:
				}
			} else {
				item.SendChan <- &pbQueue.EnqueueStreamResponse{
					ClientMessageId: item.ClientMessageID,
					Confirmed:       true,
					MessageId:       msg.MessageID,
				}
			}
		} else if item.ResponseChan != nil {
			item.ResponseChan <- EnqueueConfirmation{
				MessageID: msg.MessageID,
				Error:     nil,
			}
		}
	}
}

func notifyEnqueueError(item EnqueueBufferedMessage, err error) {
	if item.SendChan != nil {
		if item.StreamCtx != nil {
			select {
			case <-item.StreamCtx.Done():
			case item.SendChan <- &pbQueue.EnqueueStreamResponse{
				ClientMessageId: item.ClientMessageID,
				Confirmed:       false,
				Error:           err.Error(),
			}:
			}
		} else {
			item.SendChan <- &pbQueue.EnqueueStreamResponse{
				ClientMessageId: item.ClientMessageID,
				Confirmed:       false,
				Error:           err.Error(),
			}
		}
	} else if item.ResponseChan != nil {
		item.ResponseChan <- EnqueueConfirmation{
			Error: err,
		}
	}
}
