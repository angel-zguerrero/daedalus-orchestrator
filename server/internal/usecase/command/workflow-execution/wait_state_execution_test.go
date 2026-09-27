package workflow_execution_test

import (
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/bpmn"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	workflowDefCommand "deadalus-orch/server/internal/usecase/command/workflow-definition"
	workflowExecCommand "deadalus-orch/server/internal/usecase/command/workflow-execution"
	"deadalus-orch/shared/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const waitStateFormBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_WaitState" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_WaitState" isExecutable="true">
    <bpmn:startEvent id="Start_1" name="Start">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="Start_1" targetRef="UserTask_Approval" />
    <bpmn:userTask id="UserTask_Approval" name="User Approval">
      <bpmn:extensionElements>
        <camunda:formData>
          <camunda:formField id="approverName" label="Approver Name" type="string">
            <camunda:validation>
              <camunda:constraint name="required" />
              <camunda:constraint name="minlength" config="3" />
            </camunda:validation>
          </camunda:formField>
          <camunda:formField id="approved" label="Approve?" type="boolean">
            <camunda:validation>
              <camunda:constraint name="required" />
            </camunda:validation>
          </camunda:formField>
          <camunda:formField id="score" label="Score" type="long" />
        </camunda:formData>
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="UserTask_Approval" targetRef="ReceiveTask_Webhook" />
    <bpmn:receiveTask id="ReceiveTask_Webhook" name="Wait for Webhook">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:receiveTask>
    <bpmn:sequenceFlow id="Flow_3" sourceRef="ReceiveTask_Webhook" targetRef="End_1" />
    <bpmn:endEvent id="End_1" name="End">
      <bpmn:incoming>Flow_3</bpmn:incoming>
    </bpmn:endEvent>
  </bpmn:process>
</bpmn:definitions>`

func TestWaitState_PauseResumeAndValidation(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()
	testID := "wf_wait_state_test"

	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:         testID,
			Code:       "CODE_" + testID,
			VNamespace: "default",
			Name:       "Wait State Workflow Test",
			Version:    1,
			Payload:    []byte(waitStateFormBPMN),
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

	// Start Execution
	startUow := db.NewUnitOfWork(store, nil)
	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		WorkflowDefinitionID: testID,
		Input:                map[string]interface{}{"orderNumber": "ORD-999"},
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	startRes := startCmd.Execute(startUow, now)
	require.Empty(t, startRes.Error)
	require.NoError(t, startUow.Commit())

	exec := startRes.Result.(*models.WorkflowExecution)
	require.Equal(t, models.WorkflowExecutionStatusRunning, exec.Status)

	// Step 1: Advance token from Start_1 to UserTask_Approval
	tokenUow := db.NewUnitOfWork(store, nil)
	tokenRepo, err := db.NewExecutionTokenRepository(tokenUow, idFactory, "default", "default")
	require.NoError(t, err)
	tokens, err := tokenRepo.GetActiveTokensByExecutionID(exec.ID, now)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	tokenID := tokens[0].ID

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

	// Step 2: Verify token is paused in waiting state and waiting_event record exists
	checkUow := db.NewUnitOfWork(store, nil)
	tokenRepo2, err := db.NewExecutionTokenRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)
	tok, err := tokenRepo2.GetExecutionTokenByID(tokenID, now)
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionTokenStatusWaiting, tok.Status, "Token must be paused with status waiting")
	assert.Equal(t, "UserTask_Approval", tok.CurrentNodeID)

	waitRepo, err := db.NewWaitingEventRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)
	waitingEvents, err := waitRepo.GetWaitingEventsByExecutionID(exec.ID, now)
	require.NoError(t, err)
	require.Len(t, waitingEvents, 1, "Exactly one waiting_event record must exist")

	userWaitEvent := waitingEvents[0]
	assert.Equal(t, models.WaitingEventTypeUserInput, userWaitEvent.Type)
	assert.Equal(t, "UserTask_Approval", userWaitEvent.EventID)
	assert.Equal(t, exec.ID, userWaitEvent.WorkflowExecutionID)
	assert.Equal(t, tokenID, userWaitEvent.ExecutionTokenID)
	assert.Equal(t, testID, userWaitEvent.WorkflowDefinitionID)

	// Verify ExpectedInput structure
	require.NotNil(t, userWaitEvent.ExpectedInput)
	fieldsRaw, ok := userWaitEvent.ExpectedInput["fields"].([]interface{})
	require.True(t, ok, "ExpectedInput must contain fields slice")
	assert.Len(t, fieldsRaw, 3, "ExpectedInput must contain 3 form fields")

	// Step 3: Server-side validation tests
	// Test 3a: Missing required field (approverName)
	errMissingReq := bpmn.ValidateEventInput(userWaitEvent.ExpectedInput, map[string]interface{}{
		"approved": true,
	})
	assert.Error(t, errMissingReq)
	assert.Contains(t, errMissingReq.Error(), "approverName")

	// Test 3b: minlength violation on approverName
	errMinLen := bpmn.ValidateEventInput(userWaitEvent.ExpectedInput, map[string]interface{}{
		"approverName": "Al", // length 2 < 3
		"approved":     true,
	})
	assert.Error(t, errMinLen)
	assert.Contains(t, errMinLen.Error(), "length")

	// Test 3c: Wrong type for boolean
	errWrongType := bpmn.ValidateEventInput(userWaitEvent.ExpectedInput, map[string]interface{}{
		"approverName": "Alice",
		"approved":     "invalid-boolean-value",
	})
	assert.Error(t, errWrongType)
	assert.Contains(t, errWrongType.Error(), "boolean")

	// Test 3d: Wrong type for score (not integer)
	errWrongInt := bpmn.ValidateEventInput(userWaitEvent.ExpectedInput, map[string]interface{}{
		"approverName": "Alice",
		"approved":     true,
		"score":        "abc",
	})
	assert.Error(t, errWrongInt)
	assert.Contains(t, errWrongInt.Error(), "integer")

	// Test 3e: Resume attempt with invalid payload fails and does NOT resume
	badResumeUow := db.NewUnitOfWork(store, nil)
	badResumeCmd := &workflowExecCommand.ResumeWaitEventCommand{
		WaitingEventID: userWaitEvent.ID,
		Payload: map[string]interface{}{
			"approverName": "Al", // invalid minlength
			"approved":     true,
		},
		CF:  "default",
		CFS: "default",
	}
	badResumeRes := badResumeCmd.Execute(badResumeUow, now)
	assert.NotEmpty(t, badResumeRes.Error, "Resume command must reject invalid payload")

	// Step 4: Resume UserTask with VALID payload
	goodPayload := map[string]interface{}{
		"approverName": "Alice Smith",
		"approved":     true,
		"score":        100,
	}
	errValid := bpmn.ValidateEventInput(userWaitEvent.ExpectedInput, goodPayload)
	require.NoError(t, errValid, "Valid payload must pass validation")

	resumeUow := db.NewUnitOfWork(store, nil)
	resumeCmd := &workflowExecCommand.ResumeWaitEventCommand{
		WaitingEventID: userWaitEvent.ID,
		Payload:        goodPayload,
		CF:             "default",
		CFS:            "default",
	}
	resumeRes := resumeCmd.Execute(resumeUow, now)
	require.Empty(t, resumeRes.Error)
	require.NoError(t, resumeUow.Commit())

	// Step 5: Verify UserTask wait event was deleted, payload was merged into StateData, and token advanced to ReceiveTask_Webhook
	verifyUow := db.NewUnitOfWork(store, nil)
	waitRepo2, err := db.NewWaitingEventRepository(verifyUow, idFactory, "default", "default")
	require.NoError(t, err)
	oldEvent, err := waitRepo2.GetWaitingEventByID(userWaitEvent.ID, now)
	assert.Nil(t, oldEvent, "Previous waiting_event must have been deleted")

	execRepo2, err := db.NewWorkflowExecutionRepository(verifyUow, idFactory, "default", "default")
	require.NoError(t, err)
	updatedExec, err := execRepo2.GetWorkflowExecutionByID(exec.ID, now)
	require.NoError(t, err)
	assert.Equal(t, "Alice Smith", updatedExec.StateData["approverName"], "Validated values must be merged into StateData")
	assert.Equal(t, true, updatedExec.StateData["approved"])
	assert.EqualValues(t, 100, updatedExec.StateData["score"])
	assert.Equal(t, "ORD-999", updatedExec.StateData["orderNumber"])

	// Token should now have reached ReceiveTask_Webhook and paused again as SYSTEM_MESSAGE wait state
	activeWaitEvents, err := waitRepo2.GetWaitingEventsByExecutionID(exec.ID, now)
	require.NoError(t, err)
	require.Len(t, activeWaitEvents, 1, "Token should have paused at ReceiveTask_Webhook creating a new wait event")
	receiveWaitEvent := activeWaitEvents[0]
	assert.Equal(t, models.WaitingEventTypeSystemMessage, receiveWaitEvent.Type)
	assert.Equal(t, "ReceiveTask_Webhook", receiveWaitEvent.EventID)

	// Step 6: Resume ReceiveTask_Webhook with callback payload
	webhookPayload := map[string]interface{}{
		"webhookEvent": "PAYMENT_CONFIRMED",
		"transactionId": "TX-12345",
	}
	resumeWebhookUow := db.NewUnitOfWork(store, nil)
	resumeWebhookCmd := &workflowExecCommand.ResumeWaitEventCommand{
		WaitingEventID: receiveWaitEvent.ID,
		Payload:        webhookPayload,
		CF:             "default",
		CFS:            "default",
	}
	resumeWebhookRes := resumeWebhookCmd.Execute(resumeWebhookUow, now)
	require.Empty(t, resumeWebhookRes.Error)
	require.NoError(t, resumeWebhookUow.Commit())

	// Step 7: Verify Workflow reaches End_1 and is completed
	finalUow := db.NewUnitOfWork(store, nil)
	execRepoFinal, err := db.NewWorkflowExecutionRepository(finalUow, idFactory, "default", "default")
	require.NoError(t, err)
	finalExec, err := execRepoFinal.GetWorkflowExecutionByID(exec.ID, now)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowExecutionStatusCompleted, finalExec.Status, "Workflow must reach completed status")
	assert.Equal(t, "PAYMENT_CONFIRMED", finalExec.StateData["webhookEvent"])
	assert.Equal(t, "TX-12345", finalExec.StateData["transactionId"])

	waitRepoFinal, err := db.NewWaitingEventRepository(finalUow, idFactory, "default", "default")
	require.NoError(t, err)
	finalWaitEvents, err := waitRepoFinal.GetWaitingEventsByExecutionID(exec.ID, now)
	require.NoError(t, err)
	assert.Empty(t, finalWaitEvents, "No active waiting events should remain after completion")
}

func TestWorkflowExecutionDetail_IncludesWaitingEvents(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()
	testID := "wf_detail_wait_test"

	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:         testID,
			Code:       "CODE_" + testID,
			VNamespace: "default",
			Name:       "Wait Detail Test",
			Version:    1,
			Payload:    []byte(waitStateFormBPMN),
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		ExecQueue: models.Queue{ID: "q_exec_" + testID, Type: models.WorkflowExecutionQueue, WorkflowDefinitionID: testID, Code: "wf-exec-CODE_" + testID},
		ActQueue:  models.Queue{ID: "q_act_" + testID, Type: models.WorkflowActivityQueue, WorkflowDefinitionID: testID, Code: "wf-act-CODE_" + testID},
		CF:        "default",
		CFS:       "default",
	}
	require.Empty(t, defCmd.Execute(uow, now).Error)
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
	startRes := startCmd.Execute(startUow, now)
	require.Empty(t, startRes.Error)
	require.NoError(t, startUow.Commit())
	exec := startRes.Result.(*models.WorkflowExecution)

	// Advance to UserTask
	tokenUow := db.NewUnitOfWork(store, nil)
	tokenRepo, _ := db.NewExecutionTokenRepository(tokenUow, idFactory, "default", "default")
	tokens, _ := tokenRepo.GetActiveTokensByExecutionID(exec.ID, now)
	require.Len(t, tokens, 1)

	advUow := db.NewUnitOfWork(store, nil)
	advCmd := &workflowExecCommand.AdvanceTokenCommand{
		ExecutionID: exec.ID,
		TokenID:     tokens[0].ID,
		CF:          "default",
		CFS:         "default",
	}
	require.Empty(t, advCmd.Execute(advUow, now).Error)
	require.NoError(t, advUow.Commit())

	// Query detail via GetWorkflowExecutionCommand
	getUow := db.NewUnitOfWork(store, nil)
	getCmd := &workflowExecCommand.GetWorkflowExecutionCommand{
		ID:  exec.ID,
		CF:  "default",
		CFS: "default",
	}
	getRes := getCmd.Execute(getUow, now)
	require.Empty(t, getRes.Error)

	detail := getRes.Result.(workflowExecCommand.WorkflowExecutionDetail)
	assert.Equal(t, exec.ID, detail.Execution.ID)
	require.Len(t, detail.WaitingEvents, 1, "WorkflowExecutionDetail must include active waiting events")
	assert.Equal(t, "UserTask_Approval", detail.WaitingEvents[0].EventID)
	assert.Equal(t, models.WaitingEventTypeUserInput, detail.WaitingEvents[0].Type)
}

func TestWaitState_RepositoryCommandRegistryAndExtensionProperties(t *testing.T) {
	// 1. Verify Dragonboat repository command registry decoding
	typeName, getEvtBytes, err := general_command.EncodeRepoCommand(&workflowExecCommand.GetWaitingEventCommand{
		ID:  "evt_123",
		CF:  "default",
		CFS: "default",
	})
	require.NoError(t, err)
	assert.Equal(t, "GetWaitingEventCommand", typeName)
	cmd, err := general_command.DecodeRepoCommand("GetWaitingEventCommand", getEvtBytes)
	require.NoError(t, err, "GetWaitingEventCommand must be registered in repository-command-registry")
	assert.IsType(t, &workflowExecCommand.GetWaitingEventCommand{}, cmd)

	resTypeName, resumeEvtBytes, err := general_command.EncodeRepoCommand(&workflowExecCommand.ResumeWaitEventCommand{
		WaitingEventID: "evt_123",
		Payload:        map[string]interface{}{"name": "test"},
		CF:             "default",
		CFS:            "default",
	})
	require.NoError(t, err)
	assert.Equal(t, "ResumeWaitEventCommand", resTypeName)
	resumeCmd, err := general_command.DecodeRepoCommand("ResumeWaitEventCommand", resumeEvtBytes)
	require.NoError(t, err, "ResumeWaitEventCommand must be registered in repository-command-registry")
	assert.IsType(t, &workflowExecCommand.ResumeWaitEventCommand{}, resumeCmd)

	// 2. Parse BPMN with both form fields and extension properties
	xmlWithExt := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_Ext" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Ext" isExecutable="true">
    <bpmn:startEvent id="Start_1" />
    <bpmn:sequenceFlow id="Flow_1" sourceRef="Start_1" targetRef="UserTask_Ext" />
    <bpmn:userTask id="UserTask_Ext" name="Task with Extensions">
      <bpmn:extensionElements>
        <camunda:formData>
          <camunda:formField id="name" label="Full Name" type="string">
            <camunda:validation>
              <camunda:constraint name="required" />
            </camunda:validation>
          </camunda:formField>
        </camunda:formData>
        <camunda:properties>
          <camunda:property name="data.ci" value="" />
        </camunda:properties>
      </bpmn:extensionElements>
    </bpmn:userTask>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="UserTask_Ext" targetRef="End_1" />
    <bpmn:endEvent id="End_1" />
  </bpmn:process>
</bpmn:definitions>`

	model, err := bpmn.ParseBPMN([]byte(xmlWithExt))
	require.NoError(t, err)
	node := model.GetNode("UserTask_Ext")
	require.NotNil(t, node)

	expectedInput := bpmn.BuildExpectedInputFromNode(node)
	assert.True(t, expectedInput["hasFormFields"].(bool))
	assert.True(t, expectedInput["hasExtensionProperties"].(bool))

	extProps, ok := expectedInput["extensionProperties"].(map[string]string)
	require.True(t, ok)
	assert.Contains(t, extProps, "data.ci")

	// 3. Validation with nested JSON payload
	validPayload := map[string]interface{}{
		"name": "Angel",
		"data": map[string]interface{}{
			"ci": "12345678",
		},
	}
	err = bpmn.ValidateEventInput(expectedInput, validPayload)
	assert.NoError(t, err)

	// Validation missing required "name"
	invalidPayload := map[string]interface{}{
		"data": map[string]interface{}{
			"ci": "12345678",
		},
	}
	err = bpmn.ValidateEventInput(expectedInput, invalidPayload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field \"name\" is required")
}
