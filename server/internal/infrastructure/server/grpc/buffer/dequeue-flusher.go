package buffer

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/usecase/command/queue"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

func NewDequeueFlusher(logger zerolog.Logger, raftTimeout time.Duration) func(ctx context.Context, items []DequeueBufferedMessage) {
	return func(ctx context.Context, items []DequeueBufferedMessage) {
		if len(items) == 0 {
			return
		}

		// Group by TenantNode + CF + CFS (Raft partition)
		groups := make(map[string][]DequeueBufferedMessage)
		for _, item := range items {
			key := item.GetGroupKey()
			groups[key] = append(groups[key], item)
		}

		for _, group := range groups {
			processDequeueGroup(ctx, group, logger, raftTimeout)
		}
	}
}

func processDequeueGroup(ctx context.Context, items []DequeueBufferedMessage, logger zerolog.Logger, raftTimeout time.Duration) {
	if len(items) == 0 {
		return
	}

	tenantNode := items[0].TenantNode
	cf := items[0].CF
	cfs := items[0].CFS

	requests := make([]queue.BulkDequeueRequest, len(items))
	for i, item := range items {
		requests[i] = queue.BulkDequeueRequest{
			QueueID:                      item.QueueID,
			JobWorkerID:                  item.JobWorkerID,
			LeaseDuration:                item.LeaseDuration,
			JobWorkerCapacityPolicyIndex: item.JobWorkerCapacityPolicyIndex,
		}
	}

	const MaxMicroBatchSize = 250

	combinedResults := make(map[int]queue.DequeueResult)
	var lastErr error

	for batchStart := 0; batchStart < len(requests); batchStart += MaxMicroBatchSize {
		batchEnd := batchStart + MaxMicroBatchSize
		if batchEnd > len(requests) {
			batchEnd = len(requests)
		}

		chunkRequests := requests[batchStart:batchEnd]
		bulkDequeueCmd := &queue.BulkDequeueCommand{
			CF:       cf,
			CFS:      cfs,
			Requests: chunkRequests,
		}

		start := time.Now()
		execCtx, execCancel := context.WithTimeout(context.Background(), raftTimeout)
		res, err := dragonboat.ExecuteScheduledRepositoryCommand[queue.BulkDequeueResult](
			dragonboat.KindDequeue,
			tenantNode,
			execCtx,
			bulkDequeueCmd,
			raftTimeout,
			logger,
			"BulkDequeueCommand (Batch)",
		)
		execCancel()
		logger.Debug().Dur("duration", time.Since(start)).Int("count", len(chunkRequests)).Msg("BulkDequeueCommand executed micro-batch")

		if err != nil {
			lastErr = err
			break
		}

		for chunkIdx, dequeueRes := range res.Results {
			globalIdx := batchStart + chunkIdx
			combinedResults[globalIdx] = dequeueRes
		}
	}

	// Notify results back
	for i, item := range items {
		conf := DequeueConfirmation{
			Error: lastErr,
		}
		if lastErr == nil {
			if result, ok := combinedResults[i]; ok {
				conf.Result = &result
			} else {
				conf.Error = fmt.Errorf("queue is empty or unavailable")
			}
		}

		select {
		case item.ResponseChan <- conf:
		default:
		}
	}
}
