package workflow_execution

import (
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/bpmn"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"

	"github.com/google/uuid"
)

func init() {
	gob.Register(StartWorkflowExecutionCommand{})
}

type StartWorkflowExecutionCommand struct {
	ExecutionID          string
	InitialTokenID       string
	WorkflowDefinitionID string
	ExecutionKey         string
	OnVersionChange      models.VersionChangePolicy
	Input                map[string]interface{}
	VNamespace           string
	CF                   string
	CFS                  string
}

func (cmd *StartWorkflowExecutionCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.WorkflowDefinitionID == "" {
		commandResult.Error = "WorkflowDefinitionID is required"
		return *commandResult
	}

	execID := cmd.ExecutionID
	if execID == "" {
		execID = strings.ReplaceAll(uuid.New().String(), "-", "")
	}

	tokenID := cmd.InitialTokenID
	if tokenID == "" {
		tokenID = strings.ReplaceAll(uuid.New().String(), "-", "")
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	// 1. Fetch Workflow Definition
	defRepo, err := db.NewWorkflowDefinitionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	def, err := defRepo.GetWorkflowDefinitionByID(cmd.WorkflowDefinitionID, now)
	if err != nil || def == nil {
		commandResult.Error = fmt.Sprintf("workflow definition not found: %s", cmd.WorkflowDefinitionID)
		return *commandResult
	}

	if !def.IsActive {
		commandResult.Error = "workflow definition is not active"
		return *commandResult
	}

	if def.HasDesignErrors {
		errMsg := "cannot execute workflow definition: workflow has design errors"
		if len(def.DesignErrorMessages) > 0 {
			errMsg = fmt.Sprintf("cannot execute workflow definition: workflow has design errors (%s)", strings.Join(def.DesignErrorMessages, "; "))
		}
		commandResult.Error = errMsg
		return *commandResult
	}

	// 2. Parse BPMN Model
	bpmnModel, err := bpmn.ParseBPMN(def.Payload)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to parse workflow BPMN model: %s", err.Error())
		return *commandResult
	}

	if bpmnModel.StartNodeID == "" {
		commandResult.Error = "workflow definition has no StartEvent node"
		return *commandResult
	}

	startNode := bpmnModel.GetNode(bpmnModel.StartNodeID)
	if startNode != nil && len(startNode.FormFields) > 0 {
		if err := bpmn.ValidateFormInput(startNode.FormFields, cmd.Input); err != nil {
			commandResult.Error = fmt.Sprintf("invalid start form input: %s", err.Error())
			return *commandResult
		}
	}

	if err := bpmn.ValidateScriptTasks(bpmnModel); err != nil {
		commandResult.Error = fmt.Sprintf("invalid script task configuration: %s", err.Error())
		return *commandResult
	}

	vns := cmd.VNamespace
	if vns == "" {
		vns = def.VNamespace
	}
	if vns == "" {
		vns = "default"
	}

	inputData := cmd.Input
	if inputData == nil {
		inputData = make(map[string]interface{})
	}

	stateData := make(map[string]interface{})
	for k, v := range inputData {
		stateData[k] = v
	}

	// Determine execution version change policy
	execOnVersionChange := cmd.OnVersionChange
	if execOnVersionChange == "" {
		execOnVersionChange = models.VersionChangePolicyContinue
	}

	// 3. Create WorkflowExecution record
	execRepo, err := db.NewWorkflowExecutionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if cmd.ExecutionKey != "" {
		activeExec, err := execRepo.GetActiveExecutionByBusinessKey(vns, def.ID, cmd.ExecutionKey, now)
		if err != nil {
			commandResult.Error = fmt.Sprintf("failed to validate execution business key: %s", err.Error())
			return *commandResult
		}
		if activeExec != nil {
			commandResult.Error = fmt.Sprintf("cannot start workflow execution: execution with business key '%s' is already in execution (ID: %s, status: %s)", cmd.ExecutionKey, activeExec.ID, activeExec.Status)
			return *commandResult
		}
	}

	execution := &models.WorkflowExecution{
		ID:                        execID,
		WorkflowDefinitionID:      def.ID,
		WorkflowDefinitionVersion: def.Version,
		OnVersionChange:           execOnVersionChange,
		PayloadSnapshot:           def.Payload,
		VNamespace:                vns,
		ExecutionKey:              cmd.ExecutionKey,
		Status:                    models.WorkflowExecutionStatusRunning,
		Input:                     inputData,
		StateData:                 stateData,
		StartedAt:                 &now,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	_, err = execRepo.CreateWorkflowExecution(execution, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to create workflow execution: %s", err.Error())
		return *commandResult
	}

	// 4. Create Initial ExecutionToken
	tokenRepo, err := db.NewExecutionTokenRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	initialToken := &models.ExecutionToken{
		ID:                   tokenID,
		WorkflowExecutionID: execution.ID,
		WorkflowDefinitionID: def.ID,
		VNamespace:           vns,
		CurrentNodeID:        bpmnModel.StartNodeID,
		Status:               models.ExecutionTokenStatusActive,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	_, err = tokenRepo.CreateExecutionToken(initialToken, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to create execution token: %s", err.Error())
		return *commandResult
	}

	// 5. Find Execution Queue & Enqueue Token Message
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to init queue repository: %s", err.Error())
		return *commandResult
	}

	queues, err := queueRepo.GetQueuesByWorkflowDefinitionID(def.ID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to fetch workflow queues: %s", err.Error())
		return *commandResult
	}

	var execQ *models.Queue
	for i := range queues {
		if queues[i].Type == models.WorkflowExecutionQueue {
			execQ = &queues[i]
			break
		}
	}

	if execQ == nil {
		execCode := fmt.Sprintf("wf-exec-%s", def.Code)
		if qByCode, _ := queueRepo.GetQueueByCode(execCode, vns, now); qByCode != nil {
			execQ = qByCode
		} else if qByCodeDef, _ := queueRepo.GetQueueByCode(execCode, "default", now); qByCodeDef != nil {
			execQ = qByCodeDef
		}
	}

	if execQ == nil {
		commandResult.Error = fmt.Sprintf("execution queue not found for workflow definition %s (%s)", def.ID, def.Code)
		return *commandResult
	}

	if execQ.Type != models.WorkflowExecutionQueue || execQ.WorkflowDefinitionID != def.ID {
		execQ.Type = models.WorkflowExecutionQueue
		execQ.WorkflowDefinitionID = def.ID
		queueRepo.UpdateQueue(execQ, now)
	}

	if execQ.DesiredPriorityThresholds == nil {
		execQ.DesiredPriorityThresholds = map[int]int{0: 0}
		execQ.PriorityThresholds = map[int]int{0: 0}
		queueRepo.UpdateQueue(execQ, now)
	}

	msgID := strings.ReplaceAll(uuid.New().String(), "-", "")
	enqueueCmd := &queue.EnqueueCommand{
		Messages: []models.QueueMessage{
			{
				ID:          msgID,
				MessageID:   msgID,
				QueueID:     execQ.ID,
				VNamespace:  vns,
				Content:     []byte(fmt.Sprintf(`{"executionId":"%s","tokenId":"%s"}`, execution.ID, initialToken.ID)),
				ContentType: "application/json",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		CF:  cmd.CF,
		CFS: cmd.CFS,
	}
	enqueueRes := enqueueCmd.Execute(uow, now)
	if enqueueRes.Error != "" {
		commandResult.Error = fmt.Sprintf("failed to enqueue execution token message: %s", enqueueRes.Error)
		return *commandResult
	}

	commandResult.Result = execution
	return *commandResult
}
