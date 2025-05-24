package main

import (
	"context"
	"time"

	pb "deadalus-orch/server/internal/infrastructure/common/proto/health/metrics"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatal().Err(err).Msg("did not connect")
	}
	defer conn.Close()

	client := pb.NewMetricsServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	systemMetricsResponse, err := client.GetSystemMetrics(ctx, &pb.SystemMetricsRequest{})
	if err != nil {
		log.Fatal().Err(err).Msg("error calling SayHello")
	}
	log.Info().Msg("📨 Mensaje recibido")
	log.Info().Interface("systemMetricsResponse", systemMetricsResponse).Msg("")
}
