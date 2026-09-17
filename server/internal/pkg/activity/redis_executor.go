package activity

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type RedisExecutor struct{}

func parseRedisHostPort(uriStr string) string {
	if uriStr == "" {
		return "127.0.0.1:6379"
	}
	uriStr = strings.TrimPrefix(uriStr, "redis://")
	uriStr = strings.TrimPrefix(uriStr, "rediss://")
	if idx := strings.Index(uriStr, "/"); idx != -1 {
		uriStr = uriStr[:idx]
	}
	uriStr = strings.ReplaceAll(uriStr, "localhost", "127.0.0.1")
	uriStr = strings.ReplaceAll(uriStr, "host.docker.internal", "127.0.0.1")
	if !strings.Contains(uriStr, ":") {
		uriStr = uriStr + ":6379"
	}
	return uriStr
}

func (e *RedisExecutor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	if input == nil {
		return nil, fmt.Errorf("redis activity input payload is nil")
	}

	// 1. Extract Connection String (strictly 'connectionString')
	hostPort := "127.0.0.1:6379"
	if cs, ok := input["connectionString"].(string); ok && cs != "" {
		hostPort = parseRedisHostPort(cs)
	}

	// 2. Extract Command (strictly 'command')
	cmdType := ""
	if c, ok := input["command"].(string); ok && c != "" {
		cmdType = strings.ToUpper(strings.TrimSpace(c))
	}
	if cmdType == "" {
		cmdType = "GET"
	}

	// 3. Extract Key (strictly 'key' - REQUIRED)
	cmdKey := ""
	if k, ok := input["key"].(string); ok && k != "" {
		cmdKey = strings.TrimSpace(k)
	}
	if cmdKey == "" {
		return nil, fmt.Errorf("missing required property 'key' for Redis activity execution")
	}

	// 4. Extract Value (strictly 'value')
	cmdVal := ""
	if v, ok := input["value"].(string); ok {
		cmdVal = v
	}

	// 5. Extract TTL (strictly 'ttl')
	ttlStr := ""
	if t, ok := input["ttl"]; ok && t != nil {
		ttlStr = strings.TrimSpace(fmt.Sprintf("%v", t))
		if ttlStr == "<nil>" {
			ttlStr = ""
		}
	}

	// Build RESP protocol payload
	var parts []string
	parts = append(parts, cmdType)
	parts = append(parts, cmdKey)

	switch cmdType {
	case "SET":
		parts = append(parts, cmdVal)
		if ttlStr != "" && ttlStr != "0" {
			parts = append(parts, "EX", ttlStr)
		}
	case "EXPIRE":
		seconds := ttlStr
		if seconds == "" {
			seconds = cmdVal
		}
		if seconds == "" {
			return nil, fmt.Errorf("missing required expiration time ('ttl' or 'value') for EXPIRE command")
		}
		parts = append(parts, seconds)
	case "SETEX":
		if ttlStr == "" {
			return nil, fmt.Errorf("missing required expiration time ('ttl') for SETEX command")
		}
		parts = append(parts, ttlStr, cmdVal)
	case "HSET", "PUBLISH":
		if cmdVal != "" {
			parts = append(parts, cmdVal)
		}
	default:
		if cmdVal != "" && cmdType != "GET" && cmdType != "DEL" {
			parts = append(parts, cmdVal)
		}
	}

	var respBuf strings.Builder
	respBuf.WriteString(fmt.Sprintf("*%d\r\n", len(parts)))
	for _, p := range parts {
		respBuf.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(p), p))
	}

	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", hostPort)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", hostPort, err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(respBuf.String())); err != nil {
		return nil, fmt.Errorf("failed to send command to Redis: %w", err)
	}

	reader := bufio.NewReader(conn)
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response from Redis: %w", err)
	}
	firstLine = strings.TrimSpace(firstLine)

	resultMap := map[string]interface{}{
		"status":  "SUCCESS",
		"key":     cmdKey,
		"command": cmdType,
		"result":  firstLine,
	}

	if ttlStr != "" {
		resultMap["ttl"] = ttlStr
	}

	if strings.HasPrefix(firstLine, "$") {
		// Bulk string response
		valLine, _ := reader.ReadString('\n')
		resultMap["value"] = strings.TrimSpace(valLine)
	} else if strings.HasPrefix(firstLine, "+") {
		resultMap["response"] = strings.TrimPrefix(firstLine, "+")
	} else if strings.HasPrefix(firstLine, ":") {
		resultMap["response"] = strings.TrimPrefix(firstLine, ":")
	}

	// If ttl is provided for other commands like HSET, send follow-up EXPIRE command
	if ttlStr != "" && cmdType != "SET" && cmdType != "EXPIRE" && cmdType != "SETEX" {
		expireBuf := fmt.Sprintf("*3\r\n$6\r\nEXPIRE\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(cmdKey), cmdKey, len(ttlStr), ttlStr)
		_, _ = conn.Write([]byte(expireBuf))
		_, _ = reader.ReadString('\n') // consume response
	}

	return resultMap, nil
}
