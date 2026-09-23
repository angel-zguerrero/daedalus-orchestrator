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

const userProvidedBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start">
      <bpmn:extensionElements>
        <camunda:formData>
          <camunda:formField id="name" label="Nombre Persona" type="string">
            <camunda:validation>
              <camunda:constraint name="required" />
              <camunda:constraint name="minlength" config="2" />
              <camunda:constraint name="max" config="10" />
            </camunda:validation>
          </camunda:formField>
        </camunda:formData>
      </bpmn:extensionElements>
      <bpmn:outgoing>Flow_0qf5yy7</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:exclusiveGateway id="Gateway_19j8b1u" default="Flow_04bompz">
      <bpmn:incoming>Flow_0qf5yy7</bpmn:incoming>
      <bpmn:outgoing>Flow_1t8mt7w</bpmn:outgoing>
      <bpmn:outgoing>Flow_04bompz</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:sequenceFlow id="Flow_0qf5yy7" sourceRef="StartEvent_1" targetRef="Gateway_19j8b1u" />
    <bpmn:task id="Activity_1pxklqa" name="Proceso Pesado">
      <bpmn:incoming>Flow_1t8mt7w</bpmn:incoming>
      <bpmn:outgoing>Flow_05zqsj9</bpmn:outgoing>
    </bpmn:task>
    <bpmn:sequenceFlow id="Flow_1t8mt7w" sourceRef="Gateway_19j8b1u" targetRef="Activity_1pxklqa">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">name=="angel"</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:task id="Activity_0kpgn1l" name="Proceso Ligero">
      <bpmn:incoming>Flow_04bompz</bpmn:incoming>
      <bpmn:outgoing>Flow_1u05t8b</bpmn:outgoing>
    </bpmn:task>
    <bpmn:sequenceFlow id="Flow_04bompz" sourceRef="Gateway_19j8b1u" targetRef="Activity_0kpgn1l" />
    <bpmn:exclusiveGateway id="Gateway_0nzezff">
      <bpmn:incoming>Flow_05zqsj9</bpmn:incoming>
      <bpmn:incoming>Flow_1u05t8b</bpmn:incoming>
      <bpmn:outgoing>Flow_0gwh2x0</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:sequenceFlow id="Flow_05zqsj9" sourceRef="Activity_1pxklqa" targetRef="Gateway_0nzezff" />
    <bpmn:sequenceFlow id="Flow_1u05t8b" sourceRef="Activity_0kpgn1l" targetRef="Gateway_0nzezff" />
    <bpmn:endEvent id="Event_1ov7rax">
      <bpmn:incoming>Flow_12llkgj</bpmn:incoming>
      <bpmn:incoming>Flow_1uh3lh7</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_0gwh2x0" sourceRef="Gateway_0nzezff" targetRef="Gateway_09ikxb5" />
    <bpmn:parallelGateway id="Gateway_09ikxb5">
      <bpmn:incoming>Flow_0gwh2x0</bpmn:incoming>
      <bpmn:outgoing>Flow_1435ob3</bpmn:outgoing>
      <bpmn:outgoing>Flow_0duq87f</bpmn:outgoing>
    </bpmn:parallelGateway>
    <bpmn:sequenceFlow id="Flow_1435ob3" sourceRef="Gateway_09ikxb5" targetRef="Gateway_14fmreg" />
    <bpmn:inclusiveGateway id="Gateway_14fmreg" default="Flow_029k6e8">
      <bpmn:incoming>Flow_1435ob3</bpmn:incoming>
      <bpmn:outgoing>Flow_11b33sg</bpmn:outgoing>
      <bpmn:outgoing>Flow_0mfwhng</bpmn:outgoing>
      <bpmn:outgoing>Flow_029k6e8</bpmn:outgoing>
    </bpmn:inclusiveGateway>
    <bpmn:task id="Activity_1ewq0xm" name="Saludar por REdus">
      <bpmn:incoming>Flow_11b33sg</bpmn:incoming>
      <bpmn:outgoing>Flow_1ng351b</bpmn:outgoing>
    </bpmn:task>
    <bpmn:sequenceFlow id="Flow_11b33sg" sourceRef="Gateway_14fmreg" targetRef="Activity_1ewq0xm">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">name=="angel"</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:task id="Activity_1oe3o01" name="Saludsar por http">
      <bpmn:incoming>Flow_0mfwhng</bpmn:incoming>
      <bpmn:outgoing>Flow_0efzm1j</bpmn:outgoing>
    </bpmn:task>
    <bpmn:sequenceFlow id="Flow_0mfwhng" sourceRef="Gateway_14fmreg" targetRef="Activity_1oe3o01">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">name=="reina"</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:task id="Activity_1a0x9p7" name="Saludar por http def">
      <bpmn:incoming>Flow_029k6e8</bpmn:incoming>
      <bpmn:outgoing>Flow_1bq476y</bpmn:outgoing>
    </bpmn:task>
    <bpmn:sequenceFlow id="Flow_029k6e8" sourceRef="Gateway_14fmreg" targetRef="Activity_1a0x9p7" />
    <bpmn:sequenceFlow id="Flow_1ng351b" sourceRef="Activity_1ewq0xm" targetRef="Gateway_1xpgk4o" />
    <bpmn:inclusiveGateway id="Gateway_1xpgk4o">
      <bpmn:incoming>Flow_1ng351b</bpmn:incoming>
      <bpmn:incoming>Flow_0efzm1j</bpmn:incoming>
      <bpmn:incoming>Flow_1bq476y</bpmn:incoming>
      <bpmn:outgoing>Flow_12llkgj</bpmn:outgoing>
    </bpmn:inclusiveGateway>
    <bpmn:sequenceFlow id="Flow_0efzm1j" sourceRef="Activity_1oe3o01" targetRef="Gateway_1xpgk4o" />
    <bpmn:sequenceFlow id="Flow_1bq476y" sourceRef="Activity_1a0x9p7" targetRef="Gateway_1xpgk4o" />
    <bpmn:sequenceFlow id="Flow_12llkgj" sourceRef="Gateway_1xpgk4o" targetRef="Event_1ov7rax" />
    <bpmn:task id="Activity_0wfmm60" name="Leguear">
      <bpmn:incoming>Flow_0duq87f</bpmn:incoming>
      <bpmn:outgoing>Flow_1uh3lh7</bpmn:outgoing>
    </bpmn:task>
    <bpmn:sequenceFlow id="Flow_0duq87f" sourceRef="Gateway_09ikxb5" targetRef="Activity_0wfmm60" />
    <bpmn:sequenceFlow id="Flow_1uh3lh7" sourceRef="Activity_0wfmm60" targetRef="Event_1ov7rax" />
  </bpmn:process>
</bpmn:definitions>`

func runUserProvidedBPMNTest(t *testing.T, testID string, input map[string]interface{}) (map[string]bool, *models.WorkflowExecution) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()

	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:         testID,
			Code:       "CODE_" + testID,
			VNamespace: "default",
			Name:       "User Provided Diagram Test",
			Version:    1,
			Payload:    []byte(userProvidedBPMN),
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

	// Start Workflow Execution
	startUow := db.NewUnitOfWork(store, nil)
	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		WorkflowDefinitionID: testID,
		Input:                input,
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	res := startCmd.Execute(startUow, now)
	require.Empty(t, res.Error)
	require.NoError(t, startUow.Commit())

	exec := res.Result.(*models.WorkflowExecution)
	executedActivities := make(map[string]bool)

	// Simulation loop: advance active tokens and complete pending activity jobs
	for step := 0; step < 50; step++ {
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
			}
		}

		// 2. Find and complete any pending jobs
		jobUow := db.NewUnitOfWork(store, nil)
		jobRepo, err := db.NewWorkflowJobRepository(jobUow, idFactory, "default", "default")
		require.NoError(t, err)
		jobs, _ := jobRepo.GetJobsByExecutionID(exec.ID, now)

		progressMade := false
		for idx := range jobs {
			j := &jobs[idx]
			if j.Status == models.WorkflowJobStatusPending {
				executedActivities[j.ActivityID] = true
				t.Logf("Completing job for activity: %s (%s)", j.ActivityID, j.ActivityName)

				compCmd := &workflowExecCommand.CompleteJobCommand{
					JobID:    j.ID,
					WorkerID: "test-worker",
					CF:       "default",
					CFS:      "default",
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

	execUow := db.NewUnitOfWork(store, nil)
	execRepo, err := db.NewWorkflowExecutionRepository(execUow, idFactory, "default", "default")
	require.NoError(t, err)
	finalExec, _ := execRepo.GetWorkflowExecutionByID(exec.ID, now)

	return executedActivities, finalExec
}

func TestUserProvidedBPMN_NameAngel(t *testing.T) {
	executed, finalExec := runUserProvidedBPMNTest(t, "user_bpmn_angel", map[string]interface{}{"name": "angel"})

	t.Logf("Final execution status: %s, error: %s", finalExec.Status, finalExec.Error)
	assert.Equal(t, models.WorkflowExecutionStatusCompleted, finalExec.Status)
	assert.True(t, executed["Activity_1pxklqa"], "Proceso Pesado must be executed")
	assert.True(t, executed["Activity_0wfmm60"], "Leguear must be executed")
	assert.True(t, executed["Activity_1ewq0xm"], "Saludar por REdus must be executed")
	assert.False(t, executed["Activity_0kpgn1l"], "Proceso Ligero must NOT be executed")
}

func TestUserProvidedBPMN_NameReina(t *testing.T) {
	executed, finalExec := runUserProvidedBPMNTest(t, "user_bpmn_reina", map[string]interface{}{"name": "reina"})

	t.Logf("Final execution status: %s, error: %s", finalExec.Status, finalExec.Error)
	assert.Equal(t, models.WorkflowExecutionStatusCompleted, finalExec.Status)
	assert.True(t, executed["Activity_0kpgn1l"], "Proceso Ligero must be executed")
	assert.True(t, executed["Activity_0wfmm60"], "Leguear must be executed")
	assert.True(t, executed["Activity_1oe3o01"], "Saludsar por http must be executed")
	assert.False(t, executed["Activity_1pxklqa"], "Proceso Pesado must NOT be executed")
}

func TestUserProvidedBPMN_MissingActivityQueue_StrictlyFailsWithError(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()

	defID := "user_bpmn_no_queue"
	defRepo, err := db.NewWorkflowDefinitionRepository(uow, &db.DeterministicIDGeneratorFactory{}, "default", "default")
	require.NoError(t, err)

	_, err = defRepo.CreateWorkflowDefinition(&models.WorkflowDefinition{
		ID:         defID,
		Code:       "CODE_" + defID,
		VNamespace: "default",
		Name:       "Missing Queue Test",
		Version:    1,
		Payload:    []byte(userProvidedBPMN),
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, now)
	require.NoError(t, err)

	// Create ONLY execution queue, NO activity queue
	queueRepo, err := db.NewQueueRepository(uow, &db.DeterministicIDGeneratorFactory{}, "default", "default")
	require.NoError(t, err)
	_, err = queueRepo.CreateQueue(&models.Queue{
		ID:                   "q_exec_" + defID,
		Code:                 "wf-exec-CODE_" + defID,
		Name:                 "Exec Queue",
		Type:                 models.WorkflowExecutionQueue,
		WorkflowDefinitionID: defID,
		VNamespace:           "default",
		State:                models.QueueActive,
		MaxAttempts:          1000,
	}, now)
	require.NoError(t, err)
	require.NoError(t, uow.Commit())

	// Start Workflow Execution
	startUow := db.NewUnitOfWork(store, nil)
	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		WorkflowDefinitionID: defID,
		Input:                map[string]interface{}{"name": "angel"},
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	res := startCmd.Execute(startUow, now)
	require.Empty(t, res.Error)
	require.NoError(t, startUow.Commit())

	exec := res.Result.(*models.WorkflowExecution)

	// Advance token to task node
	advUow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}
	tokenRepo, _ := db.NewExecutionTokenRepository(advUow, idFactory, "default", "default")
	tokens, _ := tokenRepo.GetActiveTokensByExecutionID(exec.ID, now)
	require.NotEmpty(t, tokens)

	advCmd := &workflowExecCommand.AdvanceTokenCommand{
		ExecutionID: exec.ID,
		TokenID:     tokens[0].ID,
		CF:          "default",
		CFS:         "default",
	}
	advRes := advCmd.Execute(advUow, now)
	_ = advUow.Commit()

	// Should report error that activity queue is missing without auto-provisioning
	assert.NotEmpty(t, advRes.Error)
	assert.Contains(t, advRes.Error, "activity queue not found for workflow definition")

	execCheckUow := db.NewUnitOfWork(store, nil)
	execRepo, _ := db.NewWorkflowExecutionRepository(execCheckUow, idFactory, "default", "default")
	finalExec, _ := execRepo.GetWorkflowExecutionByID(exec.ID, now)

	assert.Equal(t, models.WorkflowExecutionStatusFailed, finalExec.Status)
	assert.Contains(t, finalExec.Error, "activity queue not found")
}
