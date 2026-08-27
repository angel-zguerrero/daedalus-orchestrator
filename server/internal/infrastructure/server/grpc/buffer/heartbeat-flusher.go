package buffer

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	job_worker_command "deadalus-orch/server/internal/usecase/command/job-worker"
	"deadalus-orch/shared/models"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

func NewJobWorkerHeartbeatFlusher(logger zerolog.Logger, raftTimeout time.Duration) func(ctx context.Context, items []JobWorkerHeartbeatBufferedMessage) {
	return func(ctx context.Context, items []JobWorkerHeartbeatBufferedMessage) {
		if len(items) == 0 {
			return
		}

		// Deduplicate job workers by ID (keep the latest state for each worker ID)
		workerMap := make(map[string]models.JobWorker)
		masterNodeMap := make(map[string]*dragonboat.RaftNode)
		itemMap := make(map[string][]JobWorkerHeartbeatBufferedMessage)

		for _, item := range items {
			wID := item.JobWorker.ID
			workerMap[wID] = item.JobWorker
			if item.MasterNode != nil {
				masterNodeMap[wID] = item.MasterNode
			}
			itemMap[wID] = append(itemMap[wID], item)
		}

		// Group deduplicated workers by MasterNode
		nodeGroups := make(map[string][]*models.JobWorker)
		nodeRefMap := make(map[string]*dragonboat.RaftNode)

		for wID, worker := range workerMap {
			mNode := masterNodeMap[wID]
			if mNode == nil {
				for _, item := range itemMap[wID] {
					notifyHeartbeatError(item, fmt.Errorf("masterNode is nil for worker %s", wID))
				}
				continue
			}
			gKey := fmt.Sprintf("%d-%d", mNode.ShardID, mNode.ReplicaID)
			wCopy := worker
			nodeGroups[gKey] = append(nodeGroups[gKey], &wCopy)
			nodeRefMap[gKey] = mNode
		}

		for gKey, workers := range nodeGroups {
			mNode := nodeRefMap[gKey]
			processHeartbeatGroup(ctx, mNode, workers, itemMap, logger, raftTimeout)
		}
	}
}

func processHeartbeatGroup(
	ctx context.Context,
	masterNode *dragonboat.RaftNode,
	workers []*models.JobWorker,
	itemMap map[string][]JobWorkerHeartbeatBufferedMessage,
	logger zerolog.Logger,
	raftTimeout time.Duration,
) {
	if len(workers) == 0 {
		return
	}

	jobWorkersList := make([]models.JobWorker, len(workers))
	for i, w := range workers {
		jobWorkersList[i] = *w
	}

	upsertCmd := &job_worker_command.UpsertJobWorkerCommand{
		JobWorkers:     jobWorkersList,
		ApplyHeartbeat: true,
	}

	_, err := dragonboat.ExecuteRepositoryCommand[[]models.JobWorker](
		masterNode,
		ctx,
		upsertCmd,
		raftTimeout,
		logger,
		"upsert job workers bulk",
	)

	for _, w := range workers {
		for _, item := range itemMap[w.ID] {
			if err != nil {
				notifyHeartbeatError(item, err)
			} else {
				notifyHeartbeatSuccess(item)
			}
		}
	}
}

func notifyHeartbeatError(item JobWorkerHeartbeatBufferedMessage, err error) {
	select {
	case item.ResponseChan <- HeartbeatConfirmation{Success: false, Error: err}:
	default:
	}
}

func notifyHeartbeatSuccess(item JobWorkerHeartbeatBufferedMessage) {
	select {
	case item.ResponseChan <- HeartbeatConfirmation{Success: true, Error: nil}:
	default:
	}
}
