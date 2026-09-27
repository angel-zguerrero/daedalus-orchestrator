package bpmn

import (
	"strings"
	"testing"
)

const validBPMNWithForms = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_1">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start">
      <bpmn:extensionElements>
        <camunda:formData>
          <camunda:formField id="username" label="User Name" type="string" defaultValue="guest" />
          <camunda:formField id="age" label="User Age" type="long" />
        </camunda:formData>
      </bpmn:extensionElements>
    </bpmn:startEvent>
    <bpmn:userTask id="UserTask_1" name="Approval Task">
      <bpmn:extensionElements>
        <camunda:formData>
          <camunda:formField id="approved" label="Is Approved" type="boolean" />
        </camunda:formData>
      </bpmn:extensionElements>
    </bpmn:userTask>
  </bpmn:process>
</bpmn:definitions>`

const bpmnWithMissingFieldID = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_1">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Inicio">
      <bpmn:extensionElements>
        <camunda:formData>
          <camunda:formField label="Nombre" type="string" />
        </camunda:formData>
      </bpmn:extensionElements>
    </bpmn:startEvent>
  </bpmn:process>
</bpmn:definitions>`

const bpmnWithEmptyFieldID = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_1">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Inicio">
      <bpmn:extensionElements>
        <camunda:formData>
          <camunda:formField id="" label="Nombre" type="string" />
        </camunda:formData>
      </bpmn:extensionElements>
    </bpmn:startEvent>
  </bpmn:process>
</bpmn:definitions>`

const bpmnWithWhitespaceFieldID = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_1">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:userTask id="Task_A" name="Revisión">
      <bpmn:extensionElements>
        <camunda:formData>
          <camunda:formField id="   " label="Comentarios" type="string" />
        </camunda:formData>
      </bpmn:extensionElements>
    </bpmn:userTask>
  </bpmn:process>
</bpmn:definitions>`

const bpmnWithDuplicateFieldID = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_1">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Inicio">
      <bpmn:extensionElements>
        <camunda:formData>
          <camunda:formField id="email" label="Email" type="string" />
          <camunda:formField id="email" label="Confirm Email" type="string" />
        </camunda:formData>
      </bpmn:extensionElements>
    </bpmn:startEvent>
  </bpmn:process>
</bpmn:definitions>`

func TestValidateDiagramFormFields_Valid(t *testing.T) {
	err := ValidateDiagramFormFields([]byte(validBPMNWithForms))
	if err != nil {
		t.Fatalf("expected valid diagram to pass form fields validation, got error: %v", err)
	}
}

func TestValidateDiagramFormFields_MissingID(t *testing.T) {
	err := ValidateDiagramFormFields([]byte(bpmnWithMissingFieldID))
	if err == nil {
		t.Fatal("expected error for form field missing id attribute, got nil")
	}
	if !strings.Contains(err.Error(), "missing required 'id'") {
		t.Errorf("expected error message to mention missing required 'id', got: %v", err)
	}
}

func TestValidateDiagramFormFields_EmptyID(t *testing.T) {
	err := ValidateDiagramFormFields([]byte(bpmnWithEmptyFieldID))
	if err == nil {
		t.Fatal("expected error for form field with empty id, got nil")
	}
	if !strings.Contains(err.Error(), "missing required 'id'") {
		t.Errorf("expected error message to mention missing required 'id', got: %v", err)
	}
}

func TestValidateDiagramFormFields_WhitespaceID(t *testing.T) {
	err := ValidateDiagramFormFields([]byte(bpmnWithWhitespaceFieldID))
	if err == nil {
		t.Fatal("expected error for form field with whitespace id, got nil")
	}
	if !strings.Contains(err.Error(), "missing required 'id'") {
		t.Errorf("expected error message to mention missing required 'id', got: %v", err)
	}
}

func TestValidateDiagramFormFields_DuplicateID(t *testing.T) {
	err := ValidateDiagramFormFields([]byte(bpmnWithDuplicateFieldID))
	if err == nil {
		t.Fatal("expected error for duplicate form field id, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate form field id") {
		t.Errorf("expected error message to mention duplicate form field id, got: %v", err)
	}
}

func TestParseBPMN_RejectsMissingFieldID(t *testing.T) {
	_, err := ParseBPMN([]byte(bpmnWithMissingFieldID))
	if err == nil {
		t.Fatal("expected ParseBPMN to error when form field has missing id, got nil")
	}
	if !strings.Contains(err.Error(), "missing required 'id'") {
		t.Errorf("expected error to mention missing required 'id', got: %v", err)
	}
}

func TestParseBPMN_AcceptsValidFieldIDs(t *testing.T) {
	model, err := ParseBPMN([]byte(validBPMNWithForms))
	if err != nil {
		t.Fatalf("expected ParseBPMN to succeed with valid form fields, got: %v", err)
	}
	startNode := model.Nodes["StartEvent_1"]
	if startNode == nil {
		t.Fatal("expected StartEvent_1 to be parsed")
	}
	if len(startNode.FormFields) != 2 {
		t.Fatalf("expected 2 form fields on StartEvent_1, got %d", len(startNode.FormFields))
	}
	if startNode.FormFields[0].ID != "username" || startNode.FormFields[1].ID != "age" {
		t.Errorf("form field IDs mismatch: got %s, %s", startNode.FormFields[0].ID, startNode.FormFields[1].ID)
	}
}
