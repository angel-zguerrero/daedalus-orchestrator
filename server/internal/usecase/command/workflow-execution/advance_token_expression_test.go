package workflow_execution_test

import (
	"os"
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	workflowDefCommand "deadalus-orch/server/internal/usecase/command/workflow-definition"
	workflowExecCommand "deadalus-orch/server/internal/usecase/command/workflow-execution"
	"deadalus-orch/shared/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvanceTokenCommand_ExpressionEvaluation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "expr_eval_test_*")
	require.NoError(t, err)

	cfNames := []string{"default", db.AdminFC}
	store, err := db.CreatePebbleStore(tempDir, cfNames, []string{})
	require.NoError(t, err)
	t.Cleanup(func() {
		store.Close()
		os.RemoveAll(tempDir)
	})

	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()

	bpmnXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_1">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="Start_1">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:serviceTask id="Task_HTTP" name="Call HTTP Endpoint">
      <bpmn:extensionElements>
        <camunda:property name="url" value="https://${host}/api/v1/users/${userId}" />
        <camunda:property name="greeting" value="Hello ${user.firstName} ${user.lastName}!" />
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_1</bpmn:incoming>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="Start_1" targetRef="Task_HTTP" />
  </bpmn:process>
</bpmn:definitions>`

	defID := "def_expr_test"
	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:         defID,
			Code:       "CODE_EXPR_TEST",
			VNamespace: "default",
			Name:       "Expression Test",
			Version:    1,
			Payload:    []byte(bpmnXML),
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		ExecQueue: models.Queue{ID: "q_exec_" + defID},
		ActQueue:  models.Queue{ID: "q_act_" + defID},
		CF:        "default",
		CFS:       "default",
	}
	resDef := defCmd.Execute(uow, now)
	require.Empty(t, resDef.Error)
	require.NoError(t, uow.Commit())

	// Start Workflow Execution with input variables
	startUow := db.NewUnitOfWork(store, nil)
	inputVars := map[string]interface{}{
		"host":   "api.daedalus.io",
		"userId": "usr_998234",
		"user": map[string]interface{}{
			"firstName": "Angel",
			"lastName":  "Guerrero",
		},
	}

	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		ExecutionID:          "exec_expr_01",
		WorkflowDefinitionID: defID,
		Input:                inputVars,
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	resStart := startCmd.Execute(startUow, now)
	require.Empty(t, resStart.Error)
	require.NoError(t, startUow.Commit())

	// Advance Token from Start Event to Service Task
	tokenUow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}
	tokenRepo, err := db.NewExecutionTokenRepository(tokenUow, idFactory, "default", "default")
	require.NoError(t, err)
	activeTokens, err := tokenRepo.GetActiveTokensByExecutionID("exec_expr_01", now)
	require.NoError(t, err)
	require.NotEmpty(t, activeTokens)

	advUow := db.NewUnitOfWork(store, nil)
	advCmd := &workflowExecCommand.AdvanceTokenCommand{
		ExecutionID: "exec_expr_01",
		TokenID:     activeTokens[0].ID,
		CF:          "default",
		CFS:         "default",
	}
	resAdv := advCmd.Execute(advUow, now)
	require.Empty(t, resAdv.Error)
	require.NoError(t, advUow.Commit())

	// Verify Created Workflow Job Payload Has Resolved Expressions
	checkUow := db.NewUnitOfWork(store, nil)
	jobRepo, err := db.NewWorkflowJobRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)

	jobs, err := jobRepo.GetJobsByExecutionID("exec_expr_01", now)
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	jobPayload := jobs[0].Input
	assert.Equal(t, "https://api.daedalus.io/api/v1/users/usr_998234", jobPayload["url"])
	assert.Equal(t, "Hello Angel Guerrero!", jobPayload["greeting"])
}
