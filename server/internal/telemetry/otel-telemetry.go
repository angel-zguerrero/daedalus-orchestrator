package telemetry

import (
	"context"
	otel_pkg "deadalus-orch/server/internal/pkg/otel"
	"deadalus-orch/shared/constants"

	"go.opentelemetry.io/otel/sdk/trace"
)

// Init initializes the OpenTelemetry tracing system for the application.
// It creates a background context and then calls otel_pkg.InitTracer to set up
// the tracer provider based on the provided configuration.
// If the tracerServiceName is empty, it defaults to "deadalus-server".
//
// Parameters:
//   - env: The deployment environment (e.g., PRODUCTION, DEVELOPMENT) which can affect tracer behavior.
//   - enableTracer: A boolean indicating whether tracing should be enabled. If false, a no-op tracer is configured.
//   - endpoint: The OTLP gRPC endpoint for the OpenTelemetry collector (e.g., "localhost:4317").
//   - tracerServiceName: The name to be used for the service when reporting traces. Defaults to "deadalus-server".
//
// Returns:
//   - A context.Context (currently a new background context, but could be used for further cancellations or values).
//   - A pointer to the initialized trace.TracerProvider.
//   - An error if the tracer initialization in otel_pkg.InitTracer fails.
func Init(env constants.Env, enableTracer bool, endpoint string, tracerServiceName string) (context.Context, *trace.TracerProvider, error) {
	// Create a new background context. This context might be used for further telemetry setup
	// or passed down to other components that need a base context.
	ctx := context.Background()

	if tracerServiceName == "" {
		tracerServiceName = "deadalus-server" // Default service name if not provided.
	}

	// Initialize the tracer using the otel_pkg.
	tp, err := otel_pkg.InitTracer(ctx, tracerServiceName, env, enableTracer, endpoint)
	if err != nil {
		// If tracer initialization fails, return nil for context and provider, along with the error.
		return nil, nil, err
	}

	// Return the created context, the tracer provider, and nil error on success.
	return ctx, tp, nil
}
