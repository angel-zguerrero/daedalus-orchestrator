package app

import (
	"testing"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"
)

func TestMergeTemplateInput_BurnsFrozenRedisConnectionString(t *testing.T) {
	redisDerivedPayload := []byte(`{
		"id": "redis-primary-prod",
		"name": "Redis Primary Prod",
		"properties": [
			{
				"label": "Redis Host / Connection String",
				"type": "String",
				"value": "redis://prod-cluster:6379",
				"editable": false,
				"binding": {
					"type": "camunda:property",
					"name": "connectionString"
				}
			},
			{
				"label": "Command",
				"type": "Dropdown",
				"value": "SET",
				"editable": true,
				"binding": {
					"type": "camunda:property",
					"name": "command"
				}
			}
		]
	}`)

	tpl := &models.ActivityTemplate{
		ID:               "tpl-1",
		Code:             "redis-primary-prod",
		Name:             "Redis Primary Prod",
		ActivityFamily:   "cache",
		ParentTemplateId: "io.camunda.connectors.Redis.v1",
		Payload:          redisDerivedPayload,
		IsActive:         true,
	}

	// Case 1: User attempts to override connectionString in job input -> frozen property MUST win
	userJobInput := map[string]interface{}{
		"connectionString": "redis://hacked:6379",
		"key":              "session:123",
		"value":            "active",
	}

	merged := mergeTemplateInput(tpl, userJobInput)

	if merged["connectionString"] != "redis://prod-cluster:6379" {
		t.Fatalf("expected burned connectionString 'redis://prod-cluster:6379', got %v", merged["connectionString"])
	}
	// Editable default 'command' should apply since user didn't provide one
	if merged["command"] != "SET" {
		t.Fatalf("expected default command 'SET', got %v", merged["command"])
	}
	if merged["key"] != "session:123" {
		t.Fatalf("expected key 'session:123', got %v", merged["key"])
	}
}

func TestMergeTemplateChainInput_MultiLevelRedisInheritance(t *testing.T) {
	// Activity 1 inherits from Redis (io.camunda.connectors.Redis.v1)
	activity1Payload := []byte(`{
		"id": "activity-1",
		"name": "Activity 1",
		"properties": [
			{
				"label": "Redis Host / Connection String",
				"type": "String",
				"value": "redis://cluster-1:6379",
				"editable": false,
				"binding": { "type": "camunda:property", "name": "connectionString" }
			},
			{
				"label": "Password",
				"type": "String",
				"value": "s3cr3t",
				"editable": true,
				"binding": { "type": "camunda:property", "name": "password" }
			},
			{
				"label": "Command",
				"type": "Dropdown",
				"value": "SET",
				"editable": true,
				"binding": { "type": "camunda:property", "name": "command" }
			}
		]
	}`)

	activity1 := &models.ActivityTemplate{
		ID:               "id-act-1",
		Code:             "activity-1",
		Name:             "Activity 1",
		ParentTemplateId: "io.camunda.connectors.Redis.v1",
		Payload:          activity1Payload,
		IsActive:         true,
	}

	// Activity 2 inherits from Activity 1 (activity-1)
	activity2Payload := []byte(`{
		"id": "activity-2",
		"name": "Activity 2",
		"properties": [
			{
				"label": "Redis Host / Connection String",
				"type": "String",
				"value": "",
				"editable": true,
				"binding": { "type": "camunda:property", "name": "connectionString" }
			},
			{
				"label": "Command",
				"type": "Dropdown",
				"value": "GET",
				"editable": true,
				"binding": { "type": "camunda:property", "name": "command" }
			},
			{
				"label": "TTL",
				"type": "String",
				"value": "600",
				"editable": true,
				"binding": { "type": "camunda:property", "name": "ttl" }
			},
			{
				"label": "Key Name",
				"type": "String",
				"value": "user:default",
				"editable": true,
				"binding": { "type": "camunda:property", "name": "key" }
			}
		]
	}`)

	activity2 := &models.ActivityTemplate{
		ID:               "id-act-2",
		Code:             "activity-2",
		Name:             "Activity 2",
		ParentTemplateId: "activity-1",
		Payload:          activity2Payload,
		IsActive:         true,
	}

	// Workflow Task uses Activity 2 and provides task-specific key & value
	taskInput := map[string]interface{}{
		"command": "GET", // untouched default written by BPMN modeler
		"key":     "user:42",
		"value":   "hello-world",
	}

	merged := mergeTemplateChainInput([]*models.ActivityTemplate{activity1, activity2}, taskInput)

	if merged["connectionString"] != "redis://cluster-1:6379" {
		t.Fatalf("expected connectionString 'redis://cluster-1:6379' from Activity 1, got %v", merged["connectionString"])
	}
	if merged["password"] != "s3cr3t" {
		t.Fatalf("expected password 's3cr3t' from Activity 1, got %v", merged["password"])
	}
	if merged["command"] != "SET" {
		t.Fatalf("expected command 'SET' from Activity 1, got %v", merged["command"])
	}
	if merged["ttl"] != "600" {
		t.Fatalf("expected ttl '600' from Activity 2, got %v", merged["ttl"])
	}
	if merged["key"] != "user:42" {
		t.Fatalf("expected task override key 'user:42', got %v", merged["key"])
	}
	if merged["value"] != "hello-world" {
		t.Fatalf("expected task value 'hello-world', got %v", merged["value"])
	}
}

func TestRootActivity_DirectO1ExecutionWithoutChainTraversal(t *testing.T) {
	// 1. Activity 1 is created from base Redis -> RootActivity = "io.camunda.connectors.Redis.v1"
	activity1Payload := []byte(`{
		"id": "activity-1",
		"name": "Activity 1",
		"properties": [
			{
				"label": "Redis Host / Connection String",
				"type": "String",
				"value": "redis://prod-redis:6379",
				"editable": false,
				"binding": { "type": "camunda:property", "name": "connectionString" }
			},
			{
				"label": "Command",
				"type": "Dropdown",
				"value": "SET",
				"editable": true,
				"binding": { "type": "camunda:property", "name": "command" }
			}
		]
	}`)

	activity1 := &models.ActivityTemplate{
		ID:               "id-act-1",
		Code:             "activity-1",
		Name:             "Activity 1",
		ParentTemplateId: "io.camunda.connectors.Redis.v1",
		RootActivity:     "io.camunda.connectors.Redis.v1",
		Payload:          activity1Payload,
		IsActive:         true,
	}

	// 2. Activity 2 is created from Activity 1 -> copies Activity 1's RootActivity ("io.camunda.connectors.Redis.v1")
	// and bakes Activity 1's properties/locks at creation time
	activity2RawPayload := []byte(`{
		"id": "activity-2",
		"name": "Activity 2",
		"properties": [
			{
				"label": "Redis Host / Connection String",
				"type": "String",
				"value": "redis://prod-redis:6379",
				"editable": false,
				"binding": { "type": "camunda:property", "name": "connectionString" }
			},
			{
				"label": "Command",
				"type": "Dropdown",
				"value": "SET",
				"editable": true,
				"binding": { "type": "camunda:property", "name": "command" }
			},
			{
				"label": "TTL",
				"type": "String",
				"value": "120",
				"editable": true,
				"binding": { "type": "camunda:property", "name": "ttl" }
			}
		]
	}`)

	activity2 := &models.ActivityTemplate{
		ID:               "id-act-2",
		Code:             "activity-2",
		Name:             "Activity 2",
		ParentTemplateId: activity1.Code,
		RootActivity:     activity1.RootActivity, // Inherited directly from Activity 1!
		Payload:          activity2RawPayload,
		IsActive:         true,
	}

	if activity2.RootActivity != "io.camunda.connectors.Redis.v1" {
		t.Fatalf("expected Activity 2 RootActivity to be 'io.camunda.connectors.Redis.v1', got %s", activity2.RootActivity)
	}

	// 3. At runtime, worker resolves ONLY Activity 2 in O(1) without touching Activity 1
	taskInput := map[string]interface{}{
		"key":   "order:99",
		"value": "confirmed",
	}
	merged := mergeTemplateInput(activity2, taskInput)

	if merged["connectionString"] != "redis://prod-redis:6379" {
		t.Fatalf("expected connectionString 'redis://prod-redis:6379', got %v", merged["connectionString"])
	}
	if merged["command"] != "SET" {
		t.Fatalf("expected command 'SET', got %v", merged["command"])
	}
	if merged["ttl"] != "120" {
		t.Fatalf("expected ttl '120', got %v", merged["ttl"])
	}
	if merged["key"] != "order:99" {
		t.Fatalf("expected key 'order:99', got %v", merged["key"])
	}
}

func TestDynamicInheritedActivityTemplate_InfersConnectorAndExecutes(t *testing.T) {
	// Parent template defining base URL binding
	parentPayload := []byte(`{
		"id": "custom-parent-api",
		"name": "Custom Parent API",
		"properties": [
			{
				"label": "Target URL",
				"type": "String",
				"value": "https://api.daedalus.internal/webhook",
				"editable": false,
				"binding": { "type": "camunda:property", "name": "url" }
			}
		]
	}`)
	parentTpl := &models.ActivityTemplate{
		ID:               "tpl-parent-id",
		Code:             "custom-parent-api",
		Name:             "Custom Parent API",
		ParentTemplateId: "",
		RootActivity:     "", // Empty RootActivity, inferred via payload url!
		Payload:          parentPayload,
		IsActive:         true,
	}

	// Child template inheriting from parent template and adding HTTP method binding
	childPayload := []byte(`{
		"id": "custom-child-webhook",
		"name": "Custom Child Webhook",
		"properties": [
			{
				"label": "HTTP Method",
				"type": "Dropdown",
				"value": "POST",
				"editable": true,
				"binding": { "type": "camunda:property", "name": "method" }
			}
		]
	}`)
	childTpl := &models.ActivityTemplate{
		ID:               "tpl-child-id",
		Code:             "custom-child-webhook",
		Name:             "Custom Child Webhook",
		ParentTemplateId: "custom-parent-api",
		RootActivity:     "",
		Payload:          childPayload,
		IsActive:         true,
	}

	// 1. Verify multi-level property merging across chain [parent, child]
	mergedInput := mergeTemplateChainInput([]*models.ActivityTemplate{parentTpl, childTpl}, map[string]interface{}{
		"body": `{"event": "user_created"}`,
	})

	if mergedInput["url"] != "https://api.daedalus.internal/webhook" {
		t.Fatalf("expected inherited url 'https://api.daedalus.internal/webhook', got %v", mergedInput["url"])
	}
	if mergedInput["method"] != "POST" {
		t.Fatalf("expected method 'POST', got %v", mergedInput["method"])
	}

	// 2. Verify connector inference from merged input map
	inferredBase := db.InferBaseActivityTypeFromMap(mergedInput)
	if inferredBase != "io.camunda.connectors.HttpJson.v1" {
		t.Fatalf("expected inferred base type 'io.camunda.connectors.HttpJson.v1', got '%s'", inferredBase)
	}

	// 3. Verify NormalizeBuiltinActivityType matches connector
	canonical := db.NormalizeBuiltinActivityType(inferredBase)
	if canonical != "io.camunda.connectors.HttpJson.v1" {
		t.Fatalf("expected canonical connector 'io.camunda.connectors.HttpJson.v1', got '%s'", canonical)
	}
}


