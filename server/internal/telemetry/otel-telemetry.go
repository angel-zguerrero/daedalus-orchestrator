package telemetry

import (
	"context"
	otel_pkg "deadalus-orch/server/internal/pkg/otel"
	"deadalus-orch/shared/constants"

	"go.opentelemetry.io/otel/sdk/trace"
)

func Init(env constants.Env, enableTracer bool, endpoint string, tracerServiceName string) (context.Context, *trace.TracerProvider, error) {
	ctx := context.Background()

	if tracerServiceName == "" {
		tracerServiceName = "deadalus-server"
	}
	tp, err := otel_pkg.InitTracer(ctx, tracerServiceName, env, enableTracer, endpoint)
	if err != nil {
		return nil, nil, err
	}
	return ctx, tp, nil
}
