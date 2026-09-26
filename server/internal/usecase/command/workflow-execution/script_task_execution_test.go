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

const scriptTaskWorkflowBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" id="Definitions_ScriptTest" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_ScriptTest" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:scriptTask id="ScriptTask_Calc" name="Calcular Descuento y Nivel" scriptFormat="javascript" camunda:resultVariable="calculoPedido">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
      <bpmn:script>
        const subtotal = monto * cantidad;
        const esVip = subtotal &gt;= 1000;
        const descuento = esVip ? subtotal * 0.15 : 0;
        return {
          subtotal: subtotal,
          descuento: descuento,
          totalFinal: subtotal - descuento,
          esVip: esVip
        };
      </bpmn:script>
    </bpmn:scriptTask>
    <bpmn:exclusiveGateway id="Gateway_Vip" default="Flow_Standard">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_Vip</bpmn:outgoing>
      <bpmn:outgoing>Flow_Standard</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:task id="Task_VipPath" name="Aprobacion VIP">
      <bpmn:incoming>Flow_Vip</bpmn:incoming>
      <bpmn:outgoing>Flow_EndVip</bpmn:outgoing>
    </bpmn:task>
    <bpmn:task id="Task_StandardPath" name="Aprobacion Estandar">
      <bpmn:incoming>Flow_Standard</bpmn:incoming>
      <bpmn:outgoing>Flow_EndStd</bpmn:outgoing>
    </bpmn:task>
    <bpmn:endEvent id="EndEvent_1" name="End">
      <bpmn:incoming>Flow_EndVip</bpmn:incoming>
      <bpmn:incoming>Flow_EndStd</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ScriptTask_Calc" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ScriptTask_Calc" targetRef="Gateway_Vip" />
    <bpmn:sequenceFlow id="Flow_Vip" sourceRef="Gateway_Vip" targetRef="Task_VipPath">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">calculoPedido.esVip == true</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_Standard" sourceRef="Gateway_Vip" targetRef="Task_StandardPath" />
    <bpmn:sequenceFlow id="Flow_EndVip" sourceRef="Task_VipPath" targetRef="EndEvent_1" />
    <bpmn:sequenceFlow id="Flow_EndStd" sourceRef="Task_StandardPath" targetRef="EndEvent_1" />
  </bpmn:process>
</bpmn:definitions>`

func TestScriptTaskWorkflowExecution_MapsReturnToResultVariableAndRoutesGateway(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()
	testID := "wf_script_test_vip"

	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:         testID,
			Code:       "CODE_" + testID,
			VNamespace: "default",
			Name:       "Script Task Workflow Test",
			Version:    1,
			Payload:    []byte(scriptTaskWorkflowBPMN),
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

	// Start execution with monto=300, cantidad=4 -> subtotal=1200 -> esVip=true, totalFinal=1020
	startUow := db.NewUnitOfWork(store, nil)
	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		WorkflowDefinitionID: testID,
		Input: map[string]interface{}{
			"monto":    300,
			"cantidad": 4,
		},
		VNamespace: "default",
		CF:         "default",
		CFS:        "default",
	}
	res := startCmd.Execute(startUow, now)
	require.Empty(t, res.Error)
	require.NoError(t, startUow.Commit())

	exec := res.Result.(*models.WorkflowExecution)
	executedActivities := make(map[string]bool)

	for step := 0; step < 30; step++ {
		execCheckUow := db.NewUnitOfWork(store, nil)
		execRepo, _ := db.NewWorkflowExecutionRepository(execCheckUow, idFactory, "default", "default")
		currentExec, _ := execRepo.GetWorkflowExecutionByID(exec.ID, now)
		if currentExec.Status == models.WorkflowExecutionStatusCompleted || currentExec.Status == models.WorkflowExecutionStatusFailed {
			exec = currentExec
			break
		}

		tokenUow := db.NewUnitOfWork(store, nil)
		tokenRepo, _ := db.NewExecutionTokenRepository(tokenUow, idFactory, "default", "default")
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

		jobUow := db.NewUnitOfWork(store, nil)
		jobRepo, _ := db.NewWorkflowJobRepository(jobUow, idFactory, "default", "default")
		jobs, _ := jobRepo.GetJobsByExecutionID(exec.ID, now)

		for idx := range jobs {
			j := &jobs[idx]
			if j.Status == models.WorkflowJobStatusPending {
				executedActivities[j.ActivityID] = true
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
			}
		}
	}

	assert.Equal(t, models.WorkflowExecutionStatusCompleted, exec.Status)
	assert.True(t, executedActivities["ScriptTask_Calc"], "ScriptTask_Calc must be executed")
	assert.True(t, executedActivities["Task_VipPath"], "Task_VipPath must be executed because calculoPedido.esVip == true")
	assert.False(t, executedActivities["Task_StandardPath"], "Task_StandardPath must NOT be executed")

	// Verify output variable mapped in StateData and Output
	require.NotNil(t, exec.StateData["calculoPedido"])
	calcMap, ok := exec.StateData["calculoPedido"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, calcMap["esVip"])
	assert.InDelta(t, 1200.0, calcMap["subtotal"], 0.01)
	assert.InDelta(t, 180.0, calcMap["descuento"], 0.01)
	assert.InDelta(t, 1020.0, calcMap["totalFinal"], 0.01)
}
