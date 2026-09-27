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

// BPMN with an Exclusive Gateway where conditions are not satisfied and no default flow exists
const failureWorkflowBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_FailTest" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_FailTest" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Gateway_Fail" />
    <bpmn:exclusiveGateway id="Gateway_Fail" name="Failing Gateway">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_Cond1</bpmn:outgoing>
      <bpmn:outgoing>Flow_Cond2</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:sequenceFlow id="Flow_Cond1" sourceRef="Gateway_Fail" targetRef="EndEvent_1">
      <bpmn:conditionExpression>${status == 'APPROVED'}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_Cond2" sourceRef="Gateway_Fail" targetRef="EndEvent_1">
      <bpmn:conditionExpression>${status == 'PENDING'}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:endEvent id="EndEvent_1" name="End">
      <bpmn:incoming>Flow_Cond1</bpmn:incoming>
      <bpmn:incoming>Flow_Cond2</bpmn:incoming>
    </bpmn:endEvent>
  </bpmn:process>
</bpmn:definitions>`

func TestActivityAndWorkflowMarkedWithFailureOnError(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()
	testID := "wf_failure_marking_test"

	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:         testID,
			Code:       "CODE_" + testID,
			VNamespace: "default",
			Name:       "Failure Marking Test",
			Version:    1,
			Payload:    []byte(failureWorkflowBPMN),
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
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

	// Start execution with status='REJECTED' -> neither condition matches and no default flow exists
	startUow := db.NewUnitOfWork(store, nil)
	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		WorkflowDefinitionID: testID,
		Input:                map[string]interface{}{"status": "REJECTED"},
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	res := startCmd.Execute(startUow, now)
	require.Empty(t, res.Error)
	require.NoError(t, startUow.Commit())

	exec := res.Result.(*models.WorkflowExecution)

	// Advance token from StartEvent_1 to Gateway_Fail
	tokenUow := db.NewUnitOfWork(store, nil)
	tokenRepo, err := db.NewExecutionTokenRepository(tokenUow, idFactory, "default", "default")
	require.NoError(t, err)
	activeTokens, err := tokenRepo.GetActiveTokensByExecutionID(exec.ID, now)
	require.NoError(t, err)
	require.Len(t, activeTokens, 1)

	advUow := db.NewUnitOfWork(store, nil)
	advCmd := &workflowExecCommand.AdvanceTokenCommand{
		ExecutionID: exec.ID,
		TokenID:     activeTokens[0].ID,
		CF:          "default",
		CFS:         "default",
	}
	advRes := advCmd.Execute(advUow, now)
	// The command should report the error
	require.NotEmpty(t, advRes.Error)
	require.NoError(t, advUow.Commit())

	// Verify that the WorkflowExecution is marked as FAILED with error message and CompletedAt
	checkUow := db.NewUnitOfWork(store, nil)
	execRepo, err := db.NewWorkflowExecutionRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)
	updatedExec, err := execRepo.GetWorkflowExecutionByID(exec.ID, now)
	require.NoError(t, err)
	require.NotNil(t, updatedExec)

	assert.Equal(t, models.WorkflowExecutionStatusFailed, updatedExec.Status, "WorkflowExecution status must be failed")
	assert.NotEmpty(t, updatedExec.Error, "WorkflowExecution error must be recorded")
	assert.NotNil(t, updatedExec.CompletedAt, "WorkflowExecution CompletedAt must be set")

	// Verify that the activity job was created and marked as FAILED with error message
	jobRepo, err := db.NewWorkflowJobRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)
	jobs, err := jobRepo.GetJobsByExecutionID(exec.ID, now)
	require.NoError(t, err)
	require.NotEmpty(t, jobs, "A WorkflowJob must be recorded for the failed activity")

	var failedJob *models.WorkflowJob
	for i := range jobs {
		if jobs[i].ActivityID == "Gateway_Fail" {
			failedJob = &jobs[i]
			break
		}
	}
	require.NotNil(t, failedJob, "Gateway_Fail must have a corresponding job")
	assert.Equal(t, models.WorkflowJobStatusFailed, failedJob.Status, "Activity job status must be failed")
	assert.NotEmpty(t, failedJob.Error, "Activity job error must be recorded")
	assert.NotNil(t, failedJob.CompletedAt, "Activity job CompletedAt must be set")

	// Verify that the token is cancelled
	tokenRepoCheck, err := db.NewExecutionTokenRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)
	token, err := tokenRepoCheck.GetExecutionTokenByID(activeTokens[0].ID, now)
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionTokenStatusCancelled, token.Status, "Token status must be cancelled")
}

func TestCompleteJobCommand_FailureMarksActivityAndWorkflowExecution(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()
	testID := "wf_complete_job_fail_test"

	idFactory := &db.DeterministicIDGeneratorFactory{}

	// Setup execution record
	execRepo, _ := db.NewWorkflowExecutionRepository(uow, idFactory, "default", "default")
	exec := &models.WorkflowExecution{
		ID:                   "exec_1",
		WorkflowDefinitionID: testID,
		Status:               models.WorkflowExecutionStatusRunning,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	_, _ = execRepo.CreateWorkflowExecution(exec, now)

	// Setup token record
	tokenRepo, _ := db.NewExecutionTokenRepository(uow, idFactory, "default", "default")
	token := &models.ExecutionToken{
		ID:                  "tok_1",
		WorkflowExecutionID: "exec_1",
		CurrentNodeID:       "Activity_1",
		Status:              models.ExecutionTokenStatusWaiting,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	_, _ = tokenRepo.CreateExecutionToken(token, now)

	// Setup job record
	jobRepo, _ := db.NewWorkflowJobRepository(uow, idFactory, "default", "default")
	job := &models.WorkflowJob{
		ID:                  "job_1",
		WorkflowExecutionID: "exec_1",
		ExecutionTokenID:    "tok_1",
		ActivityID:          "Activity_1",
		ActivityName:        "Test Activity",
		Status:              models.WorkflowJobStatusPending,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	_, _ = jobRepo.CreateWorkflowJob(job, now)
	require.NoError(t, uow.Commit())

	// Complete job with error
	compUow := db.NewUnitOfWork(store, nil)
	compCmd := &workflowExecCommand.CompleteJobCommand{
		JobID:    "job_1",
		WorkerID: "worker_1",
		Error:    "HTTP connection timeout to downstream service",
		CF:       "default",
		CFS:      "default",
	}
	res := compCmd.Execute(compUow, now)
	require.Empty(t, res.Error)
	require.NoError(t, compUow.Commit())

	// Verify job is failed with error and CompletedAt set
	verifyUow := db.NewUnitOfWork(store, nil)
	verifyJobRepo, _ := db.NewWorkflowJobRepository(verifyUow, idFactory, "default", "default")
	updatedJob, _ := verifyJobRepo.GetWorkflowJobByID("job_1", now)
	require.NotNil(t, updatedJob)
	assert.Equal(t, models.WorkflowJobStatusFailed, updatedJob.Status)
	assert.Equal(t, "HTTP connection timeout to downstream service", updatedJob.Error)
	assert.NotNil(t, updatedJob.CompletedAt)

	// Verify execution is failed with error and CompletedAt set
	verifyExecRepo, _ := db.NewWorkflowExecutionRepository(verifyUow, idFactory, "default", "default")
	updatedExec, _ := verifyExecRepo.GetWorkflowExecutionByID("exec_1", now)
	require.NotNil(t, updatedExec)
	assert.Equal(t, models.WorkflowExecutionStatusFailed, updatedExec.Status)
	assert.Equal(t, "HTTP connection timeout to downstream service", updatedExec.Error)
	assert.NotNil(t, updatedExec.CompletedAt)

	// Verify token is cancelled
	verifyTokenRepo, _ := db.NewExecutionTokenRepository(verifyUow, idFactory, "default", "default")
	updatedToken, _ := verifyTokenRepo.GetExecutionTokenByID("tok_1", now)
	require.NotNil(t, updatedToken)
	assert.Equal(t, models.ExecutionTokenStatusCancelled, updatedToken.Status)
}
