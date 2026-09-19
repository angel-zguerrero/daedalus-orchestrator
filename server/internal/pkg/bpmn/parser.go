package bpmn

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type ElementType string

const (
	ElementStartEvent       ElementType = "startEvent"
	ElementEndEvent         ElementType = "endEvent"
	ElementServiceTask      ElementType = "serviceTask"
	ElementUserTask         ElementType = "userTask"
	ElementTask             ElementType = "task"
	ElementScriptTask       ElementType = "scriptTask"
	ElementExclusiveGateway ElementType = "exclusiveGateway"
	ElementParallelGateway  ElementType = "parallelGateway"
	ElementInclusiveGateway ElementType = "inclusiveGateway"
	ElementSequenceFlow     ElementType = "sequenceFlow"
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
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Type          ElementType       `json:"type"`
	DefaultFlowID string            `json:"defaultFlowId,omitempty"`
	Incoming      []string          `json:"incoming"`
	Outgoing      []string          `json:"outgoing"`
	Properties    map[string]string `json:"properties,omitempty"`
	FormFields    []*FormField      `json:"formFields,omitempty"`
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
	var conditionBuf strings.Builder

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

			switch ElementType(local) {
			case ElementStartEvent, ElementEndEvent, ElementServiceTask, ElementUserTask, ElementTask, ElementScriptTask, ElementExclusiveGateway, ElementParallelGateway, ElementInclusiveGateway:
				id := getAttr(t.Attr, "id")
				name := getAttr(t.Attr, "name")
				defaultFlow := getAttr(t.Attr, "default")
				node := &BPMNNode{
					ID:            id,
					Name:          name,
					Type:          ElementType(local),
					DefaultFlowID: defaultFlow,
					Properties:    make(map[string]string),
				}
				model.Nodes[id] = node
				currentNode = node

				if node.Type == ElementStartEvent && model.StartNodeID == "" {
					model.StartNodeID = id
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
			}

		case xml.CharData:
			if inConditionExpression {
				conditionBuf.Write(t)
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

			switch ElementType(local) {
			case ElementStartEvent, ElementEndEvent, ElementServiceTask, ElementUserTask, ElementTask, ElementScriptTask, ElementExclusiveGateway, ElementParallelGateway, ElementInclusiveGateway:
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
