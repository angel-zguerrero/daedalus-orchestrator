package bpmn_test

import (
	"reflect"
	"testing"

	"deadalus-orch/server/internal/pkg/bpmn"

	"github.com/stretchr/testify/assert"
)

func TestResolveVariablePath(t *testing.T) {
	state := map[string]interface{}{
		"name": "Angel",
		"age":  30,
		"user": map[string]interface{}{
			"email": "angel@example.com",
			"address": map[string]interface{}{
				"city": "Buenos Aires",
				"zip":  "1414",
			},
		},
		"items": []interface{}{
			map[string]interface{}{"id": "A1", "price": 100},
			map[string]interface{}{"id": "B2", "price": 250},
		},
		"tags": []string{"urgent", "high-priority"},
	}

	assert.Equal(t, "Angel", bpmn.ResolveVariablePath("name", state))
	assert.Equal(t, 30, bpmn.ResolveVariablePath("age", state))
	assert.Equal(t, "angel@example.com", bpmn.ResolveVariablePath("user.email", state))
	assert.Equal(t, "Buenos Aires", bpmn.ResolveVariablePath("user.address.city", state))
	assert.Equal(t, "1414", bpmn.ResolveVariablePath("user.address.zip", state))

	// Slice indexing
	assert.Equal(t, "A1", bpmn.ResolveVariablePath("items[0].id", state))
	assert.Equal(t, 250, bpmn.ResolveVariablePath("items[1].price", state))
	assert.Equal(t, "urgent", bpmn.ResolveVariablePath("tags[0]", state))
	assert.Equal(t, "high-priority", bpmn.ResolveVariablePath("tags[1]", state))

	// Non-existent paths
	assert.Nil(t, bpmn.ResolveVariablePath("nonExistent", state))
	assert.Nil(t, bpmn.ResolveVariablePath("user.invalidField", state))
	assert.Nil(t, bpmn.ResolveVariablePath("items[99].id", state))
}

func TestEvaluateString(t *testing.T) {
	state := map[string]interface{}{
		"host":    "api.daedalus.dev",
		"version": "v1",
		"user": map[string]interface{}{
			"name": "Angel",
			"role": "admin",
		},
		"retryCount": 3,
		"enabled":    true,
	}

	// Standalone expression: type preservation
	valHost := bpmn.EvaluateString("${host}", state)
	assert.Equal(t, "api.daedalus.dev", valHost)

	valRetry := bpmn.EvaluateString("${retryCount}", state)
	assert.Equal(t, 3, valRetry)
	assert.True(t, reflect.DeepEqual(3, valRetry))

	valEnabled := bpmn.EvaluateString("${enabled}", state)
	assert.Equal(t, true, valEnabled)

	// String interpolation
	url := bpmn.EvaluateString("https://${host}/${version}/users", state)
	assert.Equal(t, "https://api.daedalus.dev/v1/users", url)

	greeting := bpmn.EvaluateString("User: ${user.name} (Role: ${user.role})", state)
	assert.Equal(t, "User: Angel (Role: admin)", greeting)

	// Missing variable in string interpolation -> replaced by empty string
	missing := bpmn.EvaluateString("Value: ${missingVar}", state)
	assert.Equal(t, "Value: ", missing)
}

func TestEvaluateObject(t *testing.T) {
	state := map[string]interface{}{
		"endpoint": "https://api.example.com/v1/notify",
		"auth": map[string]interface{}{
			"token": "bearer-secret-123",
		},
		"payload": map[string]interface{}{
			"userId": "usr_99",
			"active": true,
		},
		"recipients": []interface{}{"admin@example.com", "${userEmail}"},
		"userEmail":  "dev@example.com",
	}

	configObj := map[string]interface{}{
		"url": "${endpoint}",
		"headers": map[string]string{
			"Authorization": "Bearer ${auth.token}",
			"Content-Type":  "application/json",
		},
		"body":       "${payload}",
		"recipients": "${recipients}",
	}

	evaluated := bpmn.EvaluateObject(configObj, state).(map[string]interface{})

	assert.Equal(t, "https://api.example.com/v1/notify", evaluated["url"])

	headers := evaluated["headers"].(map[string]interface{})
	assert.Equal(t, "Bearer bearer-secret-123", headers["Authorization"])
	assert.Equal(t, "application/json", headers["Content-Type"])

	// Object type preservation
	body := evaluated["body"].(map[string]interface{})
	assert.Equal(t, "usr_99", body["userId"])
	assert.Equal(t, true, body["active"])

	// Array evaluation
	recipients := evaluated["recipients"].([]interface{})
	assert.Equal(t, "admin@example.com", recipients[0])
	assert.Equal(t, "dev@example.com", recipients[1])
}

type mockEnvResolver struct {
	data map[string]string
}

func (m *mockEnvResolver) ResolveEnvVar(groupType string, scope string, groupRef string, varKey string) (string, error) {
	key := groupType + "." + scope + "." + groupRef + "." + varKey
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return "", assert.AnError
}

func TestEvaluateConfigAndSecrets(t *testing.T) {
	resolver := &mockEnvResolver{
		data: map[string]string{
			"config.global.code_config.DB_HOST":  "master.db.local",
			"secret.global.sec_global.API_KEY":   "master-secret-key-99",
			"config.tenant.app_tenant.APP_NAME":  "Tenant Store Front",
			"secret.tenant.sec_tenant.DB_PASS":   "decrypted-db-pass-123",
		},
	}

	state := map[string]interface{}{"env": "production"}

	// 1. Config Global
	val1, err := bpmn.EvaluateStringResolvable("${config.global[\"code_config\"][\"DB_HOST\"]}", state, resolver)
	assert.NoError(t, err)
	assert.Equal(t, "master.db.local", val1)

	// 2. Secret Global
	val2, err := bpmn.EvaluateStringResolvable("${secret.global[\"sec_global\"][\"API_KEY\"]}", state, resolver)
	assert.NoError(t, err)
	assert.Equal(t, "master-secret-key-99", val2)

	// 3. Config Tenant
	val3, err := bpmn.EvaluateStringResolvable("${config.tenant[\"app_tenant\"][\"APP_NAME\"]}", state, resolver)
	assert.NoError(t, err)
	assert.Equal(t, "Tenant Store Front", val3)

	// 4. Secret Tenant with single quotes
	val4, err := bpmn.EvaluateStringResolvable("${secret.tenant['sec_tenant']['DB_PASS']}", state, resolver)
	assert.NoError(t, err)
	assert.Equal(t, "decrypted-db-pass-123", val4)

	// 5. Interpolated string with config
	strVal, err := bpmn.EvaluateStringResolvable("Database connection: postgres://${secret.tenant['sec_tenant']['DB_PASS']}@${config.global['code_config']['DB_HOST']}:5432", state, resolver)
	assert.NoError(t, err)
	assert.Equal(t, "Database connection: postgres://decrypted-db-pass-123@master.db.local:5432", strVal)

	// 6. Missing config/secret -> error
	_, errMissing := bpmn.EvaluateStringResolvable("${config.global['code_config']['MISSING_VAR']}", state, resolver)
	assert.Error(t, errMissing)
}

