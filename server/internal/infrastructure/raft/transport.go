// transport.go
package raft

import "context"

type MessageType string

const (
	MsgAppendEntries MessageType = "AppendEntries"
	MsgRequestVote   MessageType = "RequestVote"
)

type Message struct {
	From    string
	To      string
	Type    MessageType
	Payload []byte
}

type Transport interface {
	Send(ctx context.Context, msg Message) error
	Receive(ctx context.Context) (<-chan Message, error)
	Close() error
	AddPeer(id string, t Transport)
	GetPeers() map[string]Transport
}
