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
	"github.com/rs/zerolog/log"
)

func init() {
	gob.Register(ResumeWaitEventCommand{})
}

type ResumeWaitEventCommand struct {
	WaitingEventID string
	Payload        map[string]interface{}
	CF             string
	CFS            string
}

func (cmd *ResumeWaitEventCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if strings.TrimSpace(cmd.WaitingEventID) == "" {
		commandResult.Error = "WaitingEventID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	waitingEventRepo, err := db.NewWaitingEventRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	execRepo, err := db.NewWorkflowExecutionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	tokenRepo, err := db.NewExecutionTokenRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	defRepo, err := db.NewWorkflowDefinitionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// 1. Fetch WaitingEvent
	waitingEvent, err := waitingEventRepo.GetWaitingEventByID(cmd.WaitingEventID, now)
	if err != nil || waitingEvent == nil {
		commandResult.Error = fmt.Sprintf("waiting event not found: %s", cmd.WaitingEventID)
		return *commandResult
	}

	// 2. Fetch WorkflowExecution
	execution, err := execRepo.GetWorkflowExecutionByID(waitingEvent.WorkflowExecutionID, now)
	if err != nil || execution == nil {
		commandResult.Error = fmt.Sprintf("workflow execution not found: %s", waitingEvent.WorkflowExecutionID)
		return *commandResult
	}

	// 3. Fetch ExecutionToken
	token, err := tokenRepo.GetExecutionTokenByID(waitingEvent.ExecutionTokenID, now)
	if err != nil || token == nil {
		commandResult.Error = fmt.Sprintf("execution token not found: %s", waitingEvent.ExecutionTokenID)
		return *commandResult
	}

	// 4. Fetch WorkflowDefinition
	def, err := defRepo.GetWorkflowDefinitionByID(waitingEvent.WorkflowDefinitionID, now)
	if err != nil || def == nil {
		commandResult.Error = fmt.Sprintf("workflow definition not found: %s", waitingEvent.WorkflowDefinitionID)
		return *commandResult
	}

	// Validate incoming payload against ExpectedInput schema
	if err := bpmn.ValidateEventInput(waitingEvent.ExpectedInput, cmd.Payload); err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// 5. Injects/merges validated values into the Token's global variables context
	if execution.StateData == nil {
		execution.StateData = make(map[string]interface{})
	}
	if cmd.Payload != nil {
		for k, v := range cmd.Payload {
			execution.StateData[k] = v
		}
	}
	if token.ScopeVariables == nil {
		token.ScopeVariables = make(map[string]interface{})
	}
	if cmd.Payload != nil {
		for k, v := range cmd.Payload {
			token.ScopeVariables[k] = v
		}
	}
	execution.UpdatedAt = now
	execRepo.UpdateWorkflowExecution(execution, now)

	// 6. Deletes the waiting_events record
	if _, delErr := waitingEventRepo.DeleteWaitingEvent(waitingEvent.ID, now); delErr != nil {
		log.Warn().Err(delErr).Str("waitingEventID", waitingEvent.ID).Msg("⚠️ Failed to delete waiting event record")
	}

	// 7. Parse BPMN and resolve outgoing flows from the current wait state node
	bpmnModel, err := bpmn.ParseBPMN(def.Payload)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to parse BPMN: %s", err.Error())
		return *commandResult
	}

	outgoing := bpmnModel.GetOutgoingFlows(token.CurrentNodeID)

	if len(outgoing) > 0 {
		token.CurrentNodeID = outgoing[0].TargetRef
		token.Status = models.ExecutionTokenStatusActive
		token.UpdatedAt = now
		tokenRepo.UpdateExecutionToken(token, now)

		log.Info().
			Str("executionID", execution.ID).
			Str("tokenID", token.ID).
			Str("resumedFromNode", waitingEvent.EventID).
			Str("targetNodeID", token.CurrentNodeID).
			Msg("▶️ Resumed wait state token, advancing to next flow target")

		// Re-queue token into WorkflowExecutionQueue so workers can dequeue
		allQs, _ := queueRepo.GetQueuesByWorkflowDefinitionID(def.ID, now)
		var execQ *models.Queue
		for i := range allQs {
			if allQs[i].Type == models.WorkflowExecutionQueue {
				execQ = &allQs[i]
				break
			}
		}
		if execQ == nil {
			execCode := fmt.Sprintf("wf-exec-%s", def.Code)
			if qByCode, _ := queueRepo.GetQueueByCode(execCode, execution.VNamespace, now); qByCode != nil {
				execQ = qByCode
			} else if qByCodeDef, _ := queueRepo.GetQueueByCode(execCode, "default", now); qByCodeDef != nil {
				execQ = qByCodeDef
			}
		}

		if execQ != nil {
			msgID := strings.ReplaceAll(uuid.New().String(), "-", "")
			enqueueCmd := &queue.EnqueueCommand{
				Messages: []models.QueueMessage{
					{
						ID:          msgID,
						MessageID:   msgID,
						QueueID:     execQ.ID,
						VNamespace:  execution.VNamespace,
						Content:     []byte(fmt.Sprintf(`{"executionId":"%s","tokenId":"%s"}`, execution.ID, token.ID)),
						ContentType: "application/json",
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				},
				CF:  cmd.CF,
				CFS: cmd.CFS,
			}
			enqueueCmd.Execute(uow, now)
		}

		// Also advance token synchronously to evaluate immediately downstream nodes
		advCmd := &AdvanceTokenCommand{
			ExecutionID: execution.ID,
			TokenID:     token.ID,
			OutputData:  cmd.Payload,
			CF:          cmd.CF,
			CFS:         cmd.CFS,
		}
		return advCmd.Execute(uow, now)
	}

	// No outgoing flows: complete token
	token.Status = models.ExecutionTokenStatusCompleted
	token.UpdatedAt = now
	tokenRepo.UpdateExecutionToken(token, now)

	// Check if all tokens completed
	allTokens, _ := tokenRepo.GetTokensByExecutionID(execution.ID, now)
	hasPendingOrActive := false
	for _, tok := range allTokens {
		if tok.Status == models.ExecutionTokenStatusActive || tok.Status == models.ExecutionTokenStatusWaiting {
			hasPendingOrActive = true
			break
		}
	}
	if !hasPendingOrActive {
		execution.Status = models.WorkflowExecutionStatusCompleted
		execution.CompletedAt = &now
		execution.UpdatedAt = now
		execRepo.UpdateWorkflowExecution(execution, now)
	}

	commandResult.Result = execution
	return *commandResult
}
