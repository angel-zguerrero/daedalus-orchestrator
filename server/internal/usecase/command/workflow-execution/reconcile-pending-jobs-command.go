package workflow_execution

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"

	"github.com/google/uuid"
)

func init() {
	gob.Register(ReconcilePendingWorkflowJobsCommand{})
	gob.Register(ReconcilePendingWorkflowJobsResult{})
}

type ReconcilePendingWorkflowJobsCommand struct {
	WorkflowDefinitionID string
	QueueID              string
	CF                   string
	CFS                  string
}

type ReconcilePendingWorkflowJobsResult struct {
	ReEnqueuedCount int `json:"reEnqueuedCount"`
}

func (cmd *ReconcilePendingWorkflowJobsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	res := ReconcilePendingWorkflowJobsResult{}

	if cmd.WorkflowDefinitionID == "" || cmd.QueueID == "" {
		commandResult.Result = res
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	jobRepo, err := db.NewWorkflowJobRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	execRepo, err := db.NewWorkflowExecutionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	q, err := queueRepo.GetQueueById(cmd.QueueID, now)
	if err != nil || q == nil || q.MessagesCount > 0 || q.CurrentDeliveringMessages > 0 {
		commandResult.Result = res
		return *commandResult
	}

	query := fmt.Sprintf("WorkflowDefinitionID = %s & Status = %s", cmd.WorkflowDefinitionID, string(models.WorkflowJobStatusPending))
	found, err := jobRepo.Find(query, 50, "", now)
	if err != nil || found == nil || len(found.Entities) == 0 {
		commandResult.Result = res
		return *commandResult
	}

	var messagesToEnqueue []models.QueueMessage
	staleThreshold := now.Add(-6 * time.Second)

	for i := range found.Entities {
		job := &found.Entities[i]
		if job.Status != models.WorkflowJobStatusPending {
			continue
		}
		// Give newly created or scheduled jobs at least 6 seconds before considering them orphaned
		if !job.UpdatedAt.IsZero() && job.UpdatedAt.After(staleThreshold) {
			continue
		}

		execution, _ := execRepo.GetWorkflowExecutionByID(job.WorkflowExecutionID, now)
		if execution == nil || (execution.Status != models.WorkflowExecutionStatusRunning && execution.Status != models.WorkflowExecutionStatusPending) {
			job.Status = models.WorkflowJobStatusFailed
			if job.Error == "" {
				job.Error = "parent workflow execution is no longer active"
			}
			jobRepo.UpdateWorkflowJob(job, now)
			continue
		}

		job.UpdatedAt = now
		jobRepo.UpdateWorkflowJob(job, now)

		msgID := strings.ReplaceAll(uuid.New().String(), "-", "")
		contentBytes, _ := json.Marshal(map[string]interface{}{
			"executionId":  job.WorkflowExecutionID,
			"tokenId":      job.ExecutionTokenID,
			"jobId":        job.ID,
			"activityId":   job.ActivityID,
			"activityName": job.ActivityName,
			"activityType": job.ActivityType,
			"input":        job.Input,
		})

		messagesToEnqueue = append(messagesToEnqueue, models.QueueMessage{
			ID:          msgID,
			MessageID:   msgID,
			QueueID:     q.ID,
			VNamespace:  job.VNamespace,
			Content:     contentBytes,
			ContentType: "application/json",
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	if len(messagesToEnqueue) > 0 {
		if q.DesiredPriorityThresholds == nil {
			q.DesiredPriorityThresholds = map[int]int{0: 0}
			q.PriorityThresholds = map[int]int{0: 0}
			queueRepo.UpdateQueue(q, now)
		}
		enqCmd := &queue.EnqueueCommand{
			Messages: messagesToEnqueue,
			CF:       cmd.CF,
			CFS:      cmd.CFS,
		}
		enqRes := enqCmd.Execute(uow, now)
		if enqRes.Error == "" {
			res.ReEnqueuedCount = len(messagesToEnqueue)
		}
	}

	commandResult.Result = res
	return *commandResult
}
