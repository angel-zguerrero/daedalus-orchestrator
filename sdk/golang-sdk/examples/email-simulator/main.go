package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	daedalus "github.com/angel-zguerrero/daedalus-orchestrator/sdk/golang-sdk"
)

const BatchSize = 1000

type EmailMessage struct {
	MessageID string `json:"messageId"`
	CompanyID string `json:"companyId"`
	EmailType string `json:"emailType"`
	From      string `json:"from"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	Priority  string `json:"priority"`
	Timestamp int64  `json:"timestamp"`
	Size      int    `json:"size"`
}

type Company struct {
	Code      string
	Name      string
	Employees int
}

func main() {
	sdk := daedalus.NewDaedalusSDK(daedalus.Config{
		URI:      "http://localhost:4000",
		Username: "admin",
		Password: "123456",
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

	// ===== SETUP: Per-Company Email Queues =====
	companies := []Company{
		{Code: "tienda-a", Name: "Big E-commerce", Employees: 100000},
		{Code: "tienda-b", Name: "Small Boutique", Employees: 50},
		{Code: "banco-c", Name: "Financial Bank", Employees: 5000},
		{Code: "empresa-x", Name: "Enterprise Corp", Employees: 50000},
	}

	for _, company := range companies {
		// 1. Upsert company as tenant
		_, err := sdk.AssertTenant(ctx, daedalus.AssertTenantInput{
			Code: company.Code,
			Name: company.Name,
		})
		if err != nil {
			log.Fatalf("💥 Fatal error asserting tenant: %v", err)
		}

		// 2. Upsert exchange
		_, err = sdk.AssertExchange(ctx, daedalus.AssertExchangeInput{
			TenantCode: company.Code,
			Code:       "email-events",
			Name:       "Email Events",
			Type:       "topic",
		})
		if err != nil {
			log.Fatalf("💥 Fatal error asserting exchange: %v", err)
		}

		// 3. Upsert queues
		queueInputs := []daedalus.AssertQueueInput{
			{
				TenantCode: company.Code, Code: "email-transactional", Name: "Email Transactional", Type: "standard", State: "active", VNamespace: "default", MaxAttempts: 3, PriorityType: "normal",
			},
			{
				TenantCode: company.Code, Code: "email-marketing", Name: "Email Marketing", Type: "standard", State: "active", VNamespace: "default", MaxAttempts: 3, PriorityType: "normal",
			},
			{
				TenantCode: company.Code, Code: "email-report", Name: "Email Report", Type: "standard", State: "active", VNamespace: "default", MaxAttempts: 3, PriorityType: "normal",
			},
		}

		for _, qi := range queueInputs {
			if _, err := sdk.AssertQueue(ctx, qi); err != nil {
				log.Fatalf("💥 Fatal error asserting queue %s: %v", qi.Code, err)
			}
		}

		// 4. Bindings
		bindingInputs := []daedalus.AssertBindingInput{
			{
				TenantCode: company.Code, Code: "transactional", ExchangeCode: "email-events", QueueCode: "email-transactional", Pattern: "transactional.*", VNamespace: "default", BindingType: "classic",
			},
			{
				TenantCode: company.Code, Code: "marketing", ExchangeCode: "email-events", QueueCode: "email-marketing", Pattern: "marketing.*", VNamespace: "default", BindingType: "classic",
			},
			{
				TenantCode: company.Code, Code: "report", ExchangeCode: "email-events", QueueCode: "email-report", Pattern: "report.*", VNamespace: "default", BindingType: "classic",
			},
		}

		for _, bi := range bindingInputs {
			if _, err := sdk.AssertBinding(ctx, bi); err != nil {
				log.Fatalf("💥 Fatal error asserting binding %s: %v", bi.Code, err)
			}
		}
	}

	// ===== PUBLISH: Black Friday Email Surge =====

	log.Println("📧 CloudMail Pro: Black Friday Surge...")

	// Tienda A (Big): 5K transactional + 10K marketing
	log.Println("🛍️  Tienda A: 5K confirmations + 10K Black Friday promos")

	publishBatch(ctx, sdk, 5000, func(i int) daedalus.PublishMessageInput {
		payload, _ := json.Marshal(EmailMessage{
			MessageID: fmt.Sprintf("tienda-a-trans-%d", i),
			CompanyID: "tienda-a",
			EmailType: "transactional",
			From:      "orders@tienda-a.com",
			To:        fmt.Sprintf("customer-%d@gmail.com", i),
			Subject:   fmt.Sprintf("Order Confirmation #%d", i),
			Priority:  "urgent",
			Timestamp: time.Now().UnixMilli(),
			Size:      5000,
		})
		return daedalus.PublishMessageInput{
			TenantCode:                     "tienda-a",
			ExchangeCode:                   "email-events",
			RoutingKeyOrPatternOrQueueCode: "transactional.order-confirmation",
			Content:                        payload,
			VNamespace:                     "default",
			ContentType:                    "application/json",
		}
	})

	publishBatch(ctx, sdk, 10000, func(i int) daedalus.PublishMessageInput {
		payload, _ := json.Marshal(EmailMessage{
			MessageID: fmt.Sprintf("tienda-a-mkt-%d", i),
			CompanyID: "tienda-a",
			EmailType: "marketing",
			From:      "marketing@tienda-a.com",
			To:        fmt.Sprintf("customer-%d@gmail.com", i),
			Subject:   "🔥 BLACK FRIDAY: 50% OFF EVERYTHING",
			Priority:  "normal",
			Timestamp: time.Now().UnixMilli(),
			Size:      50000,
		})
		return daedalus.PublishMessageInput{
			TenantCode:                     "tienda-a",
			ExchangeCode:                   "email-events",
			RoutingKeyOrPatternOrQueueCode: "marketing.black-friday",
			Content:                        payload,
			VNamespace:                     "default",
			ContentType:                    "application/json",
		}
	})

	// Tienda B (Small): 5K transactional + 10K marketing (ISOLATED)
	log.Println("🏪 Tienda B: 5K confirmations + 10K promos (ISOLATED from A)")

	publishBatch(ctx, sdk, 5000, func(i int) daedalus.PublishMessageInput {
		payload, _ := json.Marshal(EmailMessage{
			MessageID: fmt.Sprintf("tienda-b-trans-%d", i),
			CompanyID: "tienda-b",
			EmailType: "transactional",
			From:      "orders@tienda-b.com",
			To:        fmt.Sprintf("customer-%d@gmail.com", i),
			Subject:   "Order Confirmation",
			Priority:  "urgent",
			Timestamp: time.Now().UnixMilli(),
			Size:      5000,
		})
		return daedalus.PublishMessageInput{
			TenantCode:                     "tienda-b",
			ExchangeCode:                   "email-events",
			RoutingKeyOrPatternOrQueueCode: "transactional.order-confirmation",
			Content:                        payload,
			VNamespace:                     "default",
			ContentType:                    "application/json",
		}
	})

	publishBatch(ctx, sdk, 10000, func(i int) daedalus.PublishMessageInput {
		payload, _ := json.Marshal(EmailMessage{
			MessageID: fmt.Sprintf("tienda-b-mkt-%d", i),
			CompanyID: "tienda-b",
			EmailType: "marketing",
			From:      "marketing@tienda-b.com",
			To:        fmt.Sprintf("customer-%d@gmail.com", i),
			Subject:   "Black Friday Deals - Up to 40% Off",
			Priority:  "normal",
			Timestamp: time.Now().UnixMilli(),
			Size:      30000,
		})
		return daedalus.PublishMessageInput{
			TenantCode:                     "tienda-b",
			ExchangeCode:                   "email-events",
			RoutingKeyOrPatternOrQueueCode: "marketing.black-friday-small",
			Content:                        payload,
			VNamespace:                     "default",
			ContentType:                    "application/json",
		}
	})

	// Banco C: OTPs (ISOLATED)
	log.Println("🏦 Banco C: 20K OTP emails (ISOLATED)")

	publishBatch(ctx, sdk, 20000, func(i int) daedalus.PublishMessageInput {
		payload, _ := json.Marshal(EmailMessage{
			MessageID: fmt.Sprintf("banco-c-otp-%d", i),
			CompanyID: "banco-c",
			EmailType: "transactional",
			From:      "security@banco-c.com",
			To:        fmt.Sprintf("client-%d@banco-c.com", i),
			Subject:   fmt.Sprintf("One-Time Password: %d", time.Now().UnixNano()%1000000),
			Priority:  "urgent",
			Timestamp: time.Now().UnixMilli(),
			Size:      2000,
		})
		return daedalus.PublishMessageInput{
			TenantCode:                     "banco-c",
			ExchangeCode:                   "email-events",
			RoutingKeyOrPatternOrQueueCode: "transactional.otp",
			Content:                        payload,
			VNamespace:                     "default",
			ContentType:                    "application/json",
		}
	})

	// ===== WORKERS: Process emails =====

	var transactionalCount int64
	var marketingCount int64
	var reportCount int64

	// Worker 1: Transactional Processor (FAST, critical SLA)
	go func() {
		err := sdk.CreateWorker(ctx, daedalus.WorkerOptions{
			WorkerName: "email-transactional-processor",
			IntervalMs: 100,
			CapacityPolicies: []daedalus.ClaimWorkCapacityPolicy{
				{
					MaxQueueMessages: 100,
					ClaimWorkFilter: &daedalus.ClaimWorkFilter{
						TenantPatterns: []string{"*"},
						QueueCodes:     []string{"email-transactional"},
					},
				},
			},
			OnMessage: func(message daedalus.ClaimedMessage, ack daedalus.AckCallback) error {
				var email EmailMessage
				json.Unmarshal([]byte(message.Message.Content), &email)

				subject := email.Subject
				if len(subject) > 30 {
					subject = subject[:30]
				}
				log.Printf("✉️  [TRANSACTIONAL] %s: To: %s | Subject: %s", email.CompanyID, email.To, subject)

				time.Sleep(10 * time.Millisecond) // Very fast
				count := atomic.AddInt64(&transactionalCount, 1)
				log.Printf("[Worker: email-transactional-processor] Processed message count: %d", count)
				return ack()
			},
		})
		if err != nil && err != context.Canceled {
			log.Printf("❌ Worker error: %v", err)
		}
	}()

	// Worker 2: Marketing Processor
	go func() {
		err := sdk.CreateWorker(ctx, daedalus.WorkerOptions{
			WorkerName: "email-marketing-processor",
			IntervalMs: 100,
			CapacityPolicies: []daedalus.ClaimWorkCapacityPolicy{
				{
					MaxQueueMessages: 100,
					ClaimWorkFilter: &daedalus.ClaimWorkFilter{
						TenantPatterns: []string{"*"},
						QueueCodes:     []string{"email-marketing"},
					},
				},
			},
			OnMessage: func(message daedalus.ClaimedMessage, ack daedalus.AckCallback) error {
				var email EmailMessage
				json.Unmarshal([]byte(message.Message.Content), &email)

				subject := email.Subject
				if len(subject) > 30 {
					subject = subject[:30]
				}
				log.Printf("📢 [MARKETING] %s: To: %s | Subject: %s", email.CompanyID, email.To, subject)

				time.Sleep(20 * time.Millisecond) // Slightly slower
				count := atomic.AddInt64(&marketingCount, 1)
				log.Printf("[Worker: email-marketing-processor] Processed message count: %d", count)
				return ack()
			},
		})
		if err != nil && err != context.Canceled {
			log.Printf("❌ Worker error: %v", err)
		}
	}()

	// Worker 3: Reports/Analytics Processor (SLOW, background)
	go func() {
		err := sdk.CreateWorker(ctx, daedalus.WorkerOptions{
			WorkerName: "email-report-processor",
			IntervalMs: 1000,
			CapacityPolicies: []daedalus.ClaimWorkCapacityPolicy{
				{
					MaxQueueMessages: 100,
					ClaimWorkFilter: &daedalus.ClaimWorkFilter{
						TenantPatterns: []string{"*"},
						QueueCodes:     []string{"email-report"},
					},
				},
			},
			OnMessage: func(message daedalus.ClaimedMessage, ack daedalus.AckCallback) error {
				var email EmailMessage
				json.Unmarshal([]byte(message.Message.Content), &email)

				subject := email.Subject
				if len(subject) > 30 {
					subject = subject[:30]
				}
				log.Printf("📊 [REPORT] %s: Archiving report: %s", email.CompanyID, subject)

				time.Sleep(500 * time.Millisecond) // Slow, background
				count := atomic.AddInt64(&reportCount, 1)
				log.Printf("[Worker: email-report-processor] Processed message count: %d", count)
				return ack()
			},
		})
		if err != nil && err != context.Canceled {
			log.Printf("❌ Worker error: %v", err)
		}
	}()

	log.Println("✅ CloudMail Pro running...")
	log.Println("📧 Processing Black Friday surge across all companies")

	<-ctx.Done()
}

func publishBatch(ctx context.Context, sdk *daedalus.DaedalusSDK, total int, createMessage func(int) daedalus.PublishMessageInput) {
	for offset := 0; offset < total; offset += BatchSize {
		count := BatchSize
		if total-offset < BatchSize {
			count = total - offset
		}

		var wg sync.WaitGroup
		errCh := make(chan error, count)
		for j := 0; j < count; j++ {
			idx := offset + j
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				input := createMessage(i)
				_, err := sdk.PublishMessage(ctx, input)
				if err != nil {
					errCh <- err
				}
			}(idx)
		}
		wg.Wait()
		close(errCh)

		for err := range errCh {
			log.Printf("❌ Publish error: %v", err)
		}

		log.Printf("Published %d/%d", offset+count, total)
	}
}
