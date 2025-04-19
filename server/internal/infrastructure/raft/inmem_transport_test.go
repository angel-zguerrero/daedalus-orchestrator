package raft_test

import (
	"context"
	"log"
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/raft"
)

func TestInMemoryTransportCommunication(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tA := raft.NewInMemoryTransport("A")
	tB := raft.NewInMemoryTransport("B")

	tA.AddPeer("B", tB)
	tB.AddPeer("A", tA)

	msg := raft.Message{
		From:    "A",
		To:      "B",
		Type:    raft.MsgRequestVote,
		Payload: []byte("vote-request"),
	}

	err := tA.Send(ctx, msg)
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	recvCh, err := tB.Receive(ctx)
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
	nodeA := raft.NewNode("A", raft.NewInMemoryTransport("A"))
	nodeB := raft.NewNode("B", raft.NewInMemoryTransport("B"))
	nodeC := raft.NewNode("C", raft.NewInMemoryTransport("C"))

	// Conectamos los nodos entre sí
	nodeA.Transport.AddPeer("B", nodeB.Transport)
	nodeA.Transport.AddPeer("C", nodeC.Transport)

	nodeB.Transport.AddPeer("A", nodeA.Transport)
	nodeB.Transport.AddPeer("C", nodeC.Transport)

	nodeC.Transport.AddPeer("A", nodeA.Transport)
	nodeC.Transport.AddPeer("B", nodeB.Transport)

	// Lanzamos cada nodo en una goroutine
	go nodeA.Run()
	go nodeB.Run()
	go nodeC.Run()

	log.Println("Raft nodes running in memory...")
	time.Sleep(10 * time.Second)

}
