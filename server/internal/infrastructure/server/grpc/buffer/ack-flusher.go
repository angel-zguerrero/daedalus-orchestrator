package buffer

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/usecase/command/queue"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

func NewAckFlusher(logger zerolog.Logger, raftTimeout time.Duration) func(ctx context.Context, items []AckBufferedMessage) {
	return func(ctx context.Context, items []AckBufferedMessage) {
		if len(items) == 0 {
			return
		}

		// Group by TenantNode + CF + CFS (Raft partition)
		groups := make(map[string][]AckBufferedMessage)
		for _, item := range items {
			key := item.GetGroupKey()
			groups[key] = append(groups[key], item)
		}

		for _, group := range groups {
			processAckGroup(ctx, group, logger, raftTimeout)
		}
	}
}

func processAckGroup(ctx context.Context, items []AckBufferedMessage, logger zerolog.Logger, raftTimeout time.Duration) {
	if len(items) == 0 {
		return
	}

	logger.Debug().Int("items_count", len(items)).Msg("processAckGroup started")

	firstItem := items[0]
	tenantNode := firstItem.TenantNode
	cf := firstItem.CF
	cfs := firstItem.CFS

	if tenantNode == nil {
		err := fmt.Errorf("tenantNode is nil for CF=%s CFS=%s", cf, cfs)
		for _, item := range items {
			notifyAckError(item, err)
		}
		return
	}

	leaseIDs := make([]string, 0, len(items))
	for _, item := range items {
		leaseIDs = append(leaseIDs, item.LeaseID)
	}

	ackCmd := queue.BulkAckMessageCommand{
		CF:       cf,
		CFS:      cfs,
		LeaseIDs: leaseIDs,
	}

	res, err := dragonboat.ExecuteRepositoryCommand[queue.BulkAckMessageResult](
		tenantNode,
		ctx,
		&ackCmd,
		raftTimeout,
		logger,
		"BulkAckMessageCommand",
	)

	if err != nil {
		for _, item := range items {
			notifyAckError(item, err)
		}
		return
	}

	for i, item := range items {
		if i < len(res.Results) {
			result := res.Results[i]
			if !result.Success {
				notifyAckError(item, fmt.Errorf(result.Message))
			} else {
				select {
				case item.ResponseChan <- AckConfirmation{
					Success:             true,
					Message:             result.Message,
					QueueCode:           result.QueueCode,
					VNamespace:          result.VNamespace,
					ProcessingLatencyMs: result.ProcessingLatencyMs,
					QueueLatencyMs:      result.QueueLatencyMs,
					Pending:             result.Pending,
					InProcess:           result.InProcess,
				}:
				default:
					logger.Warn().Str("lease_id", item.LeaseID).Msg("Ack response channel is full or closed")
				}
			}
		} else {
			notifyAckError(item, fmt.Errorf("no result returned for lease ID"))
		}
	}
}

func notifyAckError(item AckBufferedMessage, err error) {
	select {
	case item.ResponseChan <- AckConfirmation{Error: err}:
	default:
	}
}
