package main

import (
	"context"
	"log"
	"time"

	pb "deadalus-orch/server/internal/infrastructure/common/proto/health/metrics"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewMetricsServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	systemMetricsResponse, err := client.GetSystemMetrics(ctx, &pb.SystemMetricsRequest{})
	if err != nil {
		log.Fatalf("error calling SayHello: %v", err)
	}
	log.Printf("📨 Mensaje recibido")
	log.Println((systemMetricsResponse))
}
