package bpmn

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type ElementType string

const (
	ElementStartEvent             ElementType = "startEvent"
	ElementEndEvent               ElementType = "endEvent"
	ElementIntermediateCatchEvent ElementType = "intermediateCatchEvent"
	ElementIntermediateThrowEvent ElementType = "intermediateThrowEvent"
	ElementBoundaryEvent          ElementType = "boundaryEvent"
	ElementServiceTask            ElementType = "serviceTask"
	ElementUserTask               ElementType = "userTask"
	ElementReceiveTask            ElementType = "receiveTask"
	ElementSendTask               ElementType = "sendTask"
	ElementBusinessRuleTask       ElementType = "businessRuleTask"
	ElementManualTask             ElementType = "manualTask"
	ElementTask                   ElementType = "task"
	ElementScriptTask             ElementType = "scriptTask"
	ElementExclusiveGateway       ElementType = "exclusiveGateway"
	ElementParallelGateway        ElementType = "parallelGateway"
	ElementInclusiveGateway       ElementType = "inclusiveGateway"
	ElementSequenceFlow           ElementType = "sequenceFlow"
)

type FormFieldConstraint struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}

type FormFieldValue struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type FormField struct {
	ID           string                `json:"id"`
	Label        string                `json:"label"`
	Type         string                `json:"type"`
	DefaultValue string                `json:"defaultValue"`
	Values       []FormFieldValue      `json:"values,omitempty"`
	Constraints  []FormFieldConstraint `json:"constraints,omitempty"`
}

type BPMNNode struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Type                ElementType       `json:"type"`
	DefaultFlowID       string            `json:"defaultFlowId,omitempty"`
	Incoming            []string          `json:"incoming"`
	Outgoing            []string          `json:"outgoing"`
	Properties          map[string]string `json:"properties,omitempty"`
	ExtensionProperties map[string]string `json:"extensionProperties,omitempty"`
	FormFields          []*FormField      `json:"formFields,omitempty"`
}

type SequenceFlow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SourceRef string `json:"sourceRef"`
	TargetRef string `json:"targetRef"`
	Condition string `json:"condition,omitempty"`
}

type BPMNModel struct {
	ProcessID   string                  `json:"processId"`
	Nodes       map[string]*BPMNNode    `json:"nodes"`
	Flows       map[string]*SequenceFlow `json:"flows"`
	StartNodeID string                  `json:"startNodeId"`
}

func (m *BPMNModel) GetNode(id string) *BPMNNode {
	return m.Nodes[id]
}

func (m *BPMNModel) GetFlow(id string) *SequenceFlow {
	return m.Flows[id]
}

func (m *BPMNModel) GetOutgoingFlows(nodeID string) []*SequenceFlow {
	node := m.GetNode(nodeID)
	if node == nil {
		return nil
	}
	var flows []*SequenceFlow
	for _, flowID := range node.Outgoing {
		if flow, ok := m.Flows[flowID]; ok {
			flows = append(flows, flow)
		}
	}
	return flows
}

func (m *BPMNModel) GetIncomingFlows(nodeID string) []*SequenceFlow {
	node := m.GetNode(nodeID)
	if node == nil {
		return nil
	}
	var flows []*SequenceFlow
	for _, flowID := range node.Incoming {
		if flow, ok := m.Flows[flowID]; ok {
			flows = append(flows, flow)
		}
	}
	return flows
}

func ParseBPMN(xmlData []byte) (*BPMNModel, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(xmlData)))
	
	model := &BPMNModel{
		Nodes: make(map[string]*BPMNNode),
		Flows: make(map[string]*SequenceFlow),
	}

	var currentNode *BPMNNode
	var currentFormField *FormField
	var currentFlow *SequenceFlow
	var currentElement string
	var inConditionExpression bool
	var inTimeDuration bool
	var inTimeDate bool
	var inTimeCycle bool
	var conditionBuf strings.Builder
	var inScriptElement bool
	var scriptBuf strings.Builder
	var timeBuf strings.Builder

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error decoding BPMN XML: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			local := stripPrefix(t.Name.Local)
			currentElement = local
			elemType := normalizeElementType(local)

			switch elemType {
			case ElementStartEvent, ElementEndEvent, ElementIntermediateCatchEvent, ElementIntermediateThrowEvent, ElementBoundaryEvent, ElementServiceTask, ElementUserTask, ElementReceiveTask, ElementSendTask, ElementBusinessRuleTask, ElementManualTask, ElementTask, ElementScriptTask, ElementExclusiveGateway, ElementParallelGateway, ElementInclusiveGateway:
				id := getAttr(t.Attr, "id")
				name := getAttr(t.Attr, "name")
				defaultFlow := getAttr(t.Attr, "default")
				node := &BPMNNode{
					ID:            id,
					Name:          name,
					Type:          elemType,
					DefaultFlowID: defaultFlow,
					Properties:          make(map[string]string),
					ExtensionProperties: make(map[string]string),
				}
				model.Nodes[id] = node
				currentNode = node

				if node.Type == ElementStartEvent && model.StartNodeID == "" {
					model.StartNodeID = id
				}

				if msgRef := getAttr(t.Attr, "messageRef"); msgRef != "" {
					node.Properties["messageRef"] = msgRef
				}

				if template := getAttr(t.Attr, "modelerTemplate"); template != "" {
					node.Properties["modelerTemplate"] = template
				}
				if template := getAttr(t.Attr, "modeler:template"); template != "" {
					node.Properties["modelerTemplate"] = template
				}
				if template := getAttr(t.Attr, "zeebe:modelerTemplate"); template != "" {
					node.Properties["modelerTemplate"] = template
				}
				if scriptFormat := getAttr(t.Attr, "scriptFormat"); scriptFormat != "" {
					node.Properties["scriptFormat"] = scriptFormat
				}
				if resultVar := getAttr(t.Attr, "resultVariable"); resultVar != "" {
					node.Properties["resultVariable"] = resultVar
				}
				if scriptAttr := getAttr(t.Attr, "script"); scriptAttr != "" {
					node.Properties["script"] = scriptAttr
				}

			case "script":
				if currentNode != nil {
					inScriptElement = true
					scriptBuf.Reset()
					if sf := getAttr(t.Attr, "scriptFormat"); sf != "" && currentNode.Properties["scriptFormat"] == "" {
						currentNode.Properties["scriptFormat"] = sf
					}
				}

			case "taskDefinition":
				if currentNode != nil {
					if tType := getAttr(t.Attr, "type"); tType != "" {
						currentNode.Properties["taskType"] = tType
					}
				}
			case "modelerTemplate":
				if currentNode != nil {
					if id := getAttr(t.Attr, "id"); id != "" {
						currentNode.Properties["modelerTemplate"] = id
					}
				}
			case "property":
				if currentNode != nil {
					pName := getAttr(t.Attr, "name")
					pValue := getAttr(t.Attr, "value")
					if pName != "" {
						currentNode.Properties[pName] = pValue
						if currentNode.ExtensionProperties == nil {
							currentNode.ExtensionProperties = make(map[string]string)
						}
						currentNode.ExtensionProperties[pName] = pValue
					}
				}
			case "formField":
				if currentNode != nil {
					fID := getAttr(t.Attr, "id")
					fLabel := getAttr(t.Attr, "label")
					fType := getAttr(t.Attr, "type")
					fDef := getAttr(t.Attr, "defaultValue")
					if fID != "" {
						field := &FormField{
							ID:           fID,
							Label:        fLabel,
							Type:         fType,
							DefaultValue: fDef,
						}
						currentNode.FormFields = append(currentNode.FormFields, field)
						currentFormField = field
					}
				}
			case "value":
				if currentFormField != nil {
					vID := getAttr(t.Attr, "id")
					vName := getAttr(t.Attr, "name")
					if vID != "" {
						currentFormField.Values = append(currentFormField.Values, FormFieldValue{
							ID:   vID,
							Name: vName,
						})
					}
				}
			case "constraint":
				if currentFormField != nil {
					cName := getAttr(t.Attr, "name")
					cConfig := getAttr(t.Attr, "config")
					if cConfig == "" {
						cConfig = getAttr(t.Attr, "value")
					}
					if cName != "" {
						currentFormField.Constraints = append(currentFormField.Constraints, FormFieldConstraint{
							Name:   cName,
							Config: cConfig,
						})
					}
				}

			case ElementSequenceFlow:
				id := getAttr(t.Attr, "id")
				name := getAttr(t.Attr, "name")
				sourceRef := getAttr(t.Attr, "sourceRef")
				targetRef := getAttr(t.Attr, "targetRef")

				flow := &SequenceFlow{
					ID:        id,
					Name:      name,
					SourceRef: sourceRef,
					TargetRef: targetRef,
				}
				model.Flows[id] = flow
				currentFlow = flow

			case "conditionExpression":
				inConditionExpression = true
				conditionBuf.Reset()

			case "timeDuration":
				inTimeDuration = true
				timeBuf.Reset()

			case "timeDate":
				inTimeDate = true
				timeBuf.Reset()

			case "timeCycle":
				inTimeCycle = true
				timeBuf.Reset()
			}

		case xml.CharData:
			if inConditionExpression {
				conditionBuf.Write(t)
			} else if inScriptElement {
				scriptBuf.Write(t)
			} else if inTimeDuration || inTimeDate || inTimeCycle {
				timeBuf.Write(t)
			} else if currentNode != nil && (currentElement == "incoming" || currentElement == "outgoing") {
				val := strings.TrimSpace(string(t))
				if val != "" {
					if currentElement == "incoming" {
						currentNode.Incoming = append(currentNode.Incoming, val)
					} else if currentElement == "outgoing" {
						currentNode.Outgoing = append(currentNode.Outgoing, val)
					}
				}
			}

		case xml.EndElement:
			local := stripPrefix(t.Name.Local)
			if inConditionExpression && local == "conditionExpression" {
				inConditionExpression = false
				if currentFlow != nil {
					currentFlow.Condition = strings.TrimSpace(conditionBuf.String())
				}
			}
			if inScriptElement && local == "script" {
				inScriptElement = false
				if currentNode != nil {
					body := strings.TrimSpace(scriptBuf.String())
					if body != "" {
						currentNode.Properties["script"] = body
					}
				}
			}

			if inTimeDuration && local == "timeDuration" {
				inTimeDuration = false
				if currentNode != nil {
					currentNode.Properties["timeDuration"] = strings.TrimSpace(timeBuf.String())
				}
			}

			if inTimeDate && local == "timeDate" {
				inTimeDate = false
				if currentNode != nil {
					currentNode.Properties["timeDate"] = strings.TrimSpace(timeBuf.String())
				}
			}

			if inTimeCycle && local == "timeCycle" {
				inTimeCycle = false
				if currentNode != nil {
					currentNode.Properties["timeCycle"] = strings.TrimSpace(timeBuf.String())
				}
			}

			elemType := normalizeElementType(local)
			switch elemType {
			case ElementStartEvent, ElementEndEvent, ElementIntermediateCatchEvent, ElementIntermediateThrowEvent, ElementBoundaryEvent, ElementServiceTask, ElementUserTask, ElementReceiveTask, ElementSendTask, ElementBusinessRuleTask, ElementManualTask, ElementTask, ElementScriptTask, ElementExclusiveGateway, ElementParallelGateway, ElementInclusiveGateway:
				currentNode = nil
				currentFormField = nil
			case "formField":
				currentFormField = nil
			case ElementSequenceFlow:
				currentFlow = nil
			}
		}
	}

	// Link flows to node incoming/outgoing if not explicitly listed in xml children
	for _, flow := range model.Flows {
		if srcNode, ok := model.Nodes[flow.SourceRef]; ok {
			if !contains(srcNode.Outgoing, flow.ID) {
				srcNode.Outgoing = append(srcNode.Outgoing, flow.ID)
			}
		}
		if targetNode, ok := model.Nodes[flow.TargetRef]; ok {
			if !contains(targetNode.Incoming, flow.ID) {
				targetNode.Incoming = append(targetNode.Incoming, flow.ID)
			}
		}
	}

	return model, nil
}

func normalizeElementType(name string) ElementType {
	switch strings.ToLower(name) {
	case "startevent":
		return ElementStartEvent
	case "endevent":
		return ElementEndEvent
	case "intermediatecatchevent":
		return ElementIntermediateCatchEvent
	case "intermediatethrowevent":
		return ElementIntermediateThrowEvent
	case "boundaryevent":
		return ElementBoundaryEvent
	case "servicetask":
		return ElementServiceTask
	case "usertask":
		return ElementUserTask
	case "receivetask":
		return ElementReceiveTask
	case "sendtask":
		return ElementSendTask
	case "businessruletask":
		return ElementBusinessRuleTask
	case "manualtask":
		return ElementManualTask
	case "task":
		return ElementTask
	case "scripttask":
		return ElementScriptTask
	case "exclusivegateway":
		return ElementExclusiveGateway
	case "parallelgateway":
		return ElementParallelGateway
	case "inclusivegateway":
		return ElementInclusiveGateway
	case "sequenceflow":
		return ElementSequenceFlow
	default:
		return ElementType(name)
	}
}

func stripPrefix(name string) string {
	if idx := strings.Index(name, ":"); idx != -1 {
		return name[idx+1:]
	}
	return name
}

func getAttr(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func isReservedProperty(name string) bool {
	switch name {
	case "messageRef", "modelerTemplate", "modeler:template", "zeebe:modelerTemplate",
		"scriptFormat", "resultVariable", "camunda:resultVariable", "outputVariable",
		"script", "scriptBody", "taskType", "timeDuration", "timeDate", "timeCycle",
		"expectedKeys":
		return true
	default:
		return false
	}
}

// BuildExpectedInputFromNode constructs a structured ExpectedInput map from a BPMNNode's configuration.
func BuildExpectedInputFromNode(node *BPMNNode) map[string]interface{} {
	if node == nil {
		return map[string]interface{}{
			"fields":                 []interface{}{},
			"extensionProperties":    map[string]string{},
			"hasFormFields":          false,
			"hasExtensionProperties": false,
		}
	}

	result := map[string]interface{}{
		"nodeId":   node.ID,
		"nodeName": node.Name,
		"nodeType": string(node.Type),
	}

	for k, v := range node.Properties {
		result[k] = v
	}

	// Extract clean extension properties
	extProps := make(map[string]string)
	if node.ExtensionProperties != nil {
		for k, v := range node.ExtensionProperties {
			extProps[k] = v
		}
	}
	for k, v := range node.Properties {
		if !isReservedProperty(k) {
			if _, exists := extProps[k]; !exists {
				extProps[k] = v
			}
		}
	}
	result["extensionProperties"] = extProps
	result["hasExtensionProperties"] = len(extProps) > 0

	var fields []map[string]interface{}
	if len(node.FormFields) > 0 {
		fields = make([]map[string]interface{}, 0, len(node.FormFields))
		for _, f := range node.FormFields {
			isRequired := false
			for _, c := range f.Constraints {
				if strings.EqualFold(c.Name, "required") {
					isRequired = true
					break
				}
			}

			fieldMap := map[string]interface{}{
				"id":           f.ID,
				"name":         f.ID,
				"label":        f.Label,
				"type":         f.Type,
				"defaultValue": f.DefaultValue,
				"required":     isRequired,
			}
			if fieldMap["label"] == "" {
				fieldMap["label"] = f.ID
			}

			if len(f.Values) > 0 {
				vals := make([]map[string]string, 0, len(f.Values))
				for _, v := range f.Values {
					vals = append(vals, map[string]string{
						"id":   v.ID,
						"name": v.Name,
					})
				}
				fieldMap["values"] = vals
			}

			if len(f.Constraints) > 0 {
				cons := make([]map[string]string, 0, len(f.Constraints))
				for _, c := range f.Constraints {
					cons = append(cons, map[string]string{
						"name":   c.Name,
						"config": c.Config,
					})
				}
				fieldMap["constraints"] = cons
			}

			fields = append(fields, fieldMap)
		}
	} else if expectedKeysStr, ok := node.Properties["expectedKeys"]; ok && strings.TrimSpace(expectedKeysStr) != "" {
		keys := strings.Split(expectedKeysStr, ",")
		fields = make([]map[string]interface{}, 0, len(keys))
		for _, k := range keys {
			cleanKey := strings.TrimSpace(k)
			if cleanKey != "" {
				fields = append(fields, map[string]interface{}{
					"id":       cleanKey,
					"name":     cleanKey,
					"label":    cleanKey,
					"type":     "string",
					"required": true,
				})
			}
		}
	} else {
		fields = []map[string]interface{}{}
	}

	result["fields"] = fields
	result["hasFormFields"] = len(fields) > 0
	return result
}

