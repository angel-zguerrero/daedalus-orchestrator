package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	daedalus "github.com/angel-zguerrero/daedalus-orchestrator/sdk/golang-sdk"
)

type ScheduledEmail struct {
	Subject   string `json:"subject"`
	Recipient string `json:"recipient"`
	Body      string `json:"body"`
	Type      string `json:"type"`
}

func main() {
	sdk := daedalus.NewDaedalusSDK(daedalus.Config{
		URI:      "http://localhost:4000",
		Username: "admin",
		Password: "admin",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sdk.Connect(ctx); err != nil {
		log.Fatalf("💥 Fatal error connecting SDK: %v", err)
	}
	defer sdk.Disconnect()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("🛑 Shutting down...")
		cancel()
	}()

	tenantCode := "acme-corp"
	vnamespace := "default"

	// 1. Assert Tenant
	_, err := sdk.AssertTenant(ctx, daedalus.AssertTenantInput{
		Code: tenantCode,
		Name: "Acme Corporation",
	})
	if err != nil {
		log.Fatalf("💥 Fatal error asserting tenant: %v", err)
	}

	// 2. Assert Exchange
	_, err = sdk.AssertExchange(ctx, daedalus.AssertExchangeInput{
		TenantCode: tenantCode,
		Code:       "email-events",
		Name:       "Email Events Exchange",
		Type:       "topic",
		VNamespace: vnamespace,
	})
	if err != nil {
		log.Fatalf("💥 Fatal error asserting exchange: %v", err)
	}

	// 3. Assert Queues
	_, err = sdk.AssertQueue(ctx, daedalus.AssertQueueInput{
		TenantCode:  tenantCode,
		Code:        "email-delayed",
		Name:        "Delayed Email Queue",
		Type:        "standard",
		State:       "active",
		VNamespace:  vnamespace,
		MaxAttempts: 3,
	})
	if err != nil {
		log.Fatalf("💥 Fatal error asserting queue email-delayed: %v", err)
	}

	_, err = sdk.AssertQueue(ctx, daedalus.AssertQueueInput{
		TenantCode:  tenantCode,
		Code:        "weekly-reports",
		Name:        "Weekly Reports Queue",
		Type:        "standard",
		State:       "active",
		VNamespace:  vnamespace,
		MaxAttempts: 3,
	})
	if err != nil {
		log.Fatalf("💥 Fatal error asserting queue weekly-reports: %v", err)
	}

	// 4. Assert Bindings
	_, err = sdk.AssertBinding(ctx, daedalus.AssertBindingInput{
		TenantCode:   tenantCode,
		Code:         "delayed-emails-binding",
		ExchangeCode: "email-events",
		QueueCode:    "email-delayed",
		Pattern:      "email.delayed.*",
		VNamespace:   vnamespace,
		BindingType:  "classic",
	})
	if err != nil {
		log.Fatalf("💥 Fatal error asserting binding: %v", err)
	}

	_, err = sdk.AssertBinding(ctx, daedalus.AssertBindingInput{
		TenantCode:   tenantCode,
		Code:         "weekly-reports-binding",
		ExchangeCode: "email-events",
		QueueCode:    "weekly-reports",
		Pattern:      "report.weekly.*",
		VNamespace:   vnamespace,
		BindingType:  "classic",
	})
	if err != nil {
		log.Fatalf("💥 Fatal error asserting binding: %v", err)
	}

	// ===== 5. Schedule Delayed Email (OneOff Scheduled Job) =====
	log.Println("📅 Scheduling One-Off delayed welcome email (runs after 5s)...")
	delayedEmailPayload, _ := json.Marshal(ScheduledEmail{
		Subject:   "Welcome to Acme Corp!",
		Recipient: "newuser@example.com",
		Body:      "Thank you for signing up. Here is your onboarding guide.",
		Type:      "one-off-welcome",
	})

	oneOffJob, err := sdk.CreateOneOffScheduledJob(ctx, daedalus.CreateOneOffScheduledJobInput{
		TenantCode:  tenantCode,
		TargetType:  "exchange",
		TargetCode:  "email-events",
		VNamespace:  vnamespace,
		Content:     delayedEmailPayload,
		ContentType: "application/json",
		Handler:     "email.send",
		RunAfter:    "5s",
		Priority:    1,
		Headers:     map[string]string{"routing_key": "email.delayed.welcome"},
	})
	if err != nil {
		log.Fatalf("💥 Fatal error creating OneOff scheduled job: %v", err)
	}
	log.Printf("✅ OneOff Scheduled Job Created! ID: %s | NextRunAt: %s", oneOffJob.ID, oneOffJob.NextRunAt)

	// ===== 6. Schedule Recurring Weekly Report (Recurring Scheduled Job) =====
	log.Println("🔄 Scheduling Recurring Weekly Report (runs every 10s)...")
	recurringReportPayload, _ := json.Marshal(ScheduledEmail{
		Subject:   "Weekly Analytics & Performance Summary",
		Recipient: "executive-team@example.com",
		Body:      "Attached is the weekly automated report.",
		Type:      "recurring-weekly-report",
	})

	recurringJob, err := sdk.CreateRecurringScheduledJob(ctx, daedalus.CreateRecurringScheduledJobInput{
		TenantCode:  tenantCode,
		TargetType:  "queue",
		TargetCode:  "weekly-reports",
		VNamespace:  vnamespace,
		Content:     recurringReportPayload,
		ContentType: "application/json",
		Handler:     "report.generate",
		Every:       "10s",
		Priority:    2,
	})
	if err != nil {
		log.Fatalf("💥 Fatal error creating Recurring scheduled job: %v", err)
	}
	log.Printf("✅ Recurring Scheduled Job Created! ID: %s | Every: %s | NextRunAt: %s", recurringJob.ID, recurringJob.Every, recurringJob.NextRunAt)

	// ===== 7. List Scheduled Jobs =====
	jobs, _, err := sdk.ListScheduledJobs(ctx, daedalus.ListScheduledJobsInput{
		TenantCode: tenantCode,
		VNamespace: vnamespace,
		PageSize:   10,
	})
	if err != nil {
		log.Printf("⚠️ Failed to list scheduled jobs: %v", err)
	} else {
		log.Printf("📋 Currently Scheduled Jobs Count: %d", len(jobs))
		for _, j := range jobs {
			log.Printf("   - [%s] ID: %s | Target: %s (%s) | NextRunAt: %s | State: %s",
				j.Type, j.ID, j.TargetCode, j.TargetType, j.NextRunAt, j.State)
		}
	}

	// ===== 8. Start Worker to Process Scheduled Emails =====
	log.Println("🚀 Starting worker to process scheduled email tasks...")

	go func() {
		err := sdk.CreateWorker(ctx, daedalus.WorkerOptions{
			WorkerName: "scheduled-email-worker",
			IntervalMs: 200,
			CapacityPolicies: []daedalus.ClaimWorkCapacityPolicy{
				{
					MaxQueueMessages: 10,
					ClaimWorkFilter: &daedalus.ClaimWorkFilter{
						TenantCodes: []string{tenantCode},
						QueueCodes:  []string{"email-delayed", "weekly-reports"},
					},
				},
			},
			OnMessage: func(claimed daedalus.ClaimedMessage, ack daedalus.AckCallback) error {
				var email ScheduledEmail
				if err := json.Unmarshal([]byte(claimed.Message.Content), &email); err != nil {
					log.Printf("⚠️ Failed to unmarshal email payload: %v", err)
				} else {
					fmt.Printf("\n📩 [WORKER RECEIVED TASK]\n")
					fmt.Printf("   Queue: %s | Handler: %s\n", claimed.Message.QueueID, claimed.Message.Handler)
					fmt.Printf("   To: %s\n", email.Recipient)
					fmt.Printf("   Subject: %s\n", email.Subject)
					fmt.Printf("   Type: %s\n\n", email.Type)
				}
				return ack()
			},
		})
		if err != nil && err != context.Canceled {
			log.Printf("❌ Worker error: %v", err)
		}
	}()

	<-ctx.Done()
}
