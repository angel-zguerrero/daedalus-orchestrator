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

func InitTracer(ctx context.Context, serviceName string, env constants.Env, enableTracer bool, endpoint string) (*sdktrace.TracerProvider, error) {

	if !enableTracer {
		// No-op tracer provider
		tp := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(tp)
		return tp, nil
	}

	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	log.Info().
		Str("endpoint", endpoint).
		Msg("📡 OTEL Collector")

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Recursos comunes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.DeploymentEnvironmentKey.String(string(env)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var tp *sdktrace.TracerProvider

	switch env {
	case constants.PRODUCTION:
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
	case constants.DEVELOPMENT, constants.STAGING:
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSyncer(exporter),
			sdktrace.WithResource(res),
		)
	default:
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
	}

	otel.SetTracerProvider(tp)
	return tp, nil
}
