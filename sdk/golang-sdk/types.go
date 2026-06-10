package daedalus

// Config holds the connection settings for the Daedalus Orchestrator.
type Config struct {
	URI      string
	Username string
	Password string
}

// ClaimWorkFilter defines filter criteria for claiming work from queues.
type ClaimWorkFilter struct {
	TenantCodes            []string
	ExcludeTenantCodes     []string
	TenantPatterns         []string
	ExcludeTenantPatterns  []string
	VNamespaces            []string
	ExcludeVNamespaces     []string
	VNamespacePatterns     []string
	ExcludeVNamespacePatterns []string
	QueueCodes             []string
	ExcludeQueueCodes      []string
	QueuePatterns          []string
	ExcludeQueuePatterns   []string
}

// ClaimWorkCapacityPolicy defines the capacity policy for a worker.
type ClaimWorkCapacityPolicy struct {
	MaxQueueMessages int32
	ClaimWorkFilter  *ClaimWorkFilter
}

// QueueMessage represents a message stored in a queue.
type QueueMessage struct {
	ID          string
	MessageID   string
	Content     string
	ContentType string
	Headers     map[string]string
	QueueID     string
	Priority    int32
	Attempts    int32
	Handler     string
	Parameters  map[string]string
	VNamespace  string
	CreatedAt   string
}

// QueueMessageLease represents a lease held on a claimed queue message.
type QueueMessageLease struct {
	ID             string
	QueueMessageID string
	WorkerID       string
	LeaseUntil     string
}

// ClaimedMessage wraps a message and its lease, as returned by the server.
type ClaimedMessage struct {
	Message                  QueueMessage
	Lease                    QueueMessageLease
	TenantCode               string
	CapacityPolicyIndexMatch int32
}

// AckCallback is a function that acknowledges a claimed message.
type AckCallback func() error

// WorkerOptions configures a worker that consumes messages from the orchestrator.
type WorkerOptions struct {
	WorkerName       string
	CapacityPolicies []ClaimWorkCapacityPolicy
	IntervalMs       int
	OnMessage        func(message ClaimedMessage, ack AckCallback) error
}

// AssertTenantInput defines the parameters for upserting a tenant.
type AssertTenantInput struct {
	Code string
	Name string
}

// AssertExchangeInput defines the parameters for upserting an exchange.
type AssertExchangeInput struct {
	TenantCode string
	Code       string
	Name       string
	Type       string
	VNamespace string
	Headers    map[string]string
}

// AssertQueueInput defines the parameters for upserting a queue.
type AssertQueueInput struct {
	TenantCode                   string
	Code                         string
	Name                         string
	Type                         string
	State                        string
	VNamespace                   string
	DefaultQueueMessageTTL       int32
	DefaultQueueMessageDelayTime int32
	QueueExpires                 int32
	AllowDuplicated              bool
	MaxAttempts                  int32
	MaxQueueSize                 int32
	MaxDeliveringMessages        int32
	// PriorityType: "normal" = strict priority order, "fair" = provide DesiredPriorityThresholds.
	PriorityType              string
	DesiredPriorityThresholds map[int32]int32
	Headers                   map[string]string
}

// AssertBindingInput defines the parameters for upserting a binding.
type AssertBindingInput struct {
	TenantCode          string
	Code                string
	ExchangeCode        string
	QueueCode           string
	TargetExchangeCode  string
	AlternateExchangeCode string
	VNamespace          string
	RoutingKey          string
	Pattern             string
	XMatch              string
	BindingType         string
	TargetExchangeType  string
	Headers             map[string]string
}

// EnqueueMessageInput defines the parameters for enqueueing a message to a queue.
type EnqueueMessageInput struct {
	TenantCode  string
	QueueCode   string
	Content     string
	ContentType string
	VNamespace  string
	Priority    int32
	Handler     string
	Headers     map[string]string
	Parameters  map[string]string
}

// PublishMessageInput defines the parameters for publishing a message via an exchange.
type PublishMessageInput struct {
	TenantCode                    string
	ExchangeCode                  string
	RoutingKeyOrPatternOrQueueCode string
	VNamespace                    string
	Content                       []byte
	ContentType                   string
	Priority                      int32
	Handler                       string
	Headers                       map[string]string
	Parameters                    map[string]string
	MessageID                     string
}
