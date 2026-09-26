package workflow_execution_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	workflowExecCommand "deadalus-orch/server/internal/usecase/command/workflow-execution"
	"deadalus-orch/shared/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvanceTokenCommand_TimerIntermediateCatchEvent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "timer_eval_test_*")
	require.NoError(t, err)

	cfNames := []string{"default", db.AdminFC}
	store, err := db.CreatePebbleStore(tempDir, cfNames, []string{})
	require.NoError(t, err)
	t.Cleanup(func() {
		store.Close()
		os.RemoveAll(tempDir)
	})

	uow := db.NewUnitOfWork(store, nil)
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	cf := db.AdminFC
	cfs := db.AdminFCSector
	idFactory := &db.DeterministicIDGeneratorFactory{}

	// Create repositories
	defRepo, err := db.NewWorkflowDefinitionRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)
	execRepo, err := db.NewWorkflowExecutionRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)
	tokenRepo, err := db.NewExecutionTokenRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	bpmnXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  id="Definitions_TimerTest">
  <bpmn:process id="Process_TimerTest" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start" />
    <bpmn:intermediateCatchEvent id="Timer_Wait" name="Wait 5 Minutes">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
      <bpmn:timerEventDefinition id="TimerDef_1">
        <bpmn:timeDuration>5m</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:intermediateCatchEvent>
    <bpmn:endEvent id="EndEvent_1" name="End">
      <bpmn:incoming>Flow_2</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Timer_Wait" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Timer_Wait" targetRef="EndEvent_1" />
  </bpmn:process>
</bpmn:definitions>`

	// Create Definition
	def := &models.WorkflowDefinition{
		ID:         "def_timer_test",
		Code:       "wf-timer-test",
		Name:       "Timer Test Workflow",
		VNamespace: "default",
		Payload:    []byte(bpmnXML),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err = defRepo.CreateWorkflowDefinition(def, now)
	require.NoError(t, err)

	// Create Execution Queue
	execQ := &models.Queue{
		ID:                   "q_exec_timer_test",
		Code:                 "wf-exec-wf-timer-test",
		VNamespace:           "default",
		Type:                 models.WorkflowExecutionQueue,
		WorkflowDefinitionID: def.ID,
		MaxAttempts:          3,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	_, err = queueRepo.CreateQueue(execQ, now)
	require.NoError(t, err)

	// Create Execution
	exec := &models.WorkflowExecution{
		ID:                   "exec_timer_1",
		WorkflowDefinitionID: def.ID,
		VNamespace:           "default",
		Status:               models.WorkflowExecutionStatusRunning,
		StateData:            map[string]interface{}{},
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	_, err = execRepo.CreateWorkflowExecution(exec, now)
	require.NoError(t, err)

	// Create Token at StartEvent_1
	token := &models.ExecutionToken{
		ID:                   "tok_timer_1",
		WorkflowExecutionID: exec.ID,
		WorkflowDefinitionID: def.ID,
		VNamespace:           "default",
		CurrentNodeID:        "StartEvent_1",
		Status:               models.ExecutionTokenStatusActive,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	_, err = tokenRepo.CreateExecutionToken(token, now)
	require.NoError(t, err)

	// Advance Token from Start to Timer_Wait
	cmd := &workflowExecCommand.AdvanceTokenCommand{
		ExecutionID: exec.ID,
		TokenID:     token.ID,
		CF:          cf,
		CFS:         cfs,
	}
	res := cmd.Execute(uow, now)
	require.Empty(t, res.Error)

	// Verify token is now waiting at Timer_Wait and ScheduledJob was created for +5m (12:05:00 UTC)
	updatedTok, err := tokenRepo.GetExecutionTokenByID(token.ID, now)
	require.NoError(t, err)
	assert.Equal(t, "Timer_Wait", updatedTok.CurrentNodeID)
	assert.Equal(t, models.ExecutionTokenStatusWaiting, updatedTok.Status)

	// Check ScheduledJob created
	sjRepo, err := db.NewScheduledJobRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	schedJobID := fmt.Sprintf("timer_%s_%s_%d", exec.ID, token.ID, now.Add(5*time.Minute).Unix())
	job, err := sjRepo.GetScheduledJobByID(schedJobID, now)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, now.Add(5*time.Minute).Unix(), job.NextRunAt.Unix())

	var payload map[string]interface{}
	err = json.Unmarshal([]byte(job.Content), &payload)
	require.NoError(t, err)
	assert.Equal(t, exec.ID, payload["executionId"])
	assert.Equal(t, token.ID, payload["tokenId"])

	// Now simulate time advancement after timer expires (now = 12:05:01 UTC)
	laterTime := now.Add(5*time.Minute + 1*time.Second)
	cmdLater := &workflowExecCommand.AdvanceTokenCommand{
		ExecutionID: exec.ID,
		TokenID:     token.ID,
		CF:          cf,
		CFS:         cfs,
	}
	resLater := cmdLater.Execute(uow, laterTime)
	require.Empty(t, resLater.Error)

	// Verify execution completed as token moved past Timer_Wait to EndEvent_1
	updatedExec, err := execRepo.GetWorkflowExecutionByID(exec.ID, laterTime)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowExecutionStatusCompleted, updatedExec.Status)
}
