package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	models "deadalus-orch/shared/models"
)

type ActivityTemplateRepository struct {
	*Repository[models.ActivityTemplate]
}

func NewActivityTemplateRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*ActivityTemplateRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.ActivityTemplate](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &ActivityTemplateRepository{Repository: repo}, nil
}

func (r *ActivityTemplateRepository) CreateActivityTemplate(input *models.ActivityTemplate, now time.Time) (string, error) {
	if input.Code == "" {
		return "", fmt.Errorf("Code is required")
	}
	if input.VNamespace == "" {
		input.VNamespace = "default"
	}
	if input.ActivityFamily == "" {
		input.ActivityFamily = "default"
	}
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *ActivityTemplateRepository) UpdateActivityTemplate(input *models.ActivityTemplate, now time.Time) (bool, error) {
	if input.VNamespace == "" {
		input.VNamespace = "default"
	}
	if input.ActivityFamily == "" {
		input.ActivityFamily = "default"
	}
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *ActivityTemplateRepository) GetActivityTemplateByID(id string, now time.Time) (*models.ActivityTemplate, error) {
	return r.FindByField("ID", id, now)
}

func (r *ActivityTemplateRepository) GetActivityTemplateByCode(code string, vnamespace string, now time.Time) (*models.ActivityTemplate, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil
	}
	var query string
	if vnamespace != "" {
		query = fmt.Sprintf("Code = %s & VNamespace = %s", code, vnamespace)
	} else {
		query = fmt.Sprintf("Code = %s", code)
	}
	res, err := r.Find(query, 1, "", now)
	if err == nil && res != nil && len(res.Entities) > 0 {
		return &res.Entities[0], nil
	}
	return nil, err
}

// ResolveByCodeOrID resolves an ActivityTemplate by Code (with or without VNamespace), ID, or case-insensitive fallback scan.
func (r *ActivityTemplateRepository) ResolveByCodeOrID(codeOrID string, vnamespace string, now time.Time) (*models.ActivityTemplate, error) {
	codeOrID = strings.TrimSpace(codeOrID)
	if codeOrID == "" || IsBuiltinActivityType(codeOrID) {
		return nil, nil
	}

	if vnamespace != "" {
		if tpl, err := r.GetActivityTemplateByCode(codeOrID, vnamespace, now); err == nil && tpl != nil && tpl.ID != "" {
			return tpl, nil
		}
	}
	if tpl, err := r.GetActivityTemplateByCode(codeOrID, "default", now); err == nil && tpl != nil && tpl.ID != "" {
		return tpl, nil
	}
	if tpl, err := r.GetActivityTemplateByCode(codeOrID, "", now); err == nil && tpl != nil && tpl.ID != "" {
		return tpl, nil
	}
	if tpl, err := r.GetActivityTemplateByID(codeOrID, now); err == nil && tpl != nil && tpl.ID != "" {
		return tpl, nil
	}

	// Case-insensitive fallback scan across all templates in this repository
	if allRes, err := r.Find("ID != 0", 500, "", now); err == nil && allRes != nil {
		for i := range allRes.Entities {
			e := &allRes.Entities[i]
			if strings.EqualFold(strings.TrimSpace(e.Code), codeOrID) ||
				strings.EqualFold(strings.TrimSpace(e.ID), codeOrID) ||
				strings.EqualFold(strings.TrimSpace(e.Name), codeOrID) {
				return e, nil
			}
		}
	}

	return nil, nil
}

func (r *ActivityTemplateRepository) ListActivityTemplates(scope string, tenantID string, vnamespace string, activityFamily string, pageSize int, cursor string, now time.Time) (*FindResult[models.ActivityTemplate], error) {
	var conditions []string

	if scope != "" {
		conditions = append(conditions, fmt.Sprintf("Scope = %s", scope))
		if scope == string(models.ActivityTemplateScopeTenant) && tenantID != "" {
			conditions = append(conditions, fmt.Sprintf("TenantID = %s", tenantID))
		}
	} else if tenantID != "" {
		conditions = append(conditions, fmt.Sprintf("TenantID = %s", tenantID))
	}

	if vnamespace != "" {
		conditions = append(conditions, fmt.Sprintf("VNamespace = %s", vnamespace))
	}

	if activityFamily != "" {
		conditions = append(conditions, fmt.Sprintf("ActivityFamily = %s", activityFamily))
	}

	var query string
	if len(conditions) == 0 {
		query = "ID != 0"
	} else {
		query = strings.Join(conditions, " & ")
	}

	if pageSize <= 0 {
		pageSize = 50
	}

	return r.Find(query, pageSize, cursor, now)
}

func (r *ActivityTemplateRepository) DeleteActivityTemplate(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}

// IsBuiltinActivityType returns true if the given activityType is a built-in native connector ID.
func IsBuiltinActivityType(activityType string) bool {
	return NormalizeBuiltinActivityType(activityType) != ""
}

// NormalizeBuiltinActivityType returns the canonical built-in connector ID if activityType matches a native connector, or "" otherwise.
func NormalizeBuiltinActivityType(activityType string) string {
	lower := strings.ToLower(strings.TrimSpace(activityType))
	switch lower {
	case "io.camunda.connectors.httpjson.v1", "io.camunda:http-json:1", "http-json", "http", "rest":
		return "io.camunda.connectors.HttpJson.v1"
	case "io.camunda.connectors.redis.v1", "io.camunda:connector-redis:1", "io.camunda:connector-redis", "redis":
		return "io.camunda.connectors.Redis.v1"
	case "io.camunda.connectors.logtask.v1", "logtask", "log":
		return "io.camunda.connectors.LogTask.v1"
	case "task", "servicetask", "service-task", "scripttask", "script-task", "usertask", "user-task", "sendtask", "send-task", "receivetask", "receive-task", "manualtask", "manual-task", "businessruletask", "business-rule-task":
		return "task"
	default:
		return ""
	}
}

type templatePayloadDoc struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Properties []map[string]interface{} `json:"properties"`
}

type templatePayloadTyped struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Properties []struct {
		Label    string      `json:"label"`
		Type     string      `json:"type"`
		Value    interface{} `json:"value"`
		Editable *bool       `json:"editable"`
		Binding  struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"binding"`
	} `json:"properties"`
}

// InferBaseActivityTypeFromPayload inspects a template JSON payload's property bindings to infer the underlying native executor.
func InferBaseActivityTypeFromPayload(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	var parsed templatePayloadTyped
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return ""
	}
	propNames := make(map[string]bool)
	for _, p := range parsed.Properties {
		name := strings.ToLower(strings.TrimSpace(p.Binding.Name))
		if name != "" {
			propNames[name] = true
		}
	}
	if propNames["connectionstring"] || propNames["command"] || propNames["key"] {
		return "io.camunda.connectors.Redis.v1"
	}
	if propNames["url"] {
		return "io.camunda.connectors.HttpJson.v1"
	}
	if propNames["message"] || propNames["logmessage"] {
		return "io.camunda.connectors.LogTask.v1"
	}
	return ""
}

// InferBaseActivityTypeFromMap inspects a merged map of properties to infer the underlying native executor.
func InferBaseActivityTypeFromMap(input map[string]interface{}) string {
	if input == nil {
		return ""
	}
	propNames := make(map[string]bool)
	for k, v := range input {
		name := strings.ToLower(strings.TrimSpace(k))
		if name != "" && v != nil {
			valStr := strings.TrimSpace(fmt.Sprintf("%v", v))
			if valStr != "" && valStr != "<nil>" {
				propNames[name] = true
			}
		}
	}
	if propNames["connectionstring"] || (propNames["command"] && propNames["key"]) {
		return "io.camunda.connectors.Redis.v1"
	}
	if propNames["url"] {
		return "io.camunda.connectors.HttpJson.v1"
	}
	if propNames["message"] || propNames["logmessage"] {
		return "io.camunda.connectors.LogTask.v1"
	}
	return ""
}

var builtinDefaultPropertyValues = map[string]string{
	"command":             "GET",
	"method":              "GET",
	"authentication.type": "noAuth",
	"level":               "INFO",
}

// MergeActivityTemplateHierarchy merges properties across a multi-level template inheritance chain (ordered from root parent to leaf child,
// e.g. [Activity 1, Activity 2]) and combines them with the workflow task input.
func MergeActivityTemplateHierarchy(ancestorsRootToLeaf []*models.ActivityTemplate, taskInput map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range taskInput {
		merged[k] = v
	}
	if len(ancestorsRootToLeaf) == 0 {
		return merged
	}

	templateValues := make(map[string]interface{})
	lockedByAncestor := make(map[string]bool)

	for _, tpl := range ancestorsRootToLeaf {
		if tpl == nil || len(tpl.Payload) == 0 {
			continue
		}
		var parsed templatePayloadTyped
		if err := json.Unmarshal(tpl.Payload, &parsed); err != nil {
			continue
		}
		for _, prop := range parsed.Properties {
			propName := strings.TrimSpace(prop.Binding.Name)
			if propName == "" || prop.Value == nil {
				continue
			}
			valStr := strings.TrimSpace(fmt.Sprintf("%v", prop.Value))
			if valStr == "" || valStr == "<nil>" {
				continue
			}
			isFrozen := (prop.Editable != nil && !*prop.Editable) || strings.EqualFold(prop.Type, "hidden")

			// If an earlier ancestor already locked this property, a child template cannot override its value
			if lockedByAncestor[propName] {
				continue
			}

			// If an earlier ancestor configured a non-default value (e.g. command="SET") and the child template still has
			// the untouched built-in default (e.g. command="GET") without locking it, preserve the ancestor's non-default value
			if existingVal, hasExisting := templateValues[propName]; hasExisting && !isFrozen {
				existingStr := strings.TrimSpace(fmt.Sprintf("%v", existingVal))
				if defVal, isDefProp := builtinDefaultPropertyValues[propName]; isDefProp && valStr == defVal && existingStr != defVal {
					continue
				}
			}

			templateValues[propName] = prop.Value
			if isFrozen {
				lockedByAncestor[propName] = true
			}
		}
	}

	for propName, tplVal := range templateValues {
		tplValStr := strings.TrimSpace(fmt.Sprintf("%v", tplVal))
		existingVal, exists := merged[propName]
		existingStr := ""
		if exists && existingVal != nil {
			existingStr = strings.TrimSpace(fmt.Sprintf("%v", existingVal))
		}

		if lockedByAncestor[propName] {
			// If the locked template value was a ${...} expression already evaluated by AdvanceTokenCommand, keep the evaluated value
			if strings.Contains(tplValStr, "${") && existingStr != "" && !strings.Contains(existingStr, "${") {
				continue
			}
			merged[propName] = tplVal
		} else {
			if !exists || existingStr == "" || existingStr == "<nil>" {
				merged[propName] = tplVal
			} else if defVal, isDefProp := builtinDefaultPropertyValues[propName]; isDefProp && existingStr == defVal && tplValStr != defVal {
				merged[propName] = tplVal
			}
		}
	}

	return merged
}

// EnrichTemplatePayloadWithAncestors updates a leaf template's JSON payload with inherited values and locks from its ancestor chain (ordered root to leaf).
func EnrichTemplatePayloadWithAncestors(ancestorsRootToLeaf []*models.ActivityTemplate, leafPayload []byte) []byte {
	if len(ancestorsRootToLeaf) <= 1 || len(leafPayload) == 0 {
		return leafPayload
	}
	var leafDoc map[string]interface{}
	if err := json.Unmarshal(leafPayload, &leafDoc); err != nil {
		return leafPayload
	}
	rawProps, ok := leafDoc["properties"].([]interface{})
	if !ok || len(rawProps) == 0 {
		return leafPayload
	}

	// Compute inherited values & lock flags from all ancestors prior to leaf
	parentChain := ancestorsRootToLeaf[:len(ancestorsRootToLeaf)-1]
	inheritedMap := MergeActivityTemplateHierarchy(parentChain, nil)

	lockedByParent := make(map[string]bool)
	for _, parentTpl := range parentChain {
		if parentTpl == nil || len(parentTpl.Payload) == 0 {
			continue
		}
		var parsed templatePayloadTyped
		if err := json.Unmarshal(parentTpl.Payload, &parsed); err != nil {
			continue
		}
		for _, prop := range parsed.Properties {
			pName := strings.TrimSpace(prop.Binding.Name)
			isFrozen := (prop.Editable != nil && !*prop.Editable) || strings.EqualFold(prop.Type, "hidden")
			if pName != "" && isFrozen {
				lockedByParent[pName] = true
			}
		}
	}

	for _, rawProp := range rawProps {
		propMap, ok := rawProp.(map[string]interface{})
		if !ok {
			continue
		}
		bindingMap, _ := propMap["binding"].(map[string]interface{})
		if bindingMap == nil {
			continue
		}
		bName, _ := bindingMap["name"].(string)
		bName = strings.TrimSpace(bName)
		if bName == "" {
			continue
		}
		parentVal, hasParentVal := inheritedMap[bName]
		if !hasParentVal {
			continue
		}
		parentValStr := strings.TrimSpace(fmt.Sprintf("%v", parentVal))
		currVal := propMap["value"]
		currValStr := ""
		if currVal != nil {
			currValStr = strings.TrimSpace(fmt.Sprintf("%v", currVal))
		}

		if lockedByParent[bName] {
			propMap["value"] = parentVal
			propMap["editable"] = false
		} else if currValStr == "" || currValStr == "<nil>" {
			propMap["value"] = parentVal
		} else if defVal, isDef := builtinDefaultPropertyValues[bName]; isDef && currValStr == defVal && parentValStr != defVal {
			propMap["value"] = parentVal
		}
	}

	enrichedBytes, err := json.Marshal(leafDoc)
	if err != nil {
		return leafPayload
	}
	return enrichedBytes
}

