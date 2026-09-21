package activity

import (
	"context"
	"testing"
	"time"
)

func TestActivityRegistry_HTTP(t *testing.T) {
	reg := GetRegistry()
	exec, found := reg.Get("io.camunda.connectors.HttpJson.v1")
	if !found {
		t.Fatalf("Expected HTTP executor to be registered")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	input := map[string]interface{}{
		"url":    "https://httpbin.org/get",
		"method": "GET",
	}

	out, err := exec.Execute(ctx, input)
	if err != nil {
		t.Fatalf("HTTP execute failed: %v", err)
	}

	t.Logf("HTTP native output: %+v", out)
	if out["status"] != "SUCCESS" {
		t.Errorf("Expected SUCCESS status, got %v", out["status"])
	}
}

func TestActivityRegistry_Redis(t *testing.T) {
	reg := GetRegistry()
	exec, found := reg.Get("io.camunda.connectors.Redis.v1")
	if !found {
		t.Fatalf("Expected Redis executor to be registered")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test 1: Valid Redis SET from XML template properties
	input := map[string]interface{}{
		"connectionString": "redis://127.0.0.1:6379",
		"command":          "SET",
		"key":              "test_native_xml_key",
		"value":            "native_xml_redis_val_999",
	}

	out, err := exec.Execute(ctx, input)
	if err != nil {
		t.Skipf("Redis server not reachable at 127.0.0.1:6379: %v", err)
		return
	}

	t.Logf("Redis native output: %+v", out)
	if out["status"] != "SUCCESS" {
		t.Errorf("Expected SUCCESS status, got %v", out["status"])
	}

	// Test 3: Redis SET with TTL
	ttlInput := map[string]interface{}{
		"connectionString": "redis://127.0.0.1:6379",
		"command":          "SET",
		"key":              "test_ttl_key",
		"value":            "temp_value",
		"ttl":              "60",
	}
	outTTL, errTTL := exec.Execute(ctx, ttlInput)
	if errTTL != nil {
		t.Errorf("Redis SET with TTL failed: %v", errTTL)
	} else {
		t.Logf("Redis TTL output: %+v", outTTL)
		if outTTL["ttl"] != "60" {
			t.Errorf("Expected ttl 60 in output, got %v", outTTL["ttl"])
		}
	}

	// Test 4: Redis EXPIRE command
	expireInput := map[string]interface{}{
		"connectionString": "redis://127.0.0.1:6379",
		"command":          "EXPIRE",
		"key":              "test_ttl_key",
		"ttl":              "120",
	}
	outExpire, errExpire := exec.Execute(ctx, expireInput)
	if errExpire != nil {
		t.Errorf("Redis EXPIRE command failed: %v", errExpire)
	} else {
		t.Logf("Redis EXPIRE output: %+v", outExpire)
	}
}
