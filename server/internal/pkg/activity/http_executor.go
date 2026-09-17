package activity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPExecutor struct{}

func (e *HTTPExecutor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	if input == nil {
		input = make(map[string]interface{})
	}

	targetURL := ""
	if u, ok := input["url"].(string); ok && u != "" {
		targetURL = u
	} else if u, ok := input["endpoint"].(string); ok && u != "" {
		targetURL = u
	}
	if targetURL == "" {
		targetURL = "https://httpbin.org/get"
	}

	method := "GET"
	if m, ok := input["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	// Normalize localhost/127.0.0.1 if calling local host services from containers
	targetURL = strings.ReplaceAll(targetURL, "localhost", "127.0.0.1")

	var bodyReader io.Reader
	if bodyData, hasBody := input["body"]; hasBody && bodyData != nil {
		switch v := bodyData.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		default:
			b, _ := json.Marshal(v)
			bodyReader = bytes.NewReader(b)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request for %s: %w", targetURL, err)
	}

	if headersMap, ok := input["headers"].(map[string]interface{}); ok {
		for k, v := range headersMap {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}
	if req.Header.Get("Content-Type") == "" && bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request to %s failed: %w", targetURL, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	resultMap := map[string]interface{}{
		"status":     "SUCCESS",
		"statusCode": resp.StatusCode,
		"url":        targetURL,
		"method":     method,
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var decoded map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &decoded); err == nil {
			resultMap["response"] = decoded
		} else {
			resultMap["response"] = string(bodyBytes)
		}
		return resultMap, nil
	}

	return nil, fmt.Errorf("http request to %s returned error status %d: %s", targetURL, resp.StatusCode, string(bodyBytes))
}
