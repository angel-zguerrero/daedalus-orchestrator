package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	daedalus "github.com/angel-zguerrero/daedalus-orchestrator/sdk/golang-sdk"
)

func main() {
	sdk := daedalus.NewDaedalusSDK(daedalus.Config{
		URI:      "http://localhost:4000",
		Username: "admin",
		Password: "admin",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sdk.Connect(ctx); err != nil {
		log.Fatalf("💥 Fatal error: %v", err)
	}
	defer sdk.Disconnect()

	// Catch interrupt signal for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("🛑 Shutting down...")
		cancel()
	}()

	log.Println("✅ Worker is running. Press Ctrl+C to stop.")

	err := sdk.CreateWorker(ctx, daedalus.WorkerOptions{
		WorkerName: "Simple Go Worker",
		IntervalMs: 500,
		CapacityPolicies: []daedalus.ClaimWorkCapacityPolicy{
			{
				MaxQueueMessages: 10,
				ClaimWorkFilter:  &daedalus.ClaimWorkFilter{},
			},
		},
		OnMessage: func(message daedalus.ClaimedMessage, ack daedalus.AckCallback) error {
			log.Printf("👷 Processing message: %+v", message)
			log.Printf("📝 Content: %s", message.Message.Content)

			// Simulate processing
			time.Sleep(10 * time.Second)

			// Acknowledge the message after processing
			log.Println("✅ Message processed, sending ACK...")
			return ack()
		},
	})

	if err != nil && err != context.Canceled {
		log.Fatalf("💥 Fatal error: %v", err)
	}
}
