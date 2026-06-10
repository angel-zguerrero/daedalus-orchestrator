package daedalus

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	authpb "github.com/angel-zguerrero/daedalus-orchestrator/sdk/golang-sdk/proto/auth"
	bindingpb "github.com/angel-zguerrero/daedalus-orchestrator/sdk/golang-sdk/proto/binding"
	exchangepb "github.com/angel-zguerrero/daedalus-orchestrator/sdk/golang-sdk/proto/exchange"
	jobworkerpb "github.com/angel-zguerrero/daedalus-orchestrator/sdk/golang-sdk/proto/jobworker"
	queuepb "github.com/angel-zguerrero/daedalus-orchestrator/sdk/golang-sdk/proto/queue"
	tenantpb "github.com/angel-zguerrero/daedalus-orchestrator/sdk/golang-sdk/proto/tenant"
)

// DaedalusSDK is the main client for interacting with the Daedalus Orchestrator.
type DaedalusSDK struct {
	config Config

	conn           *grpc.ClientConn
	authClient     authpb.AuthServiceClient
	jobWorkerClient jobworkerpb.JobWorkerServiceClient
	tenantClient   tenantpb.TenantServiceClient
	exchangeClient exchangepb.ExchangeServiceClient
	queueClient    queuepb.QueueServiceClient
	bindingClient  bindingpb.BindingServiceClient

	token string
	mu    sync.RWMutex
}

// NewDaedalusSDK creates a new SDK instance with the given configuration.
func NewDaedalusSDK(config Config) *DaedalusSDK {
	return &DaedalusSDK{
		config: config,
	}
}

// Connect establishes the gRPC connection and performs initial authentication.
func (sdk *DaedalusSDK) Connect(ctx context.Context) error {
	target := sdk.config.URI
	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimPrefix(target, "https://")

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", target, err)
	}

	sdk.conn = conn
	sdk.authClient = authpb.NewAuthServiceClient(conn)
	sdk.jobWorkerClient = jobworkerpb.NewJobWorkerServiceClient(conn)
	sdk.tenantClient = tenantpb.NewTenantServiceClient(conn)
	sdk.exchangeClient = exchangepb.NewExchangeServiceClient(conn)
	sdk.queueClient = queuepb.NewQueueServiceClient(conn)
	sdk.bindingClient = bindingpb.NewBindingServiceClient(conn)

	// Perform initial login
	if err := sdk.Login(ctx); err != nil {
		conn.Close()
		return fmt.Errorf("initial login failed: %w", err)
	}

	return nil
}

// Disconnect closes all gRPC connections.
func (sdk *DaedalusSDK) Disconnect() error {
	if sdk.conn != nil {
		return sdk.conn.Close()
	}
	return nil
}

// Login authenticates with the orchestrator and stores the JWT token.
func (sdk *DaedalusSDK) Login(ctx context.Context) error {
	log.Printf("🔐 Logging in as %s...", sdk.config.Username)

	resp, err := sdk.authClient.Login(ctx, &authpb.LoginRequest{
		UsernameOrEmail: sdk.config.Username,
		Password:        sdk.config.Password,
	})
	if err != nil {
		log.Printf("❌ Login failed: %v", err)
		return fmt.Errorf("login failed: %w", err)
	}

	sdk.mu.Lock()
	sdk.token = resp.Token
	sdk.mu.Unlock()

	log.Println("✅ Logged in successfully")
	return nil
}

// authCtx returns a context with the Authorization Bearer metadata attached.
func (sdk *DaedalusSDK) authCtx(ctx context.Context) context.Context {
	sdk.mu.RLock()
	token := sdk.token
	sdk.mu.RUnlock()

	if token == "" {
		return ctx
	}
	md := metadata.Pairs("Authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(ctx, md)
}

// AckMessage acknowledges a claimed message by its lease ID.
func (sdk *DaedalusSDK) AckMessage(ctx context.Context, leaseID, tenantCode string) error {
	resp, err := sdk.jobWorkerClient.AckMessage(sdk.authCtx(ctx), &jobworkerpb.AckMessageRequest{
		LeaseID:    leaseID,
		TenantCode: tenantCode,
	})
	if err != nil {
		log.Printf("❌ Failed to ack message: %v", err)
		return fmt.Errorf("ack message failed: %w", err)
	}
	if !resp.Success {
		log.Printf("❌ Ack message failed: %s", resp.Message)
		return fmt.Errorf("ack message failed: %s", resp.Message)
	}
	log.Println("✅ Message acknowledged successfully")
	return nil
}

// AssertTenant upserts a tenant in the orchestrator.
func (sdk *DaedalusSDK) AssertTenant(ctx context.Context, input AssertTenantInput) (*tenantpb.Tenant, error) {
	resp, err := sdk.tenantClient.AssertTenant(sdk.authCtx(ctx), &tenantpb.AssertTenantRequest{
		Code: input.Code,
		Name: input.Name,
	})
	if err != nil {
		log.Printf("❌ Failed to assert tenant: %v", err)
		return nil, fmt.Errorf("assert tenant failed: %w", err)
	}
	log.Printf("✅ Tenant asserted: %s", input.Code)
	return resp.Result, nil
}

// AssertExchange upserts an exchange in the orchestrator.
func (sdk *DaedalusSDK) AssertExchange(ctx context.Context, input AssertExchangeInput) (*exchangepb.Exchange, error) {
	resp, err := sdk.exchangeClient.CreateExchange(sdk.authCtx(ctx), &exchangepb.CreateExchangeRequest{
		TenantCode: input.TenantCode,
		Code:       input.Code,
		Name:       input.Name,
		Type:       input.Type,
		Vnamespace: input.VNamespace,
		Headers:    input.Headers,
	})
	if err != nil {
		log.Printf("❌ Failed to assert exchange: %v", err)
		return nil, fmt.Errorf("assert exchange failed: %w", err)
	}
	log.Printf("✅ Exchange asserted: %s", input.Code)
	return resp.Result, nil
}

// AssertQueue upserts a queue in the orchestrator.
func (sdk *DaedalusSDK) AssertQueue(ctx context.Context, input AssertQueueInput) (*queuepb.Queue, error) {
	queueType := input.Type
	if queueType == "" {
		queueType = "standard"
	}
	state := input.State
	if state == "" {
		state = "active"
	}

	var desiredThresholds map[int32]int32
	if input.PriorityType == "normal" {
		desiredThresholds = map[int32]int32{}
	} else if input.DesiredPriorityThresholds != nil {
		desiredThresholds = input.DesiredPriorityThresholds
	} else {
		desiredThresholds = map[int32]int32{}
	}

	resp, err := sdk.queueClient.CreateQueue(sdk.authCtx(ctx), &queuepb.CreateQueueRequest{
		TenantCode:                   input.TenantCode,
		Code:                         input.Code,
		Name:                         input.Name,
		Type:                         queueType,
		State:                        state,
		Vnamespace:                   input.VNamespace,
		DefaultQueueMessageTTL:       input.DefaultQueueMessageTTL,
		DefaultQueueMessageDelayTime: input.DefaultQueueMessageDelayTime,
		QueueExpires:                 input.QueueExpires,
		AllowDuplicated:              input.AllowDuplicated,
		MaxAttempts:                  input.MaxAttempts,
		MaxQueueSize:                 input.MaxQueueSize,
		MaxDeliveringMessages:        input.MaxDeliveringMessages,
		DesiredPriorityThresholds:    desiredThresholds,
		Headers:                      input.Headers,
	})
	if err != nil {
		log.Printf("❌ Failed to assert queue: %v", err)
		return nil, fmt.Errorf("assert queue failed: %w", err)
	}
	log.Printf("✅ Queue asserted: %s", input.Code)
	return resp.Result, nil
}

// AssertBinding upserts a binding in the orchestrator.
func (sdk *DaedalusSDK) AssertBinding(ctx context.Context, input AssertBindingInput) (*bindingpb.Binding, error) {
	bindingType := input.BindingType
	if bindingType == "" {
		bindingType = "classic"
	}

	resp, err := sdk.bindingClient.CreateBinding(sdk.authCtx(ctx), &bindingpb.CreateBindingRequest{
		TenantCode:          input.TenantCode,
		Code:                input.Code,
		ExchangeCode:        input.ExchangeCode,
		QueueCode:           input.QueueCode,
		TargetExchangeCode:  input.TargetExchangeCode,
		AlternateExchangeCode: input.AlternateExchangeCode,
		Vnamespace:          input.VNamespace,
		RoutingKey:          input.RoutingKey,
		Pattern:             input.Pattern,
		XMatch:              input.XMatch,
		BindingType:         bindingType,
		TargetExchangeType:  input.TargetExchangeType,
		Headers:             input.Headers,
	})
	if err != nil {
		log.Printf("❌ Failed to assert binding: %v", err)
		return nil, fmt.Errorf("assert binding failed: %w", err)
	}
	log.Printf("✅ Binding asserted: %s", input.Code)
	return resp.Result, nil
}

// EnqueueMessage enqueues a message directly into a queue.
// Returns the message ID assigned by the orchestrator.
func (sdk *DaedalusSDK) EnqueueMessage(ctx context.Context, input EnqueueMessageInput) (string, error) {
	contentType := input.ContentType
	if contentType == "" {
		contentType = "text/plain"
	}

	resp, err := sdk.queueClient.EnqueueMessage(sdk.authCtx(ctx), &queuepb.EnqueueMessageRequest{
		TenantCode:  input.TenantCode,
		QueueCode:   input.QueueCode,
		Content:     input.Content,
		ContentType: contentType,
		Vnamespace:  input.VNamespace,
		Priority:    input.Priority,
		Handler:     input.Handler,
		Headers:     input.Headers,
		Parameters:  input.Parameters,
	})
	if err != nil {
		log.Printf("❌ Failed to enqueue message: %v", err)
		return "", fmt.Errorf("enqueue message failed: %w", err)
	}
	return resp.MessageId, nil
}

// PublishMessage publishes a message through an exchange.
// Returns a map of queueCode → messageID for all routed messages.
func (sdk *DaedalusSDK) PublishMessage(ctx context.Context, input PublishMessageInput) (map[string]string, error) {
	contentType := input.ContentType
	if contentType == "" {
		contentType = "text/plain"
	}

	resp, err := sdk.exchangeClient.PublishMessage(sdk.authCtx(ctx), &exchangepb.PublishMessageRequest{
		TenantCode:                    input.TenantCode,
		ExchangeCode:                  input.ExchangeCode,
		RoutingKeyOrPatternOrQueueCode: input.RoutingKeyOrPatternOrQueueCode,
		Vnamespace:                    input.VNamespace,
		Message: &exchangepb.QueueMessage{
			MessageId:   input.MessageID,
			Handler:     input.Handler,
			Priority:    input.Priority,
			Parameters:  input.Parameters,
			Headers:     input.Headers,
			ContentType: contentType,
			Content:     input.Content,
		},
	})
	if err != nil {
		log.Printf("❌ Failed to publish message: %v", err)
		return nil, fmt.Errorf("publish message failed: %w", err)
	}
	if resp.QueueMessages == nil {
		return map[string]string{}, nil
	}
	return resp.QueueMessages, nil
}

// CreateWorker starts a bidirectional streaming worker loop that claims and processes messages.
// This method blocks until the provided context is cancelled.
func (sdk *DaedalusSDK) CreateWorker(ctx context.Context, options WorkerOptions) error {
	intervalMs := options.IntervalMs
	if intervalMs <= 0 {
		intervalMs = 10000
	}
	interval := time.Duration(intervalMs) * time.Millisecond

	workerID := fmt.Sprintf("%s-%d", uuid.New().String(), time.Now().UnixMilli())

	// Track in-flight message counts per capacity policy index.
	currentCounts := make([]int32, len(options.CapacityPolicies))
	var countsMu sync.Mutex

	log.Printf("🚀 Starting worker %s (%s) with %dms interval...", options.WorkerName, workerID, intervalMs)

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Worker stopped by context cancellation")
			return ctx.Err()
		default:
		}

		err := sdk.runWorkerStream(ctx, workerID, options, interval, currentCounts, &countsMu)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("❌ Unexpected error in worker loop: %v", err)
		}

		log.Printf("⏳ Reconnecting in %dms...", intervalMs)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

func (sdk *DaedalusSDK) runWorkerStream(
	ctx context.Context,
	workerID string,
	options WorkerOptions,
	interval time.Duration,
	currentCounts []int32,
	countsMu *sync.Mutex,
) error {
	// Re-login if token is empty
	sdk.mu.RLock()
	hasToken := sdk.token != ""
	sdk.mu.RUnlock()
	if !hasToken {
		log.Println("⚠️ Not authenticated. Attempting to log in...")
		if err := sdk.Login(ctx); err != nil {
			return err
		}
	}

	// Open bidirectional stream
	stream, err := sdk.jobWorkerClient.ClaimWork(sdk.authCtx(ctx))
	if err != nil {
		return fmt.Errorf("failed to open ClaimWork stream: %w", err)
	}

	log.Printf("🔌 Opening bidirectional stream for worker %s...", workerID)

	// Channel to signal stream is done
	streamDone := make(chan struct{})

	// Handle incoming messages from server in a goroutine
	go func() {
		defer close(streamDone)
		for {
			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					log.Println("🔌 Stream ended, will reconnect...")
				} else {
					errStr := err.Error()
					if strings.Contains(errStr, "code = Unauthenticated") {
						log.Println("🔄 Session expired (Unauthenticated). Refreshing token...")
						sdk.mu.Lock()
						sdk.token = ""
						sdk.mu.Unlock()
					} else if strings.Contains(errStr, "code = Canceled") {
						log.Println("🚫 Stream cancelled")
					} else {
						log.Printf("❌ Stream error: %v", err)
					}
				}
				return
			}

			switch m := msg.Message.(type) {
			case *jobworkerpb.ClaimWorkStreamMessage_Ack:
				log.Printf("✅ Connected to server: %s", m.Ack.Knowledge)

			case *jobworkerpb.ClaimWorkStreamMessage_ClaimedMessage:
				claimed := m.ClaimedMessage
				log.Printf("📬 Received message: %s from tenant %s", claimed.Message.ID, claimed.TenantCode)

				if options.OnMessage != nil {
					claimedMsg := ClaimedMessage{
						Message: QueueMessage{
							ID:          claimed.Message.ID,
							MessageID:   claimed.Message.MessageID,
							Content:     claimed.Message.Content,
							ContentType: claimed.Message.ContentType,
							Headers:     claimed.Message.Headers,
							QueueID:     claimed.Message.QueueID,
							Priority:    claimed.Message.Priority,
							Attempts:    claimed.Message.Attempts,
							Handler:     claimed.Message.Handler,
							Parameters:  claimed.Message.Parameters,
							VNamespace:  claimed.Message.VNamespace,
							CreatedAt:   claimed.Message.CreatedAt,
						},
						Lease: QueueMessageLease{
							ID:             claimed.Lease.ID,
							QueueMessageID: claimed.Lease.QueueMessageID,
							WorkerID:       claimed.Lease.WorkerID,
							LeaseUntil:     claimed.Lease.LeaseUntil,
						},
						TenantCode:               claimed.TenantCode,
						CapacityPolicyIndexMatch: claimed.CapacityPolicyIndexMatch,
					}

					policyIdx := claimedMsg.CapacityPolicyIndexMatch
					if policyIdx >= 0 && int(policyIdx) < len(currentCounts) {
						countsMu.Lock()
						currentCounts[policyIdx]++
						countsMu.Unlock()
					}

					ackCallback := func() error {
						err := sdk.AckMessage(ctx, claimed.Lease.ID, claimed.TenantCode)
						if err == nil {
							if policyIdx >= 0 && int(policyIdx) < len(currentCounts) {
								countsMu.Lock()
								currentCounts[policyIdx] = int32(math.Max(0, float64(currentCounts[policyIdx]-1)))
								countsMu.Unlock()
							}
						}
						return err
					}

					go func() {
						if err := options.OnMessage(claimedMsg, ackCallback); err != nil {
							log.Printf("❌ Error in OnMessage handler: %v", err)
						}
					}()
				}
			}
		}
	}()

	// Send initial claim request
	if err := sdk.sendClaimRequest(stream, workerID, options, currentCounts, countsMu); err != nil {
		return err
	}

	// Send claim requests periodically
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			stream.CloseSend()
			return ctx.Err()
		case <-streamDone:
			return nil
		case <-ticker.C:
			if err := sdk.sendClaimRequest(stream, workerID, options, currentCounts, countsMu); err != nil {
				log.Printf("❌ Error sending claim request: %v", err)
				return err
			}
		}
	}
}

func (sdk *DaedalusSDK) sendClaimRequest(
	stream jobworkerpb.JobWorkerService_ClaimWorkClient,
	workerID string,
	options WorkerOptions,
	currentCounts []int32,
	countsMu *sync.Mutex,
) error {
	sysInfo := GetSystemInfo()

	countsMu.Lock()
	policies := make([]*jobworkerpb.ClaimWorkCapacityPolicy, len(options.CapacityPolicies))
	for i, p := range options.CapacityPolicies {
		policy := &jobworkerpb.ClaimWorkCapacityPolicy{
			MaxQueueMessages:     p.MaxQueueMessages,
			CurrentQueueMessages: currentCounts[i],
		}
		if p.ClaimWorkFilter != nil {
			policy.ClaimWorkFilter = &jobworkerpb.ClaimWorkFilter{
				TenantCodes:               p.ClaimWorkFilter.TenantCodes,
				ExcludeTenantCodes:        p.ClaimWorkFilter.ExcludeTenantCodes,
				TenantPatterns:            p.ClaimWorkFilter.TenantPatterns,
				ExcludeTenantPatterns:     p.ClaimWorkFilter.ExcludeTenantPatterns,
				VNamespaces:               p.ClaimWorkFilter.VNamespaces,
				ExcludeVNamespaces:        p.ClaimWorkFilter.ExcludeVNamespaces,
				VNamespacePatterns:        p.ClaimWorkFilter.VNamespacePatterns,
				ExcludeVNamespacePatterns: p.ClaimWorkFilter.ExcludeVNamespacePatterns,
				QueueCodes:               p.ClaimWorkFilter.QueueCodes,
				ExcludeQueueCodes:        p.ClaimWorkFilter.ExcludeQueueCodes,
				QueuePatterns:            p.ClaimWorkFilter.QueuePatterns,
				ExcludeQueuePatterns:     p.ClaimWorkFilter.ExcludeQueuePatterns,
			}
		}
		policies[i] = policy
	}
	countsMu.Unlock()

	return stream.Send(&jobworkerpb.ClaimWorkRequest{
		WorkerID:         workerID,
		WorkerName:       options.WorkerName,
		Information:      sysInfo,
		CapacityPolicies: policies,
	})
}
