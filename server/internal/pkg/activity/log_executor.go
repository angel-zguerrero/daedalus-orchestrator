package activity

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

type LogExecutor struct{}

func (e *LogExecutor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	if input == nil {
		input = make(map[string]interface{})
	}

	msg := "Log Activity Executed"
	if m, ok := input["message"].(string); ok && m != "" {
		msg = m
	} else if m, ok := input["logMessage"].(string); ok && m != "" {
		msg = m
	}

	log.Info().
		Fields(input).
		Msg(fmt.Sprintf("📋 [LOG TASK]: %s", msg))

	return map[string]interface{}{
		"status":  "SUCCESS",
		"message": msg,
		"input":   input,
	}, nil
}
