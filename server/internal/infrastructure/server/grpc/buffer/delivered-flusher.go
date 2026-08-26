package buffer

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/usecase/command/queue"
	"time"

	"github.com/rs/zerolog"
)

func NewDeliveredFlusher(logger zerolog.Logger, raftTimeout time.Duration) func(ctx context.Context, items []DeliveredBufferedMessage) {
	return func(ctx context.Context, items []DeliveredBufferedMessage) {
		if len(items) == 0 {
			return
		}

		// Group by TenantNode + CF + CFS (Raft partition)
		groups := make(map[string][]DeliveredBufferedMessage)
		for _, item := range items {
			key := item.GetGroupKey()
			groups[key] = append(groups[key], item)
		}

		for _, groupItems := range groups {
			if len(groupItems) == 0 {
				continue
			}

			// All items in the group share the same tenant node and CF/CFS
			tenantNode := groupItems[0].TenantNode
			cf := groupItems[0].CF
			cfs := groupItems[0].CFS

			leaseIDs := make([]string, len(groupItems))
			for i, item := range groupItems {
				leaseIDs[i] = item.LeaseID
			}

			cmd := queue.BulkMarkLeaseDeliveredCommand{
				LeaseIDs: leaseIDs,
				CF:       cf,
				CFS:      cfs,
			}

			res, err := dragonboat.ExecuteRepositoryCommand[queue.BulkMarkLeaseDeliveredResult](
				tenantNode,
				ctx,
				&cmd,
				raftTimeout,
				logger,
				"BulkMarkLeaseDeliveredCommand",
			)

			if err != nil {
				logger.Error().Err(err).Msg("❌ BulkMarkLeaseDeliveredCommand failed")
				for _, item := range groupItems {
					if item.ResponseChan != nil {
						item.ResponseChan <- DeliveredConfirmation{Success: false, Error: err}
					}
				}
				continue
			}

			for _, item := range groupItems {
				if item.ResponseChan != nil {
					item.ResponseChan <- DeliveredConfirmation{Success: res.Success, Error: nil}
				}
			}
		}
	}
}
