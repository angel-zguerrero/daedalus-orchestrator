// transport.go
// transport.go
package raft

import "context"

// MessageType is a string type representing the type of a Raft message.
type MessageType string

const (
	// MsgAppendEntries represents an AppendEntries RPC, used for log replication and heartbeats.
	MsgAppendEntries MessageType = "AppendEntries"
	// MsgRequestVote represents a RequestVote RPC, used by candidates to request votes during an election.
	MsgRequestVote MessageType = "RequestVote"
	// MsgVote represents a response to a RequestVote RPC, indicating a granted vote.
	MsgVote MessageType = "Vote"
)

// Message represents a communication message exchanged between Raft nodes.
// It includes sender and receiver information, message type, payload, and the Raft term.
type Message struct {
	From    string      // ID of the sender node.
	To      string      // ID of the recipient node.
	Type    MessageType // Type of the Raft message (e.g., MsgAppendEntries, MsgRequestVote).
	Payload []byte      // The actual content of the message (e.g., log entries, vote status).
	Term    int         // The Raft term of the sender when sending this message.
}

// Transport defines the interface for the network transport layer responsible for
// sending and receiving messages between Raft nodes. Implementations could be
// in-memory (for testing), gRPC, HTTP, etc.
// All operations are expected to be tenant-aware, identified by TenantID.
type Transport interface {
	// Send sends a message to a peer within a specific TenantID.
	// Parameters:
	//   - ctx: Context for managing cancellation and deadlines for the send operation.
	//   - TenantID: The identifier of the tenant scope for this message.
	//   - msg: The Message to be sent.
	// Returns:
	//   - An error if the send operation fails (e.g., peer unreachable, timeout, transport closed).
	Send(ctx context.Context, TenantID string, msg Message) error

	// Receive returns a read-only channel for receiving messages destined for a specific node (id)
	// within a given TenantID.
	// Parameters:
	//   - ctx: Context for managing the lifecycle of the receive operation (though often the returned channel handles this).
	//   - TenantID: The identifier of the tenant.
	//   - id: The identifier of the local node for which to receive messages.
	// Returns:
	//   - A <-chan Message from which incoming messages can be read.
	//   - An error if the transport cannot provide a channel for the given ID (e.g., transport closed, ID not registered).
	Receive(ctx context.Context, TenantID string, id string) (<-chan Message, error)

	// Close shuts down the transport layer, releasing any resources it holds (e.g., network connections, channels).
	// After Close is called, other methods should ideally return an error indicating the transport is closed.
	// Returns:
	//   - An error if closing the transport fails, though often this might be nil if cleanup is best-effort.
	Close() error

	// AddPeer informs the transport layer about a new peer for a given TenantID.
	// This allows the transport to establish connections or set up routes if necessary.
	// Parameters:
	//   - TenantID: The identifier of the tenant.
	//   - id: The identifier of the peer to add. For some transports, this might include address information
	//         or be an opaque ID that the transport can resolve.
	AddPeer(TenantID string, id string)

	// GetPeers returns a list of known peer identifiers for a specific TenantID.
	// Parameters:
	//   - TenantID: The identifier of the tenant.
	// Returns:
	//   - A slice of strings, where each string is a peer ID.
	GetPeers(TenantID string) []string
}
