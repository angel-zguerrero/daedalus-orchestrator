package zeebe

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protowire"
)

type rawCodec struct{}

func (rawCodec) Marshal(v interface{}) ([]byte, error) {
	if b, ok := v.([]byte); ok {
		return b, nil
	}
	return nil, fmt.Errorf("expected []byte, got %T", v)
}

func (rawCodec) Unmarshal(data []byte, v interface{}) error {
	if b, ok := v.(*[]byte); ok {
		*b = append((*b)[:0], data...)
		return nil
	}
	return fmt.Errorf("expected *[]byte, got %T", v)
}

func (rawCodec) Name() string { return "raw" }

func encodeResource(name string, content []byte) []byte {
	var res []byte
	res = protowire.AppendTag(res, 1, protowire.BytesType)
	res = protowire.AppendString(res, name)
	res = protowire.AppendTag(res, 2, protowire.BytesType)
	res = protowire.AppendBytes(res, content)
	return res
}

func encodeDeployResourceRequest(resourceBytes []byte) []byte {
	var req []byte
	req = protowire.AppendTag(req, 1, protowire.BytesType)
	req = protowire.AppendBytes(req, resourceBytes)
	return req
}

func encodeCreateProcessInstanceRequest(bpmnProcessID string, variablesJSON string) []byte {
	var req []byte
	req = protowire.AppendTag(req, 2, protowire.BytesType) // bpmnProcessId = field 2
	req = protowire.AppendString(req, bpmnProcessID)
	req = protowire.AppendTag(req, 3, protowire.VarintType) // version = field 3 (-1)
	var ver int64 = -1
	req = protowire.AppendVarint(req, uint64(ver))
	if variablesJSON != "" {
		req = protowire.AppendTag(req, 4, protowire.BytesType) // variables = field 4
		req = protowire.AppendString(req, variablesJSON)
	}
	return req
}

func encodeCreateProcessInstanceWithResultRequest(bpmnProcessID string, variablesJSON string, timeoutMs int64) []byte {
	cpReq := encodeCreateProcessInstanceRequest(bpmnProcessID, variablesJSON)
	var req []byte
	req = protowire.AppendTag(req, 1, protowire.BytesType) // request = field 1
	req = protowire.AppendBytes(req, cpReq)
	req = protowire.AppendTag(req, 2, protowire.VarintType) // timeout = field 2
	req = protowire.AppendVarint(req, uint64(timeoutMs))
	return req
}

func parseVariablesFromResponse(data []byte) map[string]interface{} {
	out := make(map[string]interface{})
	out["status"] = "SUCCESS"

	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			break
		}
		data = data[n:]
		if num == 5 && typ == protowire.BytesType {
			val, n := protowire.ConsumeString(data)
			if n >= 0 && val != "" {
				_ = json.Unmarshal([]byte(val), &out)
			}
			break
		} else {
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				break
			}
			data = data[n:]
		}
	}
	return out
}

type ZeebeClient struct {
	address string
}

func NewZeebeClient(address string) *ZeebeClient {
	if address == "" || strings.Contains(address, "8086") {
		// Default Zeebe Gateway gRPC port is 26500
		address = "localhost:26500"
	}
	address = strings.TrimPrefix(address, "http://")
	address = strings.TrimPrefix(address, "https://")
	return &ZeebeClient{address: address}
}

func (c *ZeebeClient) ExecuteConnectorJob(ctx context.Context, jobID, activityName, activityType string, inputData map[string]interface{}) (map[string]interface{}, error) {
	connCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(connCtx, c.address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Zeebe Gateway at %s: %w", c.address, err)
	}
	defer conn.Close()

	if activityName == "" {
		activityName = "Connector Task"
	}

	// Resolve activityType strictly from BPMN metadata or explicit payload parameters (no name matching)
	if activityType == "" || activityType == "task" || activityType == "serviceTask" || activityType == "userTask" || activityType == "scriptTask" {
		if inputData != nil {
			if typeVal, ok := inputData["connectorType"].(string); ok && typeVal != "" {
				activityType = typeVal
			} else if typeVal, ok := inputData["taskType"].(string); ok && typeVal != "" {
				activityType = typeVal
			} else if typeVal, ok := inputData["type"].(string); ok && typeVal != "" {
				activityType = typeVal
			}
		}
	}
	if activityType == "" || activityType == "task" || activityType == "serviceTask" || activityType == "userTask" || activityType == "scriptTask" {
		activityType = "io.camunda:http-json:1"
	}

	if (activityType == "io.camunda:connector-redis:1" || activityType == "io.camunda:connector-redis") && inputData != nil {
		if _, hasConn := inputData["connection"]; !hasConn {
			// Provide default connection and command if not explicitly supplied
			redisURL := "redis://localhost:6379"
			if _, hasKey := inputData["key"]; !hasKey {
				inputData["key"] = "daedalus_test_key"
				inputData["value"] = "hello_world_from_daedalus"
			}
			inputData["connection"] = map[string]interface{}{
				"uri": redisURL,
			}
			inputData["command"] = map[string]interface{}{
				"type":  "SET",
				"key":   inputData["key"],
				"value": inputData["value"],
			}
		}
	} else if activityType == "io.camunda:http-json:1" && inputData != nil {
		if _, hasURL := inputData["url"]; !hasURL {
			inputData["url"] = "https://httpbin.org/get"
			inputData["method"] = "GET"
		}
	}

	cleanJobID := strings.ReplaceAll(jobID, "-", "")
	processID := fmt.Sprintf("connector_%s", cleanJobID)

	bpmnXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:zeebe="http://camunda.org/schema/zeebe/1.0"
                  id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="%s" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Task_Connector"/>
    <bpmn:serviceTask id="Task_Connector" name="%s">
      <bpmn:extensionElements>
        <zeebe:taskDefinition type="%s"/>
      </bpmn:extensionElements>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Task_Connector" targetRef="EndEvent_1"/>
    <bpmn:endEvent id="EndEvent_1"/>
  </bpmn:process>
</bpmn:definitions>`, processID, activityName, activityType)

	callOpt := grpc.ForceCodec(rawCodec{})

	// 1. Deploy BPMN Resource to Zeebe
	resourceBytes := encodeResource(processID+".bpmn", []byte(bpmnXML))
	deployReq := encodeDeployResourceRequest(resourceBytes)
	var deployResp []byte
	if err := conn.Invoke(ctx, "/gateway_protocol.Gateway/DeployResource", deployReq, &deployResp, callOpt); err != nil {
		return nil, fmt.Errorf("failed to deploy connector BPMN to Zeebe: %w", err)
	}

	// 2. Create Process Instance With Result in Zeebe
	if inputData == nil {
		inputData = make(map[string]interface{})
	}
	inputJSON, _ := json.Marshal(inputData)
	createReq := encodeCreateProcessInstanceWithResultRequest(processID, string(inputJSON), 25000)
	var createResp []byte
	if err := conn.Invoke(ctx, "/gateway_protocol.Gateway/CreateProcessInstanceWithResult", createReq, &createResp, callOpt); err != nil {
		return nil, fmt.Errorf("failed to execute connector via Zeebe Gateway: %w", err)
	}

	outVars := parseVariablesFromResponse(createResp)
	return outVars, nil
}
