package bpmn_test

import (
	"testing"

	"deadalus-orch/server/internal/pkg/bpmn"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBPMN_TasksRecognition(t *testing.T) {
	bpmnXML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_1">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="Start_1" />
    <bpmn:sequenceFlow id="Flow_1" sourceRef="Start_1" targetRef="Receive_1" />
    <bpmn:receiveTask id="Receive_1" name="Receive Webhook" messageRef="Msg_1" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Receive_1" targetRef="User_1" />
    <bpmn:userTask id="User_1" name="Review Approval" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="User_1" targetRef="Send_1" />
    <bpmn:sendTask id="Send_1" name="Send Notification" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="Send_1" targetRef="Rule_1" />
    <bpmn:businessRuleTask id="Rule_1" name="Calculate Risk" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="Rule_1" targetRef="Manual_1" />
    <bpmn:manualTask id="Manual_1" name="Manual Check" />
    <bpmn:sequenceFlow id="Flow_6" sourceRef="Manual_1" targetRef="End_1" />
    <bpmn:endEvent id="End_1" />
  </bpmn:process>
</bpmn:definitions>`)

	model, err := bpmn.ParseBPMN(bpmnXML)
	require.NoError(t, err)
	require.NotNil(t, model)

	// Check ReceiveTask
	receiveNode := model.GetNode("Receive_1")
	require.NotNil(t, receiveNode)
	assert.Equal(t, bpmn.ElementReceiveTask, receiveNode.Type)
	assert.Equal(t, "Receive Webhook", receiveNode.Name)
	assert.Equal(t, "Msg_1", receiveNode.Properties["messageRef"])
	assert.Contains(t, receiveNode.Incoming, "Flow_1")
	assert.Contains(t, receiveNode.Outgoing, "Flow_2")

	// Check UserTask
	userNode := model.GetNode("User_1")
	require.NotNil(t, userNode)
	assert.Equal(t, bpmn.ElementUserTask, userNode.Type)
	assert.Equal(t, "Review Approval", userNode.Name)
	assert.Contains(t, userNode.Incoming, "Flow_2")
	assert.Contains(t, userNode.Outgoing, "Flow_3")

	// Check SendTask
	sendNode := model.GetNode("Send_1")
	require.NotNil(t, sendNode)
	assert.Equal(t, bpmn.ElementSendTask, sendNode.Type)
	assert.Equal(t, "Send Notification", sendNode.Name)
	assert.Contains(t, sendNode.Incoming, "Flow_3")
	assert.Contains(t, sendNode.Outgoing, "Flow_4")

	// Check BusinessRuleTask
	ruleNode := model.GetNode("Rule_1")
	require.NotNil(t, ruleNode)
	assert.Equal(t, bpmn.ElementBusinessRuleTask, ruleNode.Type)
	assert.Equal(t, "Calculate Risk", ruleNode.Name)
	assert.Contains(t, ruleNode.Incoming, "Flow_4")
	assert.Contains(t, ruleNode.Outgoing, "Flow_5")

	// Check ManualTask
	manualNode := model.GetNode("Manual_1")
	require.NotNil(t, manualNode)
	assert.Equal(t, bpmn.ElementManualTask, manualNode.Type)
	assert.Equal(t, "Manual Check", manualNode.Name)
	assert.Contains(t, manualNode.Incoming, "Flow_5")
	assert.Contains(t, manualNode.Outgoing, "Flow_6")
}

func TestParseBPMN_CaseInsensitiveElementTypes(t *testing.T) {
	// Some modelers or raw XML might use PascalCase or lowercase
	bpmnXML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_1">
  <process id="Process_1" isExecutable="true">
    <StartEvent id="Start_1" />
    <sequenceFlow id="Flow_1" sourceRef="Start_1" targetRef="Receive_1" />
    <ReceiveTask id="Receive_1" name="PascalCase Receive" />
    <sequenceFlow id="Flow_2" sourceRef="Receive_1" targetRef="End_1" />
    <EndEvent id="End_1" />
  </process>
</definitions>`)

	model, err := bpmn.ParseBPMN(bpmnXML)
	require.NoError(t, err)
	require.NotNil(t, model)

	node := model.GetNode("Receive_1")
	require.NotNil(t, node)
	assert.Equal(t, bpmn.ElementReceiveTask, node.Type)
	assert.Equal(t, "PascalCase Receive", node.Name)
	assert.Contains(t, node.Incoming, "Flow_1")
	assert.Contains(t, node.Outgoing, "Flow_2")
}
