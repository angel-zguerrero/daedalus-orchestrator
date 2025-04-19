package raft

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"
)

type NodeState string

const (
	Follower  NodeState = "Follower"
	Candidate NodeState = "Candidate"
	Leader    NodeState = "Leader"
)

type Node struct {
	ID        string
	State     NodeState
	Transport Transport
	ctx       context.Context
	cancel    context.CancelFunc

	mu            sync.Mutex
	currentTerm   int
	votedFor      string
	votesReceived int

	electionResetEvent chan struct{}
	heartbeatCancel    context.CancelFunc // para detener el heartbeat cuando ya no es líder
}

func NewNode(id string, transport Transport) *Node {
	ctx, cancel := context.WithCancel(context.Background())
	return &Node{
		ID:                 id,
		State:              Follower,
		Transport:          transport,
		ctx:                ctx,
		cancel:             cancel,
		electionResetEvent: make(chan struct{}, 1),
	}
}

func (n *Node) Run() {
	recvCh, err := n.Transport.Receive(n.ctx)
	if err != nil {
		log.Fatal(err)
	}

	go n.runElectionTimer()

	go func() {
		for {
			select {
			case <-n.ctx.Done():
				return
			case msg := <-recvCh:
				n.handleMessage(msg)
			}
		}
	}()
}

func (n *Node) handleMessage(msg Message) {
	n.mu.Lock()
	defer n.mu.Unlock()

	switch msg.Type {
	case MsgRequestVote:
		n.handleRequestVote(msg)
	case MsgAppendEntries:
		n.resetElectionTimer()
		if n.State == Candidate || n.State == Leader {
			n.becomeFollower()
		}
	case "Vote":
		if n.State == Candidate {
			n.votesReceived++
			log.Printf("[%s] got VOTE from %s (total: %d)", n.ID, msg.From, n.votesReceived)
			if n.hasMajority() {
				n.becomeLeader()
			}
		}
	}
}

func (n *Node) hasMajority() bool {
	return n.votesReceived >= (len(n.Transport.GetPeers())+1)/2+1
}

func (n *Node) handleRequestVote(msg Message) {
	if n.votedFor == "" {
		n.votedFor = msg.From
		n.resetElectionTimer()
		log.Printf("[%s] voted for %s", n.ID, msg.From)

		resp := Message{
			From:    n.ID,
			To:      msg.From,
			Type:    "Vote",
			Payload: []byte("yes"),
		}
		go n.Transport.Send(n.ctx, resp)
	}
}

func (n *Node) Stop() {
	n.cancel()
	_ = n.Transport.Close()
}

func (n *Node) runElectionTimer() {
	timeout := randomElectionTimeout()
	timer := time.NewTimer(timeout)

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-n.electionResetEvent:
			timer.Stop()
			timeout = randomElectionTimeout()
			timer = time.NewTimer(timeout)
		case <-timer.C:
			n.startElection()
			return
		}
	}
}

func (n *Node) resetElectionTimer() {
	select {
	case n.electionResetEvent <- struct{}{}:
	default:
	}
}

func (n *Node) startElection() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.State = Candidate
	n.currentTerm++
	n.votedFor = n.ID
	n.votesReceived = 1
	n.resetElectionTimer()

	log.Printf("[%s] became CANDIDATE (term %d)", n.ID, n.currentTerm)

	for peerID := range n.Transport.GetPeers() {
		if peerID == n.ID {
			continue
		}
		msg := Message{
			From:    n.ID,
			To:      peerID,
			Type:    MsgRequestVote,
			Payload: []byte("vote-request"),
		}
		go n.Transport.Send(n.ctx, msg)
	}
}

func (n *Node) becomeLeader() {
	n.State = Leader
	log.Printf("[%s] became LEADER for term %d", n.ID, n.currentTerm)
	n.startHeartbeat()
}

func (n *Node) becomeFollower() {
	if n.heartbeatCancel != nil {
		n.heartbeatCancel()
		n.heartbeatCancel = nil
	}
	n.State = Follower
	n.votedFor = ""
	n.votesReceived = 0
	log.Printf("[%s] became FOLLOWER", n.ID)
	n.resetElectionTimer()
}

func (n *Node) startHeartbeat() {
	ctx, cancel := context.WithCancel(n.ctx)
	n.heartbeatCancel = cancel

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n.mu.Lock()
				if n.State != Leader {
					n.mu.Unlock()
					return
				}
				for peerID := range n.Transport.GetPeers() {
					if peerID == n.ID {
						continue
					}
					msg := Message{
						From:    n.ID,
						To:      peerID,
						Type:    MsgAppendEntries,
						Payload: []byte("heartbeat"),
					}
					go n.Transport.Send(n.ctx, msg)
				}
				n.mu.Unlock()
			}
		}
	}()
}

func randomElectionTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}
