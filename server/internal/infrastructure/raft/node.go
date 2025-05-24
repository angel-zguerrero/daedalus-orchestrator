package raft

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
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
	TenantID  string

	mu            sync.Mutex
	currentTerm   int
	votedFor      string
	votesReceived int

	electionResetEvent chan struct{}
	heartbeatCancel    context.CancelFunc // para detener el heartbeat cuando ya no es líder
}

func NewNode(id string, TenantID string, transport Transport) *Node {
	ctx, cancel := context.WithCancel(context.Background())
	return &Node{
		ID:                 id,
		State:              Follower,
		Transport:          transport,
		ctx:                ctx,
		cancel:             cancel,
		TenantID:           TenantID,
		electionResetEvent: make(chan struct{}, 1),
	}
}

func (n *Node) Run() {
	recvCh, err := n.Transport.Receive(n.ctx, n.TenantID, n.ID)
	if err != nil {
		log.Fatal().Err(err).Msg("")
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

	if msg.Term < n.currentTerm {
		log.Info().Msgf("[%s] got OLD message from %s (Type: %s) Term %d", n.ID, msg.From, msg.Type, msg.Term)
		return //some stuck leader send a message?
	}

	switch msg.Type {
	case MsgRequestVote:
		n.handleRequestVote(msg)
	case MsgAppendEntries:
		n.resetElectionTimer()
		if n.State == Candidate || n.State == Leader {
			n.becomeFollower()
		}
	case MsgVote:
		if n.State == Candidate {
			n.votesReceived++
			log.Info().Msgf("[%s] got VOTE from %s (total: %d)", n.ID, msg.From, n.votesReceived)
			if n.hasMajority() {
				n.becomeLeader()
			}
		}
	}
}

func (n *Node) hasMajority() bool {
	peers := n.Transport.GetPeers(n.TenantID)

	count := 0
	for _, peerID := range peers {
		if peerID != n.ID {
			count++
		}
	}

	totalNodes := count + 1 // sumar el propio nodo
	majority := totalNodes/2 + 1
	return n.votesReceived >= majority
}

func (n *Node) handleRequestVote(msg Message) {

	if msg.Term > n.currentTerm {
		n.currentTerm = msg.Term
		n.votedFor = ""
		n.becomeFollower()
	}

	if n.votedFor == "" {
		n.votedFor = msg.From
		n.resetElectionTimer()
		log.Info().Msgf("[%s] voted for %s in term %d", n.ID, msg.From, msg.Term)

		resp := Message{
			From:    n.ID,
			To:      msg.From,
			Type:    MsgVote,
			Payload: []byte("yes"),
			Term:    n.currentTerm,
		}
		go n.Transport.Send(n.ctx, n.TenantID, resp)
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
			log.Info().Msg("[" + n.ID + "] > electionResetEvent >")
			timer.Stop()
			timeout = randomElectionTimeout()
			timer = time.NewTimer(timeout)
		case <-timer.C:
			n.startElection()
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

	if n.State == Leader || n.State == Candidate {
		return
	}

	n.State = Candidate
	n.currentTerm++
	n.votedFor = n.ID
	n.votesReceived = 1
	n.resetElectionTimer()

	log.Info().Msgf("[%s] became CANDIDATE (term %d)", n.ID, n.currentTerm)

	for _, peerID := range n.Transport.GetPeers(n.TenantID) {
		if peerID == n.ID {
			continue
		}
		msg := Message{
			From:    n.ID,
			To:      peerID,
			Type:    MsgRequestVote,
			Payload: []byte("vote-request"),
			Term:    n.currentTerm,
		}
		log.Info().Msgf("[%s] sending ´vote-request´ to [%s]", n.ID, peerID)
		go n.Transport.Send(n.ctx, n.TenantID, msg)
	}
}

func (n *Node) becomeLeader() {
	n.State = Leader
	log.Info().Msgf("[%s] became LEADER for term %d", n.ID, n.currentTerm)
	n.startHeartbeat()
}

func (n *Node) becomeFollower() {
	if n.heartbeatCancel != nil {
		log.Info().Msgf("[%s] stopping heartbeat", n.ID)
		n.heartbeatCancel()
		n.heartbeatCancel = nil
	}
	n.State = Follower
	n.votedFor = ""
	n.votesReceived = 0
	log.Info().Msgf("[%s] became FOLLOWER", n.ID)
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
				for _, peerID := range n.Transport.GetPeers(n.TenantID) {
					if peerID == n.ID {
						continue
					}
					msg := Message{
						From:    n.ID,
						To:      peerID,
						Type:    MsgAppendEntries,
						Payload: []byte("heartbeat"),
						Term:    n.currentTerm,
					}
					log.Info().Msgf("Stuck Heartbeat for ["+n.ID+"] Term %d", n.currentTerm)
					//time.Sleep(10 * time.Second) // test slow connection
					go n.Transport.Send(n.ctx, n.TenantID, msg)
				}
				n.mu.Unlock()
			}
		}
	}()
}

func randomElectionTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}
