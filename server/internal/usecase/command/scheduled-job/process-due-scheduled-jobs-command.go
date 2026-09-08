package scheduled_job

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	binding_command "deadalus-orch/server/internal/usecase/command/binding"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func init() {
	gob.Register(ProcessDueScheduledJobsCommand{})
	gob.Register(ProcessDueScheduledJobsResult{})
}

type ProcessDueScheduledJobsResult struct {
	ProcessedJobs int
	DispatchedMsgs int
	Errors        []string
	Gauges        []models.QueueGauges
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

		// Dispatch task message
		messagesToEnqueue, err := cmd.resolveTargetMessages(uow, idFactory, job, now)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to resolve target for job %s: %s", job.ID, err.Error()))
			continue
		}

		if len(messagesToEnqueue) == 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("no active target queues found for job %s", job.ID))
			continue
		}

		enqueueCmd := &queue_command.EnqueueCommand{
			Messages: messagesToEnqueue,
			CF:       cmd.CF,
			CFS:      cmd.CFS,
		}

		enqueueResult := enqueueCmd.Execute(uow, now)
		if enqueueResult.Error != "" {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to enqueue message for job %s: %s", job.ID, enqueueResult.Error))
			continue
		}

		if resData, ok := enqueueResult.Result.(queue_command.EnqueueResult); ok {
			result.DispatchedMsgs += len(resData.Messages)
			for _, g := range resData.Gauges {
				gaugesMap[g.QueueCode] = g
			}
		}

		result.ProcessedJobs++
	}

	for _, g := range gaugesMap {
		result.Gauges = append(result.Gauges, g)
	}

	commandResult.Result = result
	return *commandResult
}

func (cmd *ProcessDueScheduledJobsCommand) resolveTargetMessages(
	uow *db.UnitOfWork,
	idFactory db.IDGeneratorFactory,
	job *models.ScheduledJob,
	now time.Time,
) ([]models.QueueMessage, error) {
	msgID := uuid.New().String()

	baseMessage := models.QueueMessage{
		ID:             uuid.New().String(),
		MessageID:      msgID,
		Content:        []byte(job.Content),
		ContentType:    job.ContentType,
		Headers:        job.Headers,
		Handler:        job.Handler,
		Parameters:     job.Parameters,
		Priority:       job.Priority,
		VNamespace:     job.VNamespace,
		ScheduledJobID: job.ID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if job.TargetType == string(models.ScheduledJobTargetQueue) {
		baseMessage.QueueID = job.TargetID
		return []models.QueueMessage{baseMessage}, nil
	}

	if job.TargetType == string(models.ScheduledJobTargetExchange) {
		resolveCmd := &binding_command.ResolveAndFetchQueuesCommand{
			ExchangeCode:   job.TargetCode,
			RoutingKey:     job.RoutingKeyOrPatternOrQueueCode,
			MessageHeaders: job.Headers,
			VNamespace:     job.VNamespace,
			CF:             cmd.CF,
			CFS:            cmd.CFS,
		}

		res := resolveCmd.Execute(uow, now)
		if res.Error != "" {
			return nil, fmt.Errorf("exchange queue resolution error: %s", res.Error)
		}

		fetchRes, ok := res.Result.(binding_command.ResolveAndFetchQueuesResult)
		if !ok || len(fetchRes.Queues) == 0 {
			return nil, nil
		}

		messages := make([]models.QueueMessage, 0, len(fetchRes.Queues))
		for _, q := range fetchRes.Queues {
			if q.State == models.QueueActive {
				m := baseMessage
				m.ID = uuid.New().String()
				m.QueueID = q.ID
				messages = append(messages, m)
			}
		}

		return messages, nil
	}

	return nil, fmt.Errorf("unknown target type: %s", job.TargetType)
}
