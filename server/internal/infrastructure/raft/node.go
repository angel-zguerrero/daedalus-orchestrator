package raft

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// NodeState represents the state of a Raft node (Follower, Candidate, or Leader).
type NodeState string

const (
	// Follower is the state where a node follows a leader.
	Follower NodeState = "Follower"
	// Candidate is the state where a node is attempting to become a leader.
	Candidate NodeState = "Candidate"
	// Leader is the state where a node is the leader of the Raft cluster.
	Leader NodeState = "Leader"
)

// Node represents a Raft node in a cluster. It manages its state, term, votes,
// and communication with other nodes via a Transport layer.
// Each node operates within the scope of a TenantID.
type Node struct {
	ID        string    // Unique identifier for this node.
	State     NodeState // Current state of the node (Follower, Candidate, Leader).
	Transport Transport // The transport layer used for communication with other nodes.
	ctx       context.Context // Context for managing the node's lifecycle.
	cancel    context.CancelFunc // Function to cancel the node's context, used for stopping the node.
	TenantID  string    // Identifier for the tenant this node belongs to.

	mu            sync.Mutex // Protects access to mutable fields like currentTerm, votedFor, votesReceived.
	currentTerm   int        // The current Raft term this node is in.
	votedFor      string     // The ID of the candidate this node voted for in the current term, or "" if none.
	votesReceived int        // Number of votes received by this node if it's a candidate.

	electionResetEvent chan struct{}        // Channel used to signal a reset of the election timer.
	heartbeatCancel    context.CancelFunc // Function to cancel the heartbeat goroutine when the node is no longer a leader.
}

// NewNode creates a new Raft Node with the given ID, TenantID, and Transport.
// It initializes the node as a Follower in term 0.
//
// Parameters:
//   - id: The unique identifier for the new node.
//   - TenantID: The tenant identifier this node will operate under.
//   - transport: The Transport implementation for internode communication.
//
// Returns:
//   - A pointer to the newly created Node.
func NewNode(id string, TenantID string, transport Transport) *Node {
	ctx, cancel := context.WithCancel(context.Background())
	return &Node{
		ID:                 id,
		State:              Follower, // Initial state is Follower.
		Transport:          transport,
		ctx:                ctx,
		cancel:             cancel,
		TenantID:           TenantID,
		electionResetEvent: make(chan struct{}, 1), // Buffered channel to prevent blocking on reset.
		currentTerm:        0,                      // Initial term.
		votedFor:           "",                     // Hasn't voted for anyone yet.
	}
}

// Run starts the main event loop for the Raft node.
// It initializes the election timer and starts a goroutine to listen for incoming messages
// from the transport layer, dispatching them to handleMessage.
// This method will fatal log if it cannot start receiving messages from the transport.
func (n *Node) Run() {
	recvCh, err := n.Transport.Receive(n.ctx, n.TenantID, n.ID)
	if err != nil {
		// If we can't receive messages, the node cannot function.
		log.Fatal().Err(err).Str("nodeID", n.ID).Str("tenantID", n.TenantID).Msg("Failed to start receiving messages on transport")
	}

	go n.runElectionTimer() // Manages election timeouts and starting elections.

	// Goroutine to process incoming messages.
	go func() {
		for {
			select {
			case <-n.ctx.Done(): // Node is stopping.
				return
			case msg, ok := <-recvCh:
				if !ok {
					// Channel closed, indicates transport or node is shutting down.
					log.Info().Str("nodeID", n.ID).Msg("Receive channel closed, stopping message handler.")
					return
				}
				n.handleMessage(msg)
			}
		}
	}()
}

// handleMessage processes a message received from another Raft node.
// It locks the node's state (n.mu) to ensure atomic updates to term, vote, and state.
//
// Key actions:
// 1. Ignores messages with a term older than the node's currentTerm.
// 2. If message term is newer, node updates its currentTerm, resets its vote,
//    and transitions to Follower state.
// 3. Dispatches message to specific handlers (handleRequestVote) based on message Type
//    (MsgRequestVote, MsgAppendEntries, MsgVote).
// 4. For MsgAppendEntries (including heartbeats):
//    - Resets the election timer, as it indicates valid leader communication.
//    - If the node was a Candidate or a Leader (and the message is not from itself),
//      it transitions to Follower state.
// 5. For MsgVote:
//    - If the node is a Candidate, it increments its vote count.
//    - If majority votes are received, it transitions to Leader state.
//
// Parameters:
//   - msg: The Message received from another node.
func (n *Node) handleMessage(msg Message) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// If the message's term is older than the current term, ignore it.
	// This could be a delayed message from a previous leader or candidate.
	if msg.Term < n.currentTerm {
		log.Info().Str("nodeID", n.ID).Str("from", msg.From).Str("type", string(msg.Type)).Int("msgTerm", msg.Term).Int("currentTerm", n.currentTerm).Msg("Received message with older term, ignoring.")
		return
	}

	// If message term is newer, update current term and become follower.
	if msg.Term > n.currentTerm {
		log.Info().Str("nodeID", n.ID).Str("from", msg.From).Int("msgTerm", msg.Term).Int("oldTerm", n.currentTerm).Msg("Received message with newer term, updating term and becoming follower.")
		n.currentTerm = msg.Term
		n.votedFor = "" // Reset vote for the new term.
		// n.State will be set to Follower by becomeFollower()
		n.becomeFollower() // Transition to follower state.
	}


	switch msg.Type {
	case MsgRequestVote:
		n.handleRequestVote(msg)
	case MsgAppendEntries: // This includes heartbeats.
		n.resetElectionTimer() // Valid leader communication, reset timer.
		// If node was Candidate or Leader, but receives valid AppendEntries from a new leader (same or new term already handled),
		// it should become a follower.
		if n.State == Candidate || (n.State == Leader && msg.From != n.ID) { // Leader check ensures it's not its own heartbeat
			log.Info().Str("nodeID", n.ID).Str("currentState", string(n.State)).Msg("Received AppendEntries, becoming follower.")
			n.becomeFollower()
		}
	case MsgVote:
		// Only candidates care about votes.
		if n.State == Candidate {
			n.votesReceived++
			log.Info().Str("nodeID", n.ID).Str("voter", msg.From).Int("totalVotes", n.votesReceived).Msg("Received vote.")
			if n.hasMajority() {
				log.Info().Str("nodeID", n.ID).Msg("Majority votes received, becoming leader.")
				n.becomeLeader()
			}
		}
	}
}

// hasMajority checks if the node, as a candidate, has received enough votes to become a leader.
// It calculates majority based on the number of known peers for its TenantID plus itself.
// This method must be called with the node's mutex (n.mu) held.
//
// Returns:
//   - true if the number of votes received is greater than or equal to the majority, false otherwise.
func (n *Node) hasMajority() bool {
	peers := n.Transport.GetPeers(n.TenantID)
	// Count distinct peers, excluding self if listed.
	// The GetPeers method should ideally return only other peers, or this logic
	// needs to be careful about how peers are registered.
	// Assuming GetPeers returns all registered peers including potentially self.
	// For Raft, total nodes in cluster = len(peers). If self is in peers, it's fine.
	// If self is NOT in peers, then total nodes = len(peers) + 1.
	// Let's assume GetPeers provides a list of all nodes in the configuration for that tenant.
	totalNodes := len(peers)
	if totalNodes == 0 { // Should not happen in a configured cluster.
		// If only self, then it's a single node cluster, becomes leader immediately.
		// This edge case might need specific handling in startElection or NewNode if peers can be empty.
		// For now, assume GetPeers returns all members including self, or that this is handled.
		// If GetPeers doesn't include self, then totalNodes = len(peers)+1.
		// The current implementation of InMemoryTransport.GetPeers returns what was added via AddPeer.
		// Let's assume peers list from GetPeers is the list of *all* nodes for that tenant.
		isSelfInPeers := false
		for _, peerID := range peers {
			if peerID == n.ID {
				isSelfInPeers = true
				break
			}
		}
		if !isSelfInPeers && totalNodes > 0 { // This case should ideally not happen if AddPeer(self) is called.
			// If self is not in the peer list provided by transport, it implies an issue or
			// a specific setup. For safety, let's assume totalNodes is len(peers) if self is in it,
			// or len(peers)+1 if self is not (but then self cannot vote for itself unless it's part of peers).
			// The current logic is: votesReceived starts at 1 (for self).
			// totalNodes should be the total number of voting members.
			// If GetPeers() returns a list of *other* peers, then totalNodes = len(peers) + 1.
			// If GetPeers() returns a list of *all* peers (including self), then totalNodes = len(peers).
			// The InMemoryTransport.AddPeer adds to a list, so it depends on how it's populated.
			// Let's stick to the current logic: totalNodes = len(peers from transport)
			// If n.ID is one of these peers, then totalNodes = len(peers).
			// If n.ID is *not* one of these peers, then it's an external node, which is not typical for this model.
			// The existing code seems to assume GetPeers() returns all peers including self.
			// Re-evaluating: `count` in original code is number of *other* peers.
			// `totalNodes := count + 1` is correct.
			otherPeersCount := 0
			for _, peerID := range peers {
				if peerID != n.ID {
					otherPeersCount++
				}
			}
			totalNodes = otherPeersCount + 1 // Count self + other peers.
		} else if totalNodes == 0 && n.ID != "" { // Single node cluster not yet added as peer
			totalNodes = 1
		}


	} // End of my re-evaluation block, returning to original logic structure.
	
	// Original logic for calculating totalNodes based on GetPeers result:
	// It assumes GetPeers returns a list of all nodes in the tenant's Raft group.
	// If self is part of that list, it's counted. If not, it's implicitly added.
	// This is a bit ambiguous. A clearer way:
	// totalNodes = number of configured voting members.
	// For now, let's assume GetPeers(TenantID) returns all nodes in the cluster for that tenant.
	// And n.votesReceived includes vote for self.
	
	// Simpler:
	// totalNodes is the number of nodes that are supposed to be in this tenant's Raft group.
	// This should come from configuration or peer discovery.
	// GetPeers() returns the list of peers the transport knows about.
	// Let's assume GetPeers() returns all peers *including* self if self is registered.

	// The original code:
	// count := 0
	// for _, peerID := range peers {
	// 	if peerID != n.ID { // Counts *other* peers
	// 		count++
	// 	}
	// }
	// totalNodes := count + 1 // Adds self to the count of other peers. This is correct for total voting members.
	
	// Let's use the original logic for totalNodes calculation as it seems intended.
	otherPeersCount := 0
	for _, peerID := range peers {
		if peerID != n.ID {
			otherPeersCount++
		}
	}
	totalVotingNodes := otherPeersCount + 1


	if totalVotingNodes == 0 { return false } // Avoid division by zero if no peers (should not happen)
	majority := totalVotingNodes/2 + 1
	return n.votesReceived >= majority
}

// handleRequestVote processes a MsgRequestVote message from a candidate.
// It grants a vote if the candidate's term is not older than the current term,
// and if this node has not already voted in the current term or is voting for the same candidate.
// This method must be called with the node's mutex (n.mu) held.
//
// Parameters:
//   - msg: The MsgRequestVote received from the candidate.
func (n *Node) handleRequestVote(msg Message) {
	// msg.Term > n.currentTerm is handled by the caller (handleMessage)
	// which would transition this node to follower and update n.currentTerm.
	// So, here, msg.Term == n.currentTerm.

	// Grant vote if:
	// 1. This node hasn't voted yet in the current term (n.votedFor == "").
	// 2. Or, this node has already voted for the same candidate (n.votedFor == msg.From).
	//    (This handles idempotent nature of vote requests).
	// And, the candidate's log is at least as up-to-date (not implemented in this simplified Raft).
	if n.votedFor == "" || n.votedFor == msg.From {
		n.votedFor = msg.From
		n.resetElectionTimer() // Reset timer because we are acknowledging a potential leader or valid candidate.
		log.Info().Str("nodeID", n.ID).Str("candidateID", msg.From).Int("term", n.currentTerm).Msg("Voted for candidate.")

		resp := Message{
			From:    n.ID,
			To:      msg.From,
			Type:    MsgVote,
			Payload: []byte("vote_granted"), // Payload can be simple confirmation.
			Term:    n.currentTerm,
		}
		// Send vote response in a new goroutine to avoid blocking the message handling loop.
		go func() {
			if err := n.Transport.Send(n.ctx, n.TenantID, resp); err != nil {
				log.Error().Err(err).Str("nodeID", n.ID).Str("recipient", resp.To).Msg("Failed to send vote response.")
			}
		}()
	} else {
		// Already voted for someone else in this term.
		log.Info().Str("nodeID", n.ID).Str("candidateID", msg.From).Str("alreadyVotedFor", n.votedFor).Int("term", n.currentTerm).Msg("Vote request denied, already voted for another candidate in this term.")
	}
}

// Stop gracefully shuts down the Raft node.
// It cancels the node's context, which stops its goroutines (election timer, message handler),
// and then closes the transport layer.
func (n *Node) Stop() {
	log.Info().Str("nodeID", n.ID).Msg("Stopping node.")
	n.cancel() // Signal all internal goroutines to stop.
	if n.heartbeatCancel != nil { // Ensure heartbeat is stopped if it was running.
		n.heartbeatCancel()
	}
	// Attempt to close transport, ignore error as we are shutting down.
	if err := n.Transport.Close(); err != nil {
		log.Error().Err(err).Str("nodeID", n.ID).Msg("Error closing transport during stop.")
	}
}

// runElectionTimer manages the election timeout for the node.
// When the timer expires, it triggers a new election by calling startElection.
// The timer is reset if an electionResetEvent is received (e.g., due to valid leader communication).
// This goroutine stops when the node's context (n.ctx) is done.
func (n *Node) runElectionTimer() {
	// Initial timeout is randomized to prevent multiple nodes from starting elections simultaneously.
	timeoutDuration := randomElectionTimeout()
	timer := time.NewTimer(timeoutDuration)
	defer timer.Stop() // Ensure timer is stopped when goroutine exits.

	log.Debug().Str("nodeID", n.ID).Dur("initialTimeout", timeoutDuration).Msg("Election timer started.")

	for {
		select {
		case <-n.ctx.Done(): // Node is stopping.
			log.Debug().Str("nodeID", n.ID).Msg("Context done, stopping election timer.")
			return
		case <-n.electionResetEvent:
			// Stop the current timer and reset it with a new random timeout.
			if !timer.Stop() {
				// If Stop returns false, the timer has already fired and its channel might contain a value.
				// Drain the channel to prevent the old tick from being processed later.
				// This is important if the reset event comes very close to the timer firing.
				select {
				case <-timer.C: // Drain the timer channel.
				default:        // Channel was already empty or timer was not active.
				}
			}
			timeoutDuration = randomElectionTimeout()
			timer.Reset(timeoutDuration)
			log.Debug().Str("nodeID", n.ID).Dur("newTimeout", timeoutDuration).Msg("Election timer reset.")
		case <-timer.C: // Timer expired.
			log.Info().Str("nodeID", n.ID).Msg("Election timer expired, starting new election.")
			n.startElection()
			// After starting an election, the timer should be reset for the next potential timeout.
			// This happens if the election fails and the node remains a candidate or becomes a follower again.
			// The startElection method itself calls resetElectionTimer if it becomes a candidate.
			// If it becomes a leader, heartbeats take over. If it becomes a follower, reset is also handled.
			// So, resetting it here again is likely redundant if startElection handles all transitions.
			// However, to be safe, if it's still a follower/candidate after startElection (e.g. election failed immediately)
			// then a reset is appropriate.
			// The current startElection resets it when becoming a candidate.
			// If it remains follower, it should also reset.
			// Let's ensure all paths from startElection reset or manage the timer correctly.
			// For now, assume startElection handles the reset logic for its new state.
			// The timer will be reset with a new random timeout if an election reset event is triggered,
			// or if this loop continues and timer.C is selected again (which implies a new timeout starts).
			// It's important that after an election attempt (timer.C), the timer is effectively reset
			// for the *next* election timeout, not immediately re-triggering.
			// This is implicitly handled by `startElection` calling `resetElectionTimer` if it transitions to Candidate.
			// If it becomes leader, this timer loop might become irrelevant until it steps down.
			// If it becomes follower, `becomeFollower` calls `resetElectionTimer`.
			// So, the timer is reset upon state changes.
			// We need to ensure the timer is reset for the *next* cycle if it remains in the same state (e.g. follower)
			// and the election attempt (that would be triggered by this C case) did not change state.
			// But this C case *triggers* startElection.
			// The key is that after timer.C, the timer is effectively inactive until Reset.
			// So, a new timeout should be set for the next cycle.
			timeoutDuration = randomElectionTimeout()
			timer.Reset(timeoutDuration)

		}
	}
}

// resetElectionTimer sends a signal to the electionResetEvent channel if possible,
// effectively requesting the election timer goroutine to reset its timer.
// It uses a non-blocking send to avoid deadlocking if the channel buffer is full
// or if the timer goroutine is not ready to receive.
func (n *Node) resetElectionTimer() {
	select {
	case n.electionResetEvent <- struct{}{}: // Signal to reset the timer.
	default: // Non-blocking: if channel is full or no receiver, do nothing.
		// This prevents deadlock if reset is called multiple times rapidly.
	}
}

// startElection initiates a new election process.
// This method is typically called when a Follower's election timer expires.
//
// Key actions:
// 1. Acquires node lock (n.mu) for state modification.
// 2. Checks if the node is already a Leader; if so, it aborts starting a new election
//    and resets its election timer (as a safeguard).
// 3. Transitions node state to Candidate.
// 4. Increments currentTerm.
// 5. Votes for itself (n.votedFor = n.ID, n.votesReceived = 1).
// 6. Resets its election timer for the new candidacy period.
// 7. Sends MsgRequestVote messages to all other peers registered in its Transport layer
//    for the current TenantID. These messages are sent concurrently.
//
// Note: This method must acquire and release the node's mutex (n.mu) as it modifies shared state
// and is called from the election timer goroutine which does not hold the lock.
func (n *Node) startElection() {
	// n.mu.Lock() and defer n.mu.Unlock() are now handled by the caller (runElectionTimer's timer.C case calls this)
	// This needs to be re-evaluated. If startElection is called from elsewhere, it needs the lock.
	// For safety, let's assume it needs to acquire the lock.
	// However, if runElectionTimer holds the lock when calling this, it would deadlock.
	// The original code had Lock/Unlock here. Let's assume it's called independently or needs its own lock.
	// Re-checked: runElectionTimer does NOT hold the lock when calling startElection. So lock is needed here.
	n.mu.Lock()
	defer n.mu.Unlock()


	// If already a leader, do not start another election.
	// A candidate should also not start a new election if it's already in one for the current term,
	// but it might re-start if an election times out and it's still a candidate.
	// This check is a bit broad. More accurately, a candidate might time out and start a new election in a new term.
	// However, the current logic is: timer expires -> startElection.
	// If it's already a leader, it shouldn't respond to election timeout this way.
	// The election timer should ideally be stopped or ignored by leaders.
	// The current runElectionTimer runs for all states.
	// Let's refine: only followers should react to election timeout by starting election.
	// Candidates are already in an election. Leaders don't time out elections.
	// This implies state check should be at the beginning of timer.C case in runElectionTimer.
	// Or, this function is the first point of check.
	if n.State == Leader {
		log.Debug().Str("nodeID", n.ID).Msg("Already a leader, not starting new election.")
		n.resetElectionTimer() // Reset timer to prevent immediate re-trigger if it's still running
		return
	}

	// Transition to Candidate state.
	n.State = Candidate
	n.currentTerm++          // Increment term for the new election.
	n.votedFor = n.ID        // Vote for self.
	n.votesReceived = 1      // Count self-vote.
	n.resetElectionTimer()   // Reset timer for the new candidacy period.

	log.Info().Str("nodeID", n.ID).Int("term", n.currentTerm).Msg("Became CANDIDATE, starting election.")

	// Send RequestVote messages to all other peers.
	for _, peerID := range n.Transport.GetPeers(n.TenantID) {
		if peerID == n.ID { // Don't send to self.
			continue
		}
		msg := Message{
			From:    n.ID,
			To:      peerID,
			Type:    MsgRequestVote,
			Payload: []byte("vote_request"), // Payload can be simple or carry log index info in full Raft.
			Term:    n.currentTerm,
		}
		log.Debug().Str("nodeID", n.ID).Str("toPeer", peerID).Int("term", n.currentTerm).Msg("Sending RequestVote.")
		// Send in a goroutine to avoid blocking.
		go func(pID string, m Message) {
			if err := n.Transport.Send(n.ctx, n.TenantID, m); err != nil {
				log.Error().Err(err).Str("nodeID", n.ID).Str("recipient", pID).Msg("Failed to send RequestVote.")
			}
		}(peerID, msg)
	}
}

// becomeLeader transitions the node to the Leader state.
// This method is called when a Candidate has received votes from a majority of nodes.
//
// Key actions:
// 1. Checks if the node is already the Leader to prevent redundant actions.
// 2. Sets node state to Leader.
// 3. Logs the transition.
// 4. If a heartbeat goroutine was somehow active (e.g., from a rapid state change),
//    it's cancelled before starting a new one.
// 5. Calls startHeartbeat() to begin sending AppendEntries (heartbeats) to peers,
//    asserting its leadership and preventing new elections.
//
// Note: This method must be called with the node's mutex (n.mu) held, as it modifies n.State
// and calls startHeartbeat which also expects the lock or is called by a lock-holding method.
func (n *Node) becomeLeader() {
	if n.State == Leader { // Already leader, perhaps from a redundant majority signal.
		return
	}
	n.State = Leader
	log.Info().Str("nodeID", n.ID).Int("term", n.currentTerm).Msg("Transitioned to LEADER.")
	// Leaders should not have an election timer running in the same way followers/candidates do.
	// Or rather, electionResetEvent should not lead them to start new elections.
	// The runElectionTimer's C case should check state.
	// Stop any previous heartbeat goroutine if it somehow existed (e.g. from a quick L->F->L transition)
	if n.heartbeatCancel != nil {
		n.heartbeatCancel()
	}
	n.startHeartbeat() // Start sending AppendEntries (heartbeats).
}

// becomeFollower transitions the node to the Follower state.
// This can happen due to several reasons:
// - Discovering a current leader (e.g., receiving an AppendEntries from a valid leader).
// - Discovering a new term (e.g., receiving a message with a higher term).
// - An election ending without the node becoming leader.
//
// Key actions:
// 1. Checks if already a Follower with a reset vote; if so, primarily ensures election timer is reset.
// 2. Logs the transition.
// 3. If the node was previously a Leader, it cancels its heartbeat goroutine.
// 4. Sets node state to Follower.
// 5. Resets n.votedFor to "" (hasn't voted in the current context/term as follower).
// 6. Resets n.votesReceived to 0.
// 7. Resets the election timer, as Followers need to time out and start elections if they don't hear from a leader.
//
// Note: This method must be called with the node's mutex (n.mu) held.
func (n *Node) becomeFollower() {
	if n.State == Follower && n.votedFor == "" { // Already a follower and vote is reset (e.g. new term)
		// Still need to reset election timer if this was triggered by a message.
		n.resetElectionTimer()
		return
	}

	log.Info().Str("nodeID", n.ID).Str("oldState", string(n.State)).Msg("Transitioning to FOLLOWER.")
	// If this node was a leader, stop sending heartbeats.
	if n.heartbeatCancel != nil {
		log.Debug().Str("nodeID", n.ID).Msg("Stopping heartbeat as no longer leader.")
		n.heartbeatCancel()
		n.heartbeatCancel = nil // Clear the cancel function.
	}
	n.State = Follower
	n.votedFor = ""          // Reset vote.
	n.votesReceived = 0      // Reset votes received count.
	n.resetElectionTimer()   // Start/reset election timer as a follower.
}

// startHeartbeat initiates a goroutine that periodically sends heartbeat messages
// (empty AppendEntries RPCs) to all other peers. This is a Leader-specific behavior.
//
// Key actions:
// 1. Creates a new context (hbCtx) for the heartbeat goroutine, derived from the node's main context (n.ctx).
//    A cancel function (hbCancel) for this context is stored in n.heartbeatCancel,
//    allowing the heartbeats to be stopped if the node steps down from Leadership.
// 2. Logs the initiation of heartbeats.
// 3. The goroutine uses a time.Ticker for periodic execution (e.g., every 50ms).
// 4. In each tick, it first checks (with n.mu locked) if the node is still the Leader.
//    If not, the goroutine exits. This is a crucial check to stop heartbeats if state changes.
// 5. Sends an AppendEntries message (acting as a heartbeat) to each peer concurrently.
//    The message includes the Leader's currentTerm.
//
// Note: This method is typically called by becomeLeader() which holds n.mu.
// The heartbeat goroutine itself acquires n.mu when checking n.State.
func (n *Node) startHeartbeat() {
	// Create a new context for this heartbeat instance, derived from the node's main context.
	hbCtx, hbCancel := context.WithCancel(n.ctx)
	n.heartbeatCancel = hbCancel // Store the cancel function so it can be called if node steps down.

	log.Info().Str("nodeID", n.ID).Int("term", n.currentTerm).Msg("Starting heartbeats.")

	go func() {
		// Heartbeat interval is typically much shorter than election timeout.
		ticker := time.NewTicker(50 * time.Millisecond) // Example: 50ms heartbeat interval.
		defer ticker.Stop()

		for {
			select {
			case <-hbCtx.Done(): // Heartbeat context cancelled (node stopped or stepped down).
				log.Info().Str("nodeID", n.ID).Msg("Heartbeat context done, stopping heartbeats.")
				return
			case <-ticker.C:
				// It's crucial to lock here to read n.State and n.currentTerm safely.
				n.mu.Lock()
				if n.State != Leader {
					// Node is no longer leader, stop sending heartbeats.
					// This check is important because the state might change right after the ticker fires.
					log.Info().Str("nodeID", n.ID).Str("state", string(n.State)).Msg("No longer leader, stopping heartbeat loop.")
					n.mu.Unlock()
					return // Exit goroutine. Outer context (hbCtx) should be cancelled by becomeFollower.
				}
				currentTermForHb := n.currentTerm // Capture current term while lock is held.
				n.mu.Unlock() // Unlock before network calls.

				// Send heartbeats to all other peers.
				for _, peerID := range n.Transport.GetPeers(n.TenantID) {
					if peerID == n.ID {
						continue
					}
					msg := Message{
						From:    n.ID,
						To:      peerID,
						Type:    MsgAppendEntries, // Heartbeats are empty AppendEntries messages.
						Payload: []byte("heartbeat"),
						Term:    currentTermForHb,
					}
					// log.Debug().Str("nodeID", n.ID).Str("toPeer", peerID).Int("term", currentTermForHb).Msg("Sending heartbeat.")
					// Send in a new goroutine to avoid blocking the heartbeat loop.
					go func(pID string, m Message) {
						// Use n.ctx for sending, as hbCtx might be cancelled if this node steps down
						// but we still want to try sending the last batch of heartbeats.
						// Or, more correctly, use hbCtx to ensure send attempts also stop if leader steps down.
						if err := n.Transport.Send(hbCtx, n.TenantID, m); err != nil {
							// Don't log excessively for transient errors if peer is down.
							// Consider more sophisticated error handling or logging level.
							// log.Warn().Err(err).Str("nodeID", n.ID).Str("recipient", pID).Msg("Failed to send heartbeat.")
						}
					}(peerID, msg)
				}
			}
		}
	}()
}

// randomElectionTimeout generates a random duration for election timeouts.
// This randomness helps prevent multiple nodes from starting elections simultaneously.
// The timeout is typically between 150ms and 300ms.
//
// Returns:
//   - A time.Duration representing the randomized election timeout.
func randomElectionTimeout() time.Duration {
	// Raft paper suggests 150-300ms.
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}
