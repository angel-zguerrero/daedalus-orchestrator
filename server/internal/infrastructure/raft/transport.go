// transport.go
package raft

import "context"

type MessageType string

const (
	MsgAppendEntries MessageType = "AppendEntries"
	MsgRequestVote   MessageType = "RequestVote"
	MsgVote          MessageType = "Vote"
)

type Message struct {
	From    string
	To      string
	Type    MessageType
	Payload []byte
	Term    int
}

type Transport interface {
	Send(ctx context.Context, TenantID string, msg Message) error
	Receive(ctx context.Context, TenantID string, id string) (<-chan Message, error)
	Close() error
	AddPeer(TenantID string, id string)
	GetPeers(TenantID string) []string
}
