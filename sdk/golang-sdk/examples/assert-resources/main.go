package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
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

	// 1. Assert tenant
	tenant, err := sdk.AssertTenant(ctx, daedalus.AssertTenantInput{
		Code: "my-tenant",
		Name: "My Tenant",
	})
	if err != nil {
		log.Fatalf("💥 Fatal error: %v", err)
	}
	log.Printf("Tenant: %+v", tenant)

	// 2. Assert queue
	queue, err := sdk.AssertQueue(ctx, daedalus.AssertQueueInput{
		TenantCode:   "my-tenant",
		Code:         "my-queue",
		Name:         "My Queue",
		Type:         "standard",
		State:        "active",
		VNamespace:   "default",
		MaxAttempts:  3,
		MaxQueueSize: 10000,
		PriorityType: "normal",
	})
	if err != nil {
		log.Fatalf("💥 Fatal error: %v", err)
	}
	log.Printf("Queue: %+v", queue)

	// Start a worker in a separate goroutine
	go func() {
		err := sdk.CreateWorker(ctx, daedalus.WorkerOptions{
			WorkerName: "Simple Go Worker",
			IntervalMs: 500,
			CapacityPolicies: []daedalus.ClaimWorkCapacityPolicy{
				{
					MaxQueueMessages: 0,
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
			log.Printf("❌ Worker error: %v", err)
		}
	}()

	// 3. Enqueue 1000 messages directly to the queue (batches of 50)
	total := 1000
	batchSize := 50

	log.Printf("📤 Enqueueing %d messages directly to the queue (batch size: %d)...", total, batchSize)
	succeeded := 0
	for i := 0; i < total; i += batchSize {
		count := batchSize
		if total-i < batchSize {
			count = total - i
		}

		var wg sync.WaitGroup
		errCh := make(chan error, count)
		for j := 0; j < count; j++ {
			idx := i + j
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				payload, _ := json.Marshal(map[string]interface{}{
					"index": idx,
					"msg":   fmt.Sprintf("Hello from message %d", idx),
				})
				_, err := sdk.EnqueueMessage(ctx, daedalus.EnqueueMessageInput{
					TenantCode:  "my-tenant",
					QueueCode:   "my-queue",
					VNamespace:  "default",
					Content:     payload,
					ContentType: "application/json",
					Priority:    0,
					Handler:     "my-handler",
				})
				if err != nil {
					errCh <- err
				}
			}(idx)
		}
		wg.Wait()
		close(errCh)

		for err := range errCh {
			log.Printf("❌ Enqueue error: %v", err)
		}

		succeeded += count
		log.Printf("  ✅ %d/%d messages enqueued", succeeded, total)
	}
	log.Printf("✅ Done. %d messages enqueued directly to 'my-queue'.", succeeded)

	log.Println("✅ Worker is running. Press Ctrl+C to stop.")
	log.Println("✅ All resources asserted successfully.")

	// Wait for context cancellation (Ctrl+C)
	<-ctx.Done()
}
