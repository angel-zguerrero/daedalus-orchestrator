package raft_test

import (
	"context"
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/raft"
	"deadalus-orch/shared/constants"

	"github.com/stretchr/testify/require"
)

func TestInMemoryTransportCommunication(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	TenantId := constants.MasterTenant

	tt := raft.NewInMemoryTransport()

	tt.AddPeer(TenantId, "B")
	tt.AddPeer(TenantId, "A")

	msg := raft.Message{
		From:    "A",
		To:      "B",
		Type:    raft.MsgRequestVote,
		Payload: []byte("vote-request"),
	}

	err := tt.Send(ctx, TenantId, msg)
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	recvCh, err := tt.Receive(ctx, TenantId, "B")
	if err != nil {
		t.Fatalf("failed to receive: %v", err)
	}

	select {
	case received := <-recvCh:
		if received.Type != raft.MsgRequestVote || string(received.Payload) != "vote-request" {
			t.Errorf("unexpected message received: %+v", received)
		}
	case <-ctx.Done():
		t.Fatal("did not receive message in time")
	}

	msgFromB := raft.Message{
		From:    "B",
		To:      "A",
		Type:    raft.MsgRequestVote,
		Payload: []byte("vote-request"),
	}

	err = tt.Send(ctx, TenantId, msgFromB)
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	recvCh, err = tt.Receive(ctx, TenantId, "A")
	if err != nil {
		t.Fatalf("failed to receive: %v", err)
	}

	select {
	case received := <-recvCh:
		if received.Type != raft.MsgRequestVote || string(received.Payload) != "vote-request" {
			t.Errorf("unexpected message received: %+v", received)
		}
	case <-ctx.Done():
		t.Fatal("did not receive message in time")
	}
}

func _TestInMemoryTransportCommunication_Temporal_Test(t *testing.T) {
	TenantId := constants.MasterTenant
	transport := raft.NewInMemoryTransport()
	nodeA := raft.NewNode("A", TenantId, transport)
	nodeB := raft.NewNode("B", TenantId, transport)
	nodeC := raft.NewNode("C", TenantId, transport)

	transport.AddPeer(TenantId, "A")
	transport.AddPeer(TenantId, "B")
	transport.AddPeer(TenantId, "C")

	go nodeA.Run()
	go nodeB.Run()
	go nodeC.Run()

	//time.Sleep(240 * time.Second)

	nodes := []*raft.Node{nodeA, nodeB, nodeC}
	leaderCount := 0
	var leaderID string

	for _, node := range nodes {
		if node.State == raft.Leader {
			leaderCount++
			leaderID = node.ID
		}
	}

	require.Equal(t, 1, leaderCount, "Exactly one node should be elected as leader")
	t.Logf("Leader elected: %s", leaderID)

	// Ensure the other nodes are followers
	for _, node := range nodes {
		if node.ID != leaderID {
			require.Equal(t, raft.Follower, node.State, "Non-leader nodes must be in Follower state")
		}
	}
}
