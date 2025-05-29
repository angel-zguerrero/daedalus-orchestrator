package raft

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// InMemoryTransport provides an in-memory implementation of a transport layer
// suitable for testing or single-process Raft deployments. It simulates message passing
// between peers within the same process using channels.
// All operations are goroutine-safe.
type InMemoryTransport struct {
	mu sync.RWMutex // Protects access to internal maps and the closed flag.
	// incoming maps a TenantID to another map, where the inner map keys are peer IDs (strings)
	// and values are channels used to send messages to that specific peer for that tenant.
	incoming map[string]map[string]chan Message
	// peers maps a TenantID to a slice of peer IDs (strings) known for that tenant.
	peers  map[string][]string
	closed bool // Flag indicating if the transport has been closed.
}

// NewInMemoryTransport creates and returns a new instance of InMemoryTransport.
// It initializes the internal maps for incoming message channels and peer tracking.
func NewInMemoryTransport() *InMemoryTransport {
	return &InMemoryTransport{
		incoming: make(map[string]map[string]chan Message),
		peers:    make(map[string][]string),
	}
}

// AddPeer registers a new peer for a given TenantID.
// It initializes the necessary data structures for the peer, including its incoming message channel.
// If the TenantID does not exist, it will be created.
//
// Parameters:
//   - TenantID: The identifier for the tenant to which the peer belongs.
//   - peer: The identifier string for the peer being added.
func (t *InMemoryTransport) AddPeer(TenantID string, peer string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Initialize peers map for TenantID if it doesn't exist.
	if t.peers[TenantID] == nil {
		t.peers[TenantID] = []string{}
	}

	// Initialize incoming channels map for TenantID if it doesn't exist.
	if t.incoming[TenantID] == nil {
		t.incoming[TenantID] = make(map[string]chan Message)
	}
	// Initialize the specific peer's incoming message channel if it doesn't exist.
	// The channel has a buffer size of 100.
	if t.incoming[TenantID][peer] == nil {
		t.incoming[TenantID][peer] = make(chan Message, 100)
	}
	// Add the peer to the list of peers for this TenantID.
	t.peers[TenantID] = append(t.peers[TenantID], peer)
}

// Send attempts to send a message to a peer within a given TenantID.
// It looks up the recipient's channel and sends the message.
// The send operation has a timeout of 100 milliseconds.
//
// Parameters:
//   - ctx: A context.Context for cancellation and deadlines.
//   - TenantID: The identifier for the tenant.
//   - msg: The Message to be sent. The `To` field of the message specifies the recipient peer ID.
//
// Returns:
//   - nil if the message is successfully sent.
//   - An error if the transport is closed, the peer is not found, the send times out, or the context is done.
func (t *InMemoryTransport) Send(ctx context.Context, TenantID string, msg Message) error {
	t.mu.RLock() // Use RLock for reading peers and incoming channels.
	defer t.mu.RUnlock()

	if t.closed {
		return errors.New("transport closed")
	}

	// Check if the tenant exists and if the specific peer channel exists.
	tenantChannels, tenantExists := t.incoming[TenantID]
	if !tenantExists {
		return fmt.Errorf("tenant %s not found", TenantID)
	}
	peerTransport, peerExists := tenantChannels[msg.To]
	if !peerExists {
		// Before returning an error, explicitly check if the peer *should* exist
		// by looking in the t.peers map. This helps differentiate between a missing
		// peer registration and a logic error.
		knownPeers := t.peers[TenantID]
		isKnownPeer := false
		for _, p := range knownPeers {
			if p == msg.To {
				isKnownPeer = true
				break
			}
		}
		if !isKnownPeer {
			return fmt.Errorf("peer %s (for tenant %s) not registered with this transport", msg.To, TenantID)
		}
		// If the peer is known but the channel is missing, it's an internal inconsistency.
		return fmt.Errorf("internal error: peer %s (for tenant %s) registered but channel missing", msg.To, TenantID)
	}

	select {
	case peerTransport <- msg:
		return nil
	case <-time.After(100 * time.Millisecond): // Send timeout.
		return fmt.Errorf("timeout sending to %s for tenant %s", msg.To, TenantID)
	case <-ctx.Done(): // Context cancellation.
		return ctx.Err()
	}
}

// Receive returns a read-only channel for receiving messages destined for a specific peer ID within a TenantID.
//
// Parameters:
//   - ctx: A context.Context (currently unused in this method but good practice for interface consistency).
//   - TenantID: The identifier for the tenant.
//   - id: The identifier of the peer whose message channel is to be returned.
//
// Returns:
//   - A <-chan Message from which incoming messages can be read.
//   - An error if the transport is closed or if the peer/tenant does not exist.
func (t *InMemoryTransport) Receive(ctx context.Context, TenantID string, id string) (<-chan Message, error) {
	t.mu.RLock() // Use RLock for reading the incoming map.
	defer t.mu.RUnlock()

	if t.closed {
		return nil, errors.New("transport closed")
	}

	tenantChannels, tenantExists := t.incoming[TenantID]
	if !tenantExists {
		return nil, fmt.Errorf("tenant %s not found for receiving messages", TenantID)
	}
	ch, peerExists := tenantChannels[id]
	if !peerExists {
		return nil, fmt.Errorf("peer %s (for tenant %s) not found for receiving messages", id, TenantID)
	}
	return ch, nil
}

// Close shuts down the InMemoryTransport.
// It closes all active incoming message channels for all tenants and peers.
// Subsequent calls to Send or Receive on a closed transport will return an error.
// Close is idempotent.
//
// Returns:
//   - nil. Errors during channel closing are not propagated but handled internally.
func (t *InMemoryTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.closed {
		for tenantID, mch := range t.incoming {
			for peerID, ch := range mch {
				close(ch) // Close each peer's message channel.
				delete(t.incoming[tenantID], peerID) // Optional: clean up map entries after closing.
			}
			delete(t.incoming, tenantID)
		}
		t.peers = make(map[string][]string) // Clear peers map.
		t.closed = true
	}
	return nil
}

// GetPeers returns a slice of peer identifiers for a given TenantID.
//
// Parameters:
//   - TenantID: The identifier for the tenant whose peers are requested.
//
// Returns:
//   - A slice of strings, where each string is a peer ID. Returns nil if the TenantID is not found.
//     The returned slice is a copy and can be modified by the caller without affecting the transport's internal state.
func (t *InMemoryTransport) GetPeers(TenantID string) []string {
	t.mu.RLock() // Use RLock for reading the peers map.
	defer t.mu.RUnlock()

	if t.closed { // Or if peers for TenantID is nil
		return nil
	}
	// Return a copy to prevent external modification of the internal slice.
	tenantPeers, ok := t.peers[TenantID]
	if !ok {
		return nil
	}
	peersCopy := make([]string, len(tenantPeers))
	copy(peersCopy, tenantPeers)
	return peersCopy
}
