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

	bulkDequeueCmd := &queue.BulkDequeueCommand{
		CF:       cf,
		CFS:      cfs,
		Requests: requests,
	}

	start := time.Now()
	res, err := dragonboat.ExecuteRepositoryCommand[queue.BulkDequeueResult](
		tenantNode,
		ctx,
		bulkDequeueCmd,
		raftTimeout,
		logger,
		"BulkDequeueCommand (Batch)",
	)

	logger.Info().Dur("duration", time.Since(start)).Int("count", len(items)).Msg("BulkDequeueCommand executed")

	// Notify results back
	for i, item := range items {
		conf := DequeueConfirmation{
			Error: err,
		}
		if err == nil {
			if result, ok := res.Results[i]; ok {
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
