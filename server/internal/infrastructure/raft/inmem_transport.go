package raft

import (
	"context"
	"errors"
	"sync"
)

type InMemoryTransport struct {
	id       string
	mu       sync.RWMutex
	incoming chan Message
	peers    map[string]Transport
	closed   bool
}

func NewInMemoryTransport(id string) *InMemoryTransport {
	return &InMemoryTransport{
		id:       id,
		incoming: make(chan Message, 100),
		peers:    make(map[string]Transport),
	}
}

func (t *InMemoryTransport) AddPeer(peerID string, peer Transport) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.peers[peerID] = peer
}

func (t *InMemoryTransport) Send(ctx context.Context, msg Message) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return errors.New("transport closed")
	}

	peer, ok := t.peers[msg.To]
	if !ok {
		return errors.New("peer not found: " + msg.To)
	}
	peerTransport, ok := peer.(*InMemoryTransport)
	if !ok {
		return errors.New("peer is not of type *InMemoryTransport")
	}

	select {
	case peerTransport.incoming <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *InMemoryTransport) Receive(ctx context.Context) (<-chan Message, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.closed {
		return nil, errors.New("transport closed")
	}
	return t.incoming, nil
}

func (t *InMemoryTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.closed {
		close(t.incoming)
		t.closed = true
	}
	return nil
}
func (t *InMemoryTransport) GetPeers() map[string]Transport {
	return t.peers
}
