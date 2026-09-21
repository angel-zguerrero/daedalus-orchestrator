package workflow_definition

import (
	"encoding/gob"
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
	gob.Register(GetWorkflowQueuesCommand{})
	gob.Register([]models.Queue{})
}

type GetWorkflowQueuesCommand struct {
	WorkflowID string
	CF         string
	CFS        string
}

func (cmd *GetWorkflowQueuesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.WorkflowID == "" {
		commandResult.Error = "WorkflowID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	queues, err := queueRepo.GetQueuesByWorkflowDefinitionID(cmd.WorkflowID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to get workflow queues: %v", err)
		return *commandResult
	}

	// Auto-provision if missing (e.g., for workflows created prior to Phase 2)
	if len(queues) == 0 {
		wfRepo, errWfRepo := db.NewWorkflowDefinitionRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if errWfRepo == nil {
			wf, errWf := wfRepo.GetWorkflowDefinitionByID(cmd.WorkflowID, now)
			if errWf == nil && wf != nil {
				execQueueID := strings.ReplaceAll(uuid.New().String(), "-", "")
				actQueueID := strings.ReplaceAll(uuid.New().String(), "-", "")
				randToken := uuid.New().String()[:8]

				execQueueCode := fmt.Sprintf("wf-exec-%s-%s", wf.Code, randToken)
				execQueueName := fmt.Sprintf("%s Execution Queue (%s)", wf.Name, randToken)
				actQueueCode := fmt.Sprintf("wf-act-%s-%s", wf.Code, randToken)
				actQueueName := fmt.Sprintf("%s Activity Queue (%s)", wf.Name, randToken)

				vns := wf.VNamespace
				if vns == "" {
					vns = "default"
				}

				execQueue := models.Queue{
					ID:                   execQueueID,
					Code:                 execQueueCode,
					Name:                 execQueueName,
					Type:                 models.WorkflowExecutionQueue,
					WorkflowDefinitionID: wf.ID,
					VNamespace:           vns,
					State:                models.QueueActive,
					AllowDuplicated:      true,
					MaxAttempts:          1,
				}

				actQueue := models.Queue{
					ID:                   actQueueID,
					Code:                 actQueueCode,
					Name:                 actQueueName,
					Type:                 models.WorkflowActivityQueue,
					WorkflowDefinitionID: wf.ID,
					VNamespace:           vns,
					State:                models.QueueActive,
					AllowDuplicated:      true,
					MaxAttempts:          1,
				}

				assertCmd := &queue.AssertQueueCommand{
					Queues: []models.Queue{execQueue, actQueue},
					CF:     cmd.CF,
					CFS:    cmd.CFS,
				}
				_ = assertCmd.Execute(uow, now)

				// Re-query after provisioning
				queues, _ = queueRepo.GetQueuesByWorkflowDefinitionID(cmd.WorkflowID, now)
			}
		}
	}

	commandResult.Result = queues
	return *commandResult
}
