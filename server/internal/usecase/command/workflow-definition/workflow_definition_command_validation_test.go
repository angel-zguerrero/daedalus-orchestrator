package workflow_definition_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	workflowDefCommand "deadalus-orch/server/internal/usecase/command/workflow-definition"
	"deadalus-orch/shared/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestValidationPebbleStore(t *testing.T) db.KVStore {
	tempDir, err := os.MkdirTemp("", "workflow_def_test_*")
	require.NoError(t, err)

	cfNames := []string{"default", db.AdminFC}
	store, err := db.CreatePebbleStore(tempDir, cfNames, []string{})
	require.NoError(t, err)

	t.Cleanup(func() {
		store.Close()
		os.RemoveAll(tempDir)
	})
	return store
}

const testValidBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_1">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start">
      <bpmn:extensionElements>
        <camunda:formData>
          <camunda:formField id="user_id" label="User ID" type="string" />
        </camunda:formData>
      </bpmn:extensionElements>
    </bpmn:startEvent>
  </bpmn:process>
</bpmn:definitions>`

const testInvalidBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_1">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start">
      <bpmn:extensionElements>
        <camunda:formData>
          <camunda:formField label="User ID" type="string" />
        </camunda:formData>
      </bpmn:extensionElements>
    </bpmn:startEvent>
  </bpmn:process>
</bpmn:definitions>`

func TestCreateWorkflowDefinitionCommand_RejectsInvalidFormFieldID(t *testing.T) {
	store := newTestValidationPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()

	cmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:            "wf-1",
			Code:          "wf-test",
			Name:          "Test Workflow",
			PayloadFormat: models.WorkflowPayloadFormatBPMN,
			Payload:       []byte(testInvalidBPMN),
			VNamespace:    "default",
			IsActive:      true,
		},
		CF:  "default",
		CFS: "default",
	}

	res := cmd.Execute(uow, now)
	assert.NotEmpty(t, res.Error)
	assert.Contains(t, res.Error, "invalid workflow diagram")
	assert.True(t, strings.Contains(res.Error, "missing required 'id'"))
}

func TestCreateWorkflowDefinitionCommand_AcceptsValidDiagram(t *testing.T) {
	store := newTestValidationPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()

	cmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:            "wf-2",
			Code:          "wf-valid",
			Name:          "Valid Workflow",
			PayloadFormat: models.WorkflowPayloadFormatBPMN,
			Payload:       []byte(testValidBPMN),
			VNamespace:    "default",
			IsActive:      true,
		},
		CF:  "default",
		CFS: "default",
	}

	res := cmd.Execute(uow, now)
	assert.Empty(t, res.Error)
	assert.NotNil(t, res.Result)
}

func TestUpdateWorkflowDefinitionCommand_RejectsInvalidFormFieldID(t *testing.T) {
	store := newTestValidationPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()

	// First create a valid workflow definition
	createCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:            "wf-3",
			Code:          "wf-upd",
			Name:          "Updatable Workflow",
			PayloadFormat: models.WorkflowPayloadFormatBPMN,
			Payload:       []byte(testValidBPMN),
			VNamespace:    "default",
			IsActive:      true,
		},
		CF:  "default",
		CFS: "default",
	}
	createRes := createCmd.Execute(uow, now)
	require.Empty(t, createRes.Error)
	require.NoError(t, uow.Commit())

	// Now attempt to update with invalid diagram (missing form field id)
	updateUow := db.NewUnitOfWork(store, nil)
	updateCmd := &workflowDefCommand.UpdateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:            "wf-3",
			Name:          "Updated Workflow",
			PayloadFormat: models.WorkflowPayloadFormatBPMN,
			Payload:       []byte(testInvalidBPMN),
		},
		CF:  "default",
		CFS: "default",
	}

	updateRes := updateCmd.Execute(updateUow, now)
	assert.NotEmpty(t, updateRes.Error)
	assert.Contains(t, updateRes.Error, "invalid workflow diagram")
	assert.True(t, strings.Contains(updateRes.Error, "missing required 'id'"))
}
