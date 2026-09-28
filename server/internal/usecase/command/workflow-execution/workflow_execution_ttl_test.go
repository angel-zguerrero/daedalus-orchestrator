package workflow_execution_test

import (
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	workflowDefCommand "deadalus-orch/server/internal/usecase/command/workflow-definition"
	workflowExecCommand "deadalus-orch/server/internal/usecase/command/workflow-execution"
	"deadalus-orch/shared/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ttlWorkflowBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_TTL" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_TTL" isExecutable="true">
    <bpmn:startEvent id="Start_1" name="Start">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="Start_1" targetRef="Task_1" />
    <bpmn:task id="Task_1" name="Process Step">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:task>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Task_1" targetRef="UserTask_1" />
    <bpmn:userTask id="UserTask_1" name="Approval Wait">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:sequenceFlow id="Flow_3" sourceRef="UserTask_1" targetRef="End_1" />
    <bpmn:endEvent id="End_1" name="End">
      <bpmn:incoming>Flow_3</bpmn:incoming>
    </bpmn:endEvent>
  </bpmn:process>
</bpmn:definitions>`

func TestWorkflowExecutionTTL_HelperCalculation(t *testing.T) {
	// 100 + 30% = 130
	assert.Equal(t, int64(130), models.CalculateWorkflowExecutionTTL(100))

	// 10 + 30% = 13
	assert.Equal(t, int64(13), models.CalculateWorkflowExecutionTTL(10))

	// 3600 + 30% = 4680
	assert.Equal(t, int64(4680), models.CalculateWorkflowExecutionTTL(3600))

	// Default fallback: 86400 * 1.30 = 112320
	assert.Equal(t, int64(112320), models.CalculateWorkflowExecutionTTL(0))
	assert.Equal(t, int64(112320), models.CalculateWorkflowExecutionTTL(-10))

	// Model method
	defWithDuration := &models.WorkflowDefinition{MaxDurationSeconds: 200}
	assert.Equal(t, int64(260), defWithDuration.CalculateExecutionTTL())

	defWithZero := &models.WorkflowDefinition{MaxDurationSeconds: 0}
	assert.Equal(t, int64(112320), defWithZero.CalculateExecutionTTL())
}

func TestWorkflowExecutionTTL_StartAndAdvancePropagation(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()
	testID := "wf_ttl_propagation_test"

	maxDuration := int32(200)
	expectedTTL := int64(260) // 200 + 30%

	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:                 testID,
			Code:               "CODE_" + testID,
			VNamespace:         "default",
			Name:               "TTL Propagation Test",
			Version:            1,
			MaxDurationSeconds: maxDuration,
			Payload:            []byte(ttlWorkflowBPMN),
			IsActive:           true,
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		ExecQueue: models.Queue{ID: "q_exec_" + testID, Type: models.WorkflowExecutionQueue, WorkflowDefinitionID: testID, Code: "wf-exec-CODE_" + testID},
		ActQueue:  models.Queue{ID: "q_act_" + testID, Type: models.WorkflowActivityQueue, WorkflowDefinitionID: testID, Code: "wf-act-CODE_" + testID},
		CF:        "default",
		CFS:       "default",
	}

	createRes := defCmd.Execute(uow, now)
	require.Empty(t, createRes.Error)
	require.NoError(t, uow.Commit())

	idFactory := &db.DeterministicIDGeneratorFactory{}

	// 1. Start execution
	startUow := db.NewUnitOfWork(store, nil)
	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		WorkflowDefinitionID: testID,
		Input:                map[string]interface{}{"data": "test"},
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	startRes := startCmd.Execute(startUow, now)
	require.Empty(t, startRes.Error)
	require.NoError(t, startUow.Commit())

	exec := startRes.Result.(*models.WorkflowExecution)
	assert.Equal(t, expectedTTL, exec.TTL, "WorkflowExecution in start response must have expected TTL")

	// Verify WorkflowExecution in DB
	checkUow := db.NewUnitOfWork(store, nil)
	execRepo, err := db.NewWorkflowExecutionRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)
	dbExec, err := execRepo.GetWorkflowExecutionByID(exec.ID, now)
	require.NoError(t, err)
	require.NotNil(t, dbExec)
	assert.Equal(t, expectedTTL, dbExec.TTL, "WorkflowExecution in DB must have expected TTL")

	// Verify ExecutionToken in DB
	tokenRepo, err := db.NewExecutionTokenRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)
	tokens, err := tokenRepo.GetActiveTokensByExecutionID(exec.ID, now)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	tokenID := tokens[0].ID
	assert.Equal(t, expectedTTL, tokens[0].TTL, "Initial ExecutionToken in DB must have expected TTL")

	// 2. Advance token to ServiceTask_1 (creates WorkflowJob)
	advUow := db.NewUnitOfWork(store, nil)
	advCmd := &workflowExecCommand.AdvanceTokenCommand{
		ExecutionID: exec.ID,
		TokenID:     tokenID,
		CF:          "default",
		CFS:         "default",
	}
	advRes := advCmd.Execute(advUow, now)
	require.Empty(t, advRes.Error)
	require.NoError(t, advUow.Commit())

	// Verify WorkflowJob created with TTL
	jobCheckUow := db.NewUnitOfWork(store, nil)
	jobRepo, err := db.NewWorkflowJobRepository(jobCheckUow, idFactory, "default", "default")
	require.NoError(t, err)
	jobs, err := jobRepo.GetJobsByExecutionID(exec.ID, now)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, expectedTTL, jobs[0].TTL, "WorkflowJob created for ServiceTask must have expected TTL")

	// 3. Complete the job to advance token to UserTask_1 (creates WaitingEvent)
	completeUow := db.NewUnitOfWork(store, nil)
	completeCmd := &workflowExecCommand.CompleteJobCommand{
		JobID:      jobs[0].ID,
		WorkerID:   "test_worker",
		OutputData: map[string]interface{}{"processed": true},
		CF:         "default",
		CFS:        "default",
	}
	completeRes := completeCmd.Execute(completeUow, now)
	require.Empty(t, completeRes.Error)
	require.NoError(t, completeUow.Commit())

	// Verify WaitingEvent created with TTL
	waitCheckUow := db.NewUnitOfWork(store, nil)
	waitRepo, err := db.NewWaitingEventRepository(waitCheckUow, idFactory, "default", "default")
	require.NoError(t, err)
	waitingEvents, err := waitRepo.GetWaitingEventsByExecutionID(exec.ID, now)
	require.NoError(t, err)
	require.Len(t, waitingEvents, 1)
	assert.Equal(t, expectedTTL, waitingEvents[0].TTL, "WaitingEvent created for UserTask must have expected TTL")

	// Verify updated token still has TTL
	tokenRepoCheck, err := db.NewExecutionTokenRepository(waitCheckUow, idFactory, "default", "default")
	require.NoError(t, err)
	currentTok, err := tokenRepoCheck.GetExecutionTokenByID(tokenID, now)
	require.NoError(t, err)
	require.NotNil(t, currentTok)
	assert.Equal(t, expectedTTL, currentTok.TTL, "Paused ExecutionToken must preserve expected TTL")
}

func TestWorkflowExecutionTTL_FallbackWhenMaxDurationZero(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()
	testID := "wf_ttl_zero_duration_test"

	expectedDefaultTTL := int64(112320) // 86400 * 1.30

	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:                 testID,
			Code:               "CODE_" + testID,
			VNamespace:         "default",
			Name:               "TTL Zero Duration Test",
			Version:            1,
			MaxDurationSeconds: 0, // Not configured or 0
			Payload:            []byte(ttlWorkflowBPMN),
			IsActive:           true,
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		ExecQueue: models.Queue{ID: "q_exec_" + testID, Type: models.WorkflowExecutionQueue, WorkflowDefinitionID: testID, Code: "wf-exec-CODE_" + testID},
		ActQueue:  models.Queue{ID: "q_act_" + testID, Type: models.WorkflowActivityQueue, WorkflowDefinitionID: testID, Code: "wf-act-CODE_" + testID},
		CF:        "default",
		CFS:       "default",
	}

	createRes := defCmd.Execute(uow, now)
	require.Empty(t, createRes.Error)
	require.NoError(t, uow.Commit())

	idFactory := &db.DeterministicIDGeneratorFactory{}

	startUow := db.NewUnitOfWork(store, nil)
	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		WorkflowDefinitionID: testID,
		Input:                map[string]interface{}{"data": "test"},
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	startRes := startCmd.Execute(startUow, now)
	require.Empty(t, startRes.Error)
	require.NoError(t, startUow.Commit())

	exec := startRes.Result.(*models.WorkflowExecution)
	assert.Equal(t, expectedDefaultTTL, exec.TTL, "WorkflowExecution must fallback to 112320 (24h + 30%)")

	checkUow := db.NewUnitOfWork(store, nil)
	tokenRepo, err := db.NewExecutionTokenRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)
	tokens, err := tokenRepo.GetActiveTokensByExecutionID(exec.ID, now)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, expectedDefaultTTL, tokens[0].TTL, "Initial ExecutionToken must fallback to 112320")
}

func TestWorkflowExecutionTTL_StorageExpiration(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()
	testID := "wf_ttl_expiration_test"

	maxDuration := int32(10)
	expectedTTL := int64(13) // 10 + 30% = 13s

	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:                 testID,
			Code:               "CODE_" + testID,
			VNamespace:         "default",
			Name:               "TTL Expiration Test",
			Version:            1,
			MaxDurationSeconds: maxDuration,
			Payload:            []byte(ttlWorkflowBPMN),
			IsActive:           true,
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		ExecQueue: models.Queue{ID: "q_exec_" + testID, Type: models.WorkflowExecutionQueue, WorkflowDefinitionID: testID, Code: "wf-exec-CODE_" + testID},
		ActQueue:  models.Queue{ID: "q_act_" + testID, Type: models.WorkflowActivityQueue, WorkflowDefinitionID: testID, Code: "wf-act-CODE_" + testID},
		CF:        "default",
		CFS:       "default",
	}

	createRes := defCmd.Execute(uow, now)
	require.Empty(t, createRes.Error)
	require.NoError(t, uow.Commit())

	idFactory := &db.DeterministicIDGeneratorFactory{}

	startUow := db.NewUnitOfWork(store, nil)
	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		WorkflowDefinitionID: testID,
		Input:                map[string]interface{}{"data": "test"},
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	startRes := startCmd.Execute(startUow, now)
	require.Empty(t, startRes.Error)
	require.NoError(t, startUow.Commit())

	exec := startRes.Result.(*models.WorkflowExecution)
	assert.Equal(t, expectedTTL, exec.TTL)

	// Before TTL expires: record exists
	checkUowBefore := db.NewUnitOfWork(store, nil)
	execRepoBefore, err := db.NewWorkflowExecutionRepository(checkUowBefore, idFactory, "default", "default")
	require.NoError(t, err)
	dbExec, err := execRepoBefore.GetWorkflowExecutionByID(exec.ID, now)
	require.NoError(t, err)
	require.NotNil(t, dbExec)

	// After TTL expires (14 seconds later): record is expired and Get returns nil
	expiredTime := now.Add(14 * time.Second)
	checkUowAfter := db.NewUnitOfWork(store, nil)
	execRepoAfter, err := db.NewWorkflowExecutionRepository(checkUowAfter, idFactory, "default", "default")
	require.NoError(t, err)
	expiredExec, err := execRepoAfter.GetWorkflowExecutionByID(exec.ID, expiredTime)
	require.NoError(t, err)
	assert.Nil(t, expiredExec, "WorkflowExecution must be expired and return nil after TTL duration")
}
