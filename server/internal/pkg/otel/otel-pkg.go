package otel_pkg

import (
	"context"
	"deadalus-orch/shared/constants"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// InitTracer initializes and configures an OpenTelemetry TracerProvider.
// It sets up an OTLP gRPC exporter to send trace data to an OpenTelemetry collector.
// The behavior of the tracer (e.g., using a batcher or syncer) can vary based on the environment.
// If `enableTracer` is false, a no-op TracerProvider is configured, effectively disabling tracing.
//
// Parameters:
//   - ctx: The context.Context to use for initialization, particularly for the exporter.
//   - serviceName: The name of the service that will be associated with the traces (e.g., "my-app-server").
//   - env: The deployment environment (constants.Env type, e.g., PRODUCTION, DEVELOPMENT, STAGING).
//     This influences the choice of trace processor (Batcher for PRODUCTION, Syncer for DEVELOPMENT/STAGING).
//   - enableTracer: A boolean flag to enable or disable tracing. If false, a no-op provider is returned.
//   - endpoint: The address (host:port) of the OpenTelemetry collector. If empty, "localhost:4317" is used as default.
//
// Returns:
//   - A pointer to the configured sdktrace.TracerProvider.
//   - An error if any part of the initialization fails (e.g., creating the exporter or resource).
//     Returns nil error if a no-op provider is successfully configured or if initialization is successful.
func InitTracer(ctx context.Context, serviceName string, env constants.Env, enableTracer bool, endpoint string) (*sdktrace.TracerProvider, error) {

	if !enableTracer {
		// If tracing is not enabled, configure and set a no-op tracer provider.
		// This allows trace calls throughout the code to function without actually exporting data.
		tp := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(tp)
		log.Info().Msg("🕵️ OTEL tracing is disabled. Using NoOp TracerProvider.")
		return tp, nil
	}

	if endpoint == "" {
		endpoint = "localhost:4317" // Default OTLP gRPC endpoint.
	}

	log.Info().
		Str("otel_collector_endpoint", endpoint).
		Str("service_name", serviceName).
		Str("environment", string(env)).
		Msg("📡 Initializing OTEL Tracer with OTLP gRPC exporter")

	// Configure the OTLP gRPC exporter.
	// otlptracegrpc.WithInsecure() is used here; for production, secure transport (TLS) is recommended.
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),     // Connect to the collector using insecure gRPC.
		otlptracegrpc.WithEndpoint(endpoint), // Set the collector endpoint.
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP gRPC trace exporter: %w", err)
	}

	// Define resources associated with the application. These attributes will be attached to all traces.
	// semconv provides standardized attribute keys.
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),             // e.g., "my-service"
			semconv.DeploymentEnvironmentKey.String(string(env)), // e.g., "production", "development"
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenTelemetry resource: %w", err)
	}

	var tp *sdktrace.TracerProvider

	// Choose the trace processor based on the environment.
	// For PRODUCTION, WithBatcher is generally recommended for performance.
	// For DEVELOPMENT/STAGING, WithSyncer (or WithSimpleSpanProcessor) can be useful for immediate feedback.
	switch env {
	case constants.PRODUCTION:
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter), // Asynchronously exports traces in batches.
			sdktrace.WithResource(res),
		)
	case constants.DEVELOPMENT, constants.STAGING:
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSyncer(exporter), // Synchronously exports each span as it's completed. Good for debugging.
			// Alternative for dev: sdktrace.WithSimpleSpanProcessor(exporter)
			sdktrace.WithResource(res),
		)
	default:
		// Default to Batcher for unknown environments, similar to production.
		log.Warn().Str("environment", string(env)).Msg("Unknown environment specified for OTEL, defaulting to Batcher span processor.")
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
	}

	// Set the global TracerProvider.
	otel.SetTracerProvider(tp)
	log.Info().Msg("✅ OTEL TracerProvider initialized and set globally.")
	return tp, nil
}
