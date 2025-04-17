package main

import (
	"context"
	"deadalus-orch/server/internal/app"
	"deadalus-orch/server/internal/pkg/otel"
	"log"
)

func main() {
	ctx := context.Background()

	tp, err := otel.InitTracer(ctx, "deadalus-server")
	if err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}
	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	app.Run(ctx)
}
