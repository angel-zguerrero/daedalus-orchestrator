package raft

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type InMemoryTransport struct {
	mu       sync.RWMutex
	incoming map[string]map[string]chan Message
	peers    map[string][]string
	closed   bool
}

func NewInMemoryTransport() *InMemoryTransport {
	return &InMemoryTransport{
		incoming: make(map[string]map[string]chan Message),
		peers:    make(map[string][]string),
	}
}

func (t *InMemoryTransport) AddPeer(TenantID string, peer string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.peers[TenantID] == nil {
		t.peers[TenantID] = []string{}
	}

	if t.incoming[TenantID] == nil {
		t.incoming[TenantID] = make(map[string]chan Message)
	}
	if t.incoming[TenantID][peer] == nil {
		t.incoming[TenantID][peer] = make(chan Message, 100)
	}
	t.peers[TenantID] = append(t.peers[TenantID], peer)
}

func (t *InMemoryTransport) Send(ctx context.Context, TenantID string, msg Message) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return errors.New("transport closed")
	}

	peerTransport, ok := t.incoming[TenantID][msg.To]
	if !ok {
		return errors.New("peer is not of type *InMemoryTransport")
	}

	select {
	case peerTransport <- msg:
		return nil
	case <-time.After(100 * time.Millisecond):
		return fmt.Errorf("timeout sending to %s", msg.To)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *InMemoryTransport) Receive(ctx context.Context, TenantID string, id string) (<-chan Message, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.closed {
		return nil, errors.New("transport closed")
	}
	return t.incoming[TenantID][id], nil
}

func (t *InMemoryTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.closed {
		for _, mch := range t.incoming {
			for _, ch := range mch {
				close(ch)
			}
		}
		t.closed = true
	}
	return nil
}
func (t *InMemoryTransport) GetPeers(TenantID string) []string {
	return t.peers[TenantID]
}
