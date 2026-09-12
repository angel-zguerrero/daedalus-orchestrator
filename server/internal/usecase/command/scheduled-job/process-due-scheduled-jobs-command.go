package scheduled_job

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	binding_command "deadalus-orch/server/internal/usecase/command/binding"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"strings"
	"time"
)

func init() {
	gob.Register(ProcessDueScheduledJobsCommand{})
	gob.Register(ProcessDueScheduledJobsResult{})
	gob.Register(DispatchedMsgDetail{})
}

type DispatchedMsgDetail struct {
	QueueCode  string
	VNamespace string
	Count      uint64
}

type ProcessDueScheduledJobsResult struct {
	ProcessedJobs     int
	DispatchedMsgs    int
	DispatchedDetails []DispatchedMsgDetail
	Errors            []string
	Gauges            []models.QueueGauges
}

type ProcessDueScheduledJobsCommand struct {
	BatchSize int
	CF        string
	CFS       string
}

func (cmd *ProcessDueScheduledJobsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	result := &ProcessDueScheduledJobsResult{
		Errors: make([]string, 0),
	}

	if cmd.BatchSize <= 0 {
		cmd.BatchSize = 1000
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	scheduledJobRepo, err := db.NewScheduledJobRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	dueJobs, _, err := scheduledJobRepo.FindDueScheduledJobs(now, cmd.BatchSize, "")
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to find due scheduled jobs: %s", err.Error())
		return *commandResult
	}

	if len(dueJobs) == 0 {
		commandResult.Result = result
		return *commandResult
	}

	gaugesMap := make(map[string]models.QueueGauges)
	detailsMap := make(map[string]uint64)

	for _, job := range dueJobs {
		// Concurrency rule: A job in "delivered" state CANNOT be re-queued.
		if job.State == models.ScheduledJobDelivered {
			continue
		}

		oldNextRunAt := job.NextRunAt
		job.State = models.ScheduledJobDelivered

		_, err := scheduledJobRepo.UpdateScheduledJobStateAndRunAt(job, &oldNextRunAt, now)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to update state for job %s: %s", job.ID, err.Error()))
			continue
		}

		// Dispatch task message with deterministic execution ID
		executionID := fmt.Sprintf("exec_%s_%d", job.ID, oldNextRunAt.Unix())
		messagesToEnqueue, trackersToCreate, err := cmd.resolveTargetMessages(uow, idFactory, job, executionID, now)
		if err != nil || len(messagesToEnqueue) == 0 {
			// Revert state back to idle so dispatch can be retried on next poll
			job.State = models.ScheduledJobIdle
			_, _ = scheduledJobRepo.UpdateScheduledJobStateAndRunAt(job, &oldNextRunAt, now)

			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to resolve target for job %s: %s", job.ID, err.Error()))
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("no active target queues found for job %s", job.ID))
			}
			continue
		}

		enqueueCmd := &queue_command.EnqueueCommand{
			Messages: messagesToEnqueue,
			CF:       cmd.CF,
			CFS:      cmd.CFS,
		}

		enqueueResult := enqueueCmd.Execute(uow, now)
		if enqueueResult.Error != "" {
			// Revert state back to idle so dispatch can be retried on next poll
			job.State = models.ScheduledJobIdle
			_, _ = scheduledJobRepo.UpdateScheduledJobStateAndRunAt(job, &oldNextRunAt, now)

			result.Errors = append(result.Errors, fmt.Sprintf("failed to enqueue message for job %s: %s", job.ID, enqueueResult.Error))
			continue
		}

		if len(trackersToCreate) > 0 {
			_, err = trackerRepo.BulkCreateTrackers(trackersToCreate, now)
			if err != nil {
				job.State = models.ScheduledJobIdle
				_, _ = scheduledJobRepo.UpdateScheduledJobStateAndRunAt(job, &oldNextRunAt, now)

				result.Errors = append(result.Errors, fmt.Sprintf("failed to create job trackers for job %s: %s", job.ID, err.Error()))
				continue
			}
		}

		if resData, ok := enqueueResult.Result.(queue_command.EnqueueResult); ok {
			result.DispatchedMsgs += len(resData.Messages)
			for _, g := range resData.Gauges {
				gaugesMap[g.QueueCode] = g
			}

			countsByQueueID := make(map[string]uint64)
			for _, msg := range resData.Messages {
				countsByQueueID[msg.QueueID]++
			}
			for qID, count := range countsByQueueID {
				if queueRepo != nil {
					q, err := queueRepo.GetQueueById(qID, now)
					if err == nil && q != nil {
						detailsMap[q.Code+"|"+q.VNamespace] += count
					}
				}
			}
		}

		result.ProcessedJobs++
	}

	for _, g := range gaugesMap {
		result.Gauges = append(result.Gauges, g)
	}

	for key, count := range detailsMap {
		parts := strings.Split(key, "|")
		result.DispatchedDetails = append(result.DispatchedDetails, DispatchedMsgDetail{
			QueueCode:  parts[0],
			VNamespace: parts[1],
			Count:      count,
		})
	}

	commandResult.Result = result
	return *commandResult
}

func (cmd *ProcessDueScheduledJobsCommand) resolveTargetMessages(
	uow *db.UnitOfWork,
	idFactory db.IDGeneratorFactory,
	job *models.ScheduledJob,
	executionID string,
	now time.Time,
) ([]models.QueueMessage, []*models.ScheduledJobTracker, error) {
	msgID := fmt.Sprintf("msg_%s_%d", job.ID, job.NextRunAt.Unix())

	baseMessage := models.QueueMessage{
		ID:             fmt.Sprintf("msg_%s_%d", job.ID, job.NextRunAt.Unix()),
		MessageID:      msgID,
		Content:        []byte(job.Content),
		ContentType:    job.ContentType,
		Headers:        job.Headers,
		Handler:        job.Handler,
		Parameters:     job.Parameters,
		Priority:       job.Priority,
		VNamespace:     job.VNamespace,
		ScheduledJobID: job.ID,
		ExecutionID:    executionID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if job.TargetType == string(models.ScheduledJobTargetQueue) {
		baseMessage.QueueID = job.TargetID
		baseMessage.ID = fmt.Sprintf("msg_%s_%d_%s", job.ID, job.NextRunAt.Unix(), job.TargetID)

		tracker := &models.ScheduledJobTracker{
			ID:             fmt.Sprintf("trk_%s_%s", executionID, job.TargetID),
			ScheduledJobID: job.ID,
			ExecutionID:    executionID,
			QueueID:        job.TargetID,
			Status:         models.ScheduledJobTrackerPending,
		}
		return []models.QueueMessage{baseMessage}, []*models.ScheduledJobTracker{tracker}, nil
	}

	if job.TargetType == string(models.ScheduledJobTargetExchange) {
		routingKey := job.RoutingKeyOrPatternOrQueueCode
		if rk, ok := job.Headers["routing_key"]; ok && rk != "" {
			routingKey = rk
		} else if rk, ok := job.Headers["routingKey"]; ok && rk != "" {
			routingKey = rk
		}

		resolveCmd := &binding_command.ResolveAndFetchQueuesCommand{
			ExchangeCode:   job.TargetCode,
			RoutingKey:     routingKey,
			MessageHeaders: job.Headers,
			VNamespace:     job.VNamespace,
			CF:             cmd.CF,
			CFS:            cmd.CFS,
		}

		res := resolveCmd.Execute(uow, now)
		if res.Error != "" {
			return nil, nil, fmt.Errorf("exchange queue resolution error: %s", res.Error)
		}

		fetchRes, ok := res.Result.(binding_command.ResolveAndFetchQueuesResult)
		if !ok || len(fetchRes.Queues) == 0 {
			return nil, nil, nil
		}

		messages := make([]models.QueueMessage, 0, len(fetchRes.Queues))
		trackers := make([]*models.ScheduledJobTracker, 0, len(fetchRes.Queues))
		for _, q := range fetchRes.Queues {
			if q.State == models.QueueActive {
				m := baseMessage
				m.ID = fmt.Sprintf("msg_%s_%d_%s", job.ID, job.NextRunAt.Unix(), q.ID)
				m.QueueID = q.ID
				messages = append(messages, m)

				t := &models.ScheduledJobTracker{
					ID:             fmt.Sprintf("trk_%s_%s", executionID, q.ID),
					ScheduledJobID: job.ID,
					ExecutionID:    executionID,
					QueueID:        q.ID,
					Status:         models.ScheduledJobTrackerPending,
				}
				trackers = append(trackers, t)
			}
		}

		return messages, trackers, nil
	}

	return nil, nil, fmt.Errorf("unknown target type: %s", job.TargetType)
}
