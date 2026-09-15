package workflow_definition

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
)

func init() {
	gob.Register(CreateWorkflowDefinitionCommand{})
}

type CreateWorkflowDefinitionCommand struct {
	WorkflowDefinition models.WorkflowDefinition
	ExecQueue          models.Queue
	ActQueue           models.Queue
	CF                 string
	CFS                string
}

func (cmd *CreateWorkflowDefinitionCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.WorkflowDefinition.ID == "" {
		commandResult.Error = "ID is required and must be generated outside the command"
		return *commandResult
	}

	if cmd.WorkflowDefinition.Code == "" {
		commandResult.Error = "Code is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewWorkflowDefinitionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	id, err := repo.CreateWorkflowDefinition(&cmd.WorkflowDefinition, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to create workflow definition: %s", err.Error())
		return *commandResult
	}

	// Prepare Execution and Activity queues
	execQueue := cmd.ExecQueue
	actQueue := cmd.ActQueue

	vns := cmd.WorkflowDefinition.VNamespace
	if vns == "" {
		vns = "default"
	}

	// Ensure IDs are generated if not pre-populated
	if execQueue.ID == "" {
		execQueue.ID = idFactory.GenerateID()
	}
	if execQueue.Code == "" {
		execQueue.Code = fmt.Sprintf("wf-exec-%s", cmd.WorkflowDefinition.Code)
	}
	if execQueue.Name == "" {
		execQueue.Name = fmt.Sprintf("%s Executions", cmd.WorkflowDefinition.Name)
	}
	execQueue.Type = models.WorkflowExecutionQueue
	execQueue.WorkflowDefinitionID = id
	execQueue.VNamespace = vns
	execQueue.State = models.QueueActive
	execQueue.AllowDuplicated = true
	execQueue.MaxAttempts = 1000
	if execQueue.DesiredPriorityThresholds == nil {
		execQueue.DesiredPriorityThresholds = map[int]int{0: 0}
	}
	if execQueue.PriorityThresholds == nil {
		execQueue.PriorityThresholds = map[int]int{0: 0}
	}

	if actQueue.ID == "" {
		actQueue.ID = idFactory.GenerateID()
	}
	if actQueue.Code == "" {
		actQueue.Code = fmt.Sprintf("wf-act-%s", cmd.WorkflowDefinition.Code)
	}
	if actQueue.Name == "" {
		actQueue.Name = fmt.Sprintf("%s Activities", cmd.WorkflowDefinition.Name)
	}
	actQueue.Type = models.WorkflowActivityQueue
	actQueue.WorkflowDefinitionID = id
	actQueue.VNamespace = vns
	actQueue.State = models.QueueActive
	actQueue.AllowDuplicated = true
	actQueue.MaxAttempts = 1000
	if actQueue.DesiredPriorityThresholds == nil {
		actQueue.DesiredPriorityThresholds = map[int]int{0: 0}
	}
	if actQueue.PriorityThresholds == nil {
		actQueue.PriorityThresholds = map[int]int{0: 0}
	}

	assertCmd := &queue.AssertQueueCommand{
		Queues: []models.Queue{execQueue, actQueue},
		CF:     cmd.CF,
		CFS:    cmd.CFS,
	}
	assertRes := assertCmd.Execute(uow, now)
	if assertRes.Error != "" {
		commandResult.Error = fmt.Sprintf("failed to provision workflow queues: %s", assertRes.Error)
		return *commandResult
	}

	cmd.WorkflowDefinition.ID = id
	commandResult.Result = cmd.WorkflowDefinition
	return *commandResult
}
