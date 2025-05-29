package main

import (
	"context"
	"time"

	pb "deadalus-orch/server/internal/infrastructure/common/proto/health/metrics"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

// main is the entry point for a temporary gRPC client application.
// This client is intended for testing the MetricsService by connecting to it,
// sending a GetSystemMetrics request, and logging the response.
//
// It performs the following steps:
// 1. Dials a gRPC connection to "localhost:50052" using an insecure connection.
//    Fatal error occurs if the connection fails.
// 2. Defers the closing of the connection.
// 3. Creates a new MetricsServiceClient using the established connection.
// 4. Creates a context with a timeout of one minute for the RPC call.
//    Defers the cancellation of this context.
// 5. Calls the GetSystemMetrics method on the client with an empty SystemMetricsRequest.
//    Fatal error occurs if the RPC call fails.
// 6. Logs the received SystemMetricsResponse.
//
// This function is typically run as a standalone program for testing purposes.
func main() {
	// Establish a gRPC connection to the server running on localhost:50052.
	// grpc.WithInsecure() is used, meaning no transport security (TLS) is configured.
	// This is common for testing in trusted environments.
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
		log.Fatal().Err(err).Msg("error calling GetSystemMetrics") // Corrected method name in log
	}
	// Log the successful reception of the message and the response content.
	// "Mensaje recibido" means "Message received" in Spanish.
	log.Info().Msg("📨 Message received") // Changed to English for consistency
	log.Info().Interface("systemMetricsResponse", systemMetricsResponse).Msg("")
}
