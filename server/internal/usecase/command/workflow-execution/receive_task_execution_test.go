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

const receiveTaskWorkflowBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_ReceiveTest" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_ReceiveTest" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ReceiveTask_1" />
    <bpmn:receiveTask id="ReceiveTask_1" name="Wait for Callback">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:receiveTask>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ReceiveTask_1" targetRef="UserTask_1" />
    <bpmn:userTask id="UserTask_1" name="Approve Request">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:sequenceFlow id="Flow_3" sourceRef="UserTask_1" targetRef="Task_1" />
    <bpmn:task id="Task_1" name="Standard Task">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:task>
    <bpmn:sequenceFlow id="Flow_4" sourceRef="Task_1" targetRef="EndEvent_1" />
    <bpmn:endEvent id="EndEvent_1" name="End">
      <bpmn:incoming>Flow_4</bpmn:incoming>
    </bpmn:endEvent>
  </bpmn:process>
</bpmn:definitions>`

func TestReceiveTaskWorkflowExecution_AdvancesCleanlyLikeUserTask(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()
	testID := "wf_receive_task_test"

	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:         testID,
			Code:       "CODE_" + testID,
			VNamespace: "default",
			Name:       "Receive Task Workflow Test",
			Version:    1,
			Payload:    []byte(receiveTaskWorkflowBPMN),
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

	// Start execution
	startUow := db.NewUnitOfWork(store, nil)
	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		WorkflowDefinitionID: testID,
		Input:                map[string]interface{}{"orderId": "12345"},
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	res := startCmd.Execute(startUow, now)
	require.Empty(t, res.Error)
	require.NoError(t, startUow.Commit())

	exec := res.Result.(*models.WorkflowExecution)
	require.Equal(t, models.WorkflowExecutionStatusRunning, exec.Status)

	executedActivities := make(map[string]bool)

	// Step through execution: advance tokens and complete activity jobs
	for step := 0; step < 20; step++ {
		execCheckUow := db.NewUnitOfWork(store, nil)
		execRepo, err := db.NewWorkflowExecutionRepository(execCheckUow, idFactory, "default", "default")
		require.NoError(t, err)
		currentExec, err := execRepo.GetWorkflowExecutionByID(exec.ID, now)
		require.NoError(t, err)

		if currentExec.Status == models.WorkflowExecutionStatusCompleted || currentExec.Status == models.WorkflowExecutionStatusFailed {
			exec = currentExec
			break
		}

		// 1. Advance all active tokens
		tokenUow := db.NewUnitOfWork(store, nil)
		tokenRepo, err := db.NewExecutionTokenRepository(tokenUow, idFactory, "default", "default")
		require.NoError(t, err)
		activeTokens, _ := tokenRepo.GetActiveTokensByExecutionID(exec.ID, now)

		for _, tok := range activeTokens {
			advUow := db.NewUnitOfWork(store, nil)
			advCmd := &workflowExecCommand.AdvanceTokenCommand{
				ExecutionID: exec.ID,
				TokenID:     tok.ID,
				CF:          "default",
				CFS:         "default",
			}
			advRes := advCmd.Execute(advUow, now)
			if advRes.Error == "" {
				_ = advUow.Commit()
			} else {
				t.Fatalf("AdvanceTokenCommand failed on node %s: %s", tok.CurrentNodeID, advRes.Error)
			}
		}

		// 2. Find and resume any active waiting events
		waitUow := db.NewUnitOfWork(store, nil)
		waitRepo, err := db.NewWaitingEventRepository(waitUow, idFactory, "default", "default")
		require.NoError(t, err)
		waitingEvents, _ := waitRepo.GetWaitingEventsByExecutionID(exec.ID, now)

		progressMade := false
		for _, w := range waitingEvents {
			executedActivities[w.EventID] = true
			t.Logf("Resuming wait event for activity: %s of type %s", w.EventID, w.Type)

			resumeCmd := &workflowExecCommand.ResumeWaitEventCommand{
				WaitingEventID: w.ID,
				Payload: map[string]interface{}{
					"status": "SUCCESS",
				},
				CF:  "default",
				CFS: "default",
			}
			rUow := db.NewUnitOfWork(store, nil)
			resRes := resumeCmd.Execute(rUow, now)
			require.Empty(t, resRes.Error)
			require.NoError(t, rUow.Commit())
			progressMade = true
		}

		// 3. Find and complete any pending jobs
		jobUow := db.NewUnitOfWork(store, nil)
		jobRepo, err := db.NewWorkflowJobRepository(jobUow, idFactory, "default", "default")
		require.NoError(t, err)
		jobs, _ := jobRepo.GetJobsByExecutionID(exec.ID, now)

		for idx := range jobs {
			j := &jobs[idx]
			if j.Status == models.WorkflowJobStatusPending {
				executedActivities[j.ActivityID] = true
				t.Logf("Completing job for activity: %s (%s) of type %s", j.ActivityID, j.ActivityName, j.ActivityType)

				compCmd := &workflowExecCommand.CompleteJobCommand{
					JobID:    j.ID,
					WorkerID: "test-worker",
					OutputData: map[string]interface{}{
						"status": "SUCCESS",
					},
					CF:  "default",
					CFS: "default",
				}
				cUow := db.NewUnitOfWork(store, nil)
				compRes := compCmd.Execute(cUow, now)
				require.Empty(t, compRes.Error)
				require.NoError(t, cUow.Commit())
				progressMade = true
			}
		}

		if len(activeTokens) == 0 && !progressMade {
			break
		}
	}

	assert.Equal(t, models.WorkflowExecutionStatusCompleted, exec.Status)
	assert.Empty(t, exec.Error)

	// Verify all activities in sequence were executed including ReceiveTask_1, UserTask_1, and Task_1
	assert.True(t, executedActivities["ReceiveTask_1"], "ReceiveTask_1 should have been executed")
	assert.True(t, executedActivities["UserTask_1"], "UserTask_1 should have been executed")
	assert.True(t, executedActivities["Task_1"], "Task_1 should have been executed")
}

const unsupportedTaskBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_UnsupportedTest" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_UnsupportedTest" isExecutable="true">
    <bpmn:startEvent id="Start_1">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="Start_1" targetRef="SendTask_1" />
    <bpmn:sendTask id="SendTask_1" name="Send Confirmation">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:sendTask>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="SendTask_1" targetRef="End_1" />
    <bpmn:endEvent id="End_1">
      <bpmn:incoming>Flow_2</bpmn:incoming>
    </bpmn:endEvent>
  </bpmn:process>
</bpmn:definitions>`

func TestUnsupportedTaskType_FailsExplicitlyAndMarksActivityAndWorkflow(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()
	testID := "wf_unsupported_task_test"

	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:         testID,
			Code:       "CODE_" + testID,
			VNamespace: "default",
			Name:       "Unsupported Task Test",
			Version:    1,
			Payload:    []byte(unsupportedTaskBPMN),
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

	startUow := db.NewUnitOfWork(store, nil)
	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		WorkflowDefinitionID: testID,
		Input:                map[string]interface{}{},
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	res := startCmd.Execute(startUow, now)
	require.Empty(t, res.Error)
	require.NoError(t, startUow.Commit())

	exec := res.Result.(*models.WorkflowExecution)

	// Advance token from Start_1 to SendTask_1 -> SendTask is unsupported and must fail
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
	require.NotEmpty(t, advRes.Error, "AdvanceTokenCommand must return an error for unsupported task type")
	assert.Contains(t, advRes.Error, "unsupported or unrecognized element type \"sendTask\"")
	require.NoError(t, advUow.Commit())

	// Verify WorkflowExecution is marked as FAILED
	checkUow := db.NewUnitOfWork(store, nil)
	execRepo, err := db.NewWorkflowExecutionRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)
	updatedExec, err := execRepo.GetWorkflowExecutionByID(exec.ID, now)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowExecutionStatusFailed, updatedExec.Status)
	assert.Contains(t, updatedExec.Error, "unsupported or unrecognized element type \"sendTask\"")
	assert.NotNil(t, updatedExec.CompletedAt)

	// Verify WorkflowJob for SendTask_1 was recorded as FAILED
	jobRepo, err := db.NewWorkflowJobRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)
	jobs, err := jobRepo.GetJobsByExecutionID(exec.ID, now)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "SendTask_1", jobs[0].ActivityID)
	assert.Equal(t, models.WorkflowJobStatusFailed, jobs[0].Status)
	assert.Contains(t, jobs[0].Error, "unsupported or unrecognized element type \"sendTask\"")
	assert.NotNil(t, jobs[0].CompletedAt)

	// Verify token was cancelled
	updatedToken, err := tokenRepo.GetExecutionTokenByID(activeTokens[0].ID, now)
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionTokenStatusCancelled, updatedToken.Status)
}

func TestServiceTask_FailsExplicitlyAsUnsupportedElementType(t *testing.T) {
	const serviceTaskBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_ServiceFail" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_ServiceFail" isExecutable="true">
    <bpmn:startEvent id="Start_1">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="Start_1" targetRef="Service_1" />
    <bpmn:serviceTask id="Service_1" name="Service Activity">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Service_1" targetRef="End_1" />
    <bpmn:endEvent id="End_1">
      <bpmn:incoming>Flow_2</bpmn:incoming>
    </bpmn:endEvent>
  </bpmn:process>
</bpmn:definitions>`

	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()
	testID := "wf_servicetask_fail_test"

	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:         testID,
			Code:       "CODE_" + testID,
			VNamespace: "default",
			Name:       "Service Task Fail Test",
			Version:    1,
			Payload:    []byte(serviceTaskBPMN),
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

	startUow := db.NewUnitOfWork(store, nil)
	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		WorkflowDefinitionID: testID,
		Input:                map[string]interface{}{},
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	res := startCmd.Execute(startUow, now)
	require.Empty(t, res.Error)
	require.NoError(t, startUow.Commit())

	exec := res.Result.(*models.WorkflowExecution)

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
	require.NotEmpty(t, advRes.Error)
	assert.Contains(t, advRes.Error, "unsupported or unrecognized element type \"serviceTask\"")
	require.NoError(t, advUow.Commit())

	checkUow := db.NewUnitOfWork(store, nil)
	execRepo, err := db.NewWorkflowExecutionRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)
	updatedExec, err := execRepo.GetWorkflowExecutionByID(exec.ID, now)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowExecutionStatusFailed, updatedExec.Status)
	assert.Contains(t, updatedExec.Error, "unsupported or unrecognized element type \"serviceTask\"")
	assert.NotNil(t, updatedExec.CompletedAt)
}
