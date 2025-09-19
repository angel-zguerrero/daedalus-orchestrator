package models

import "time"

type RoutingHeaderType string

const (
	HeaderTypeExchange     RoutingHeaderType = "exchange"
	HeaderTypeQueue        RoutingHeaderType = "queue"
	HeaderTypeQueueMessage RoutingHeaderType = "queue-message"
	HeaderTypeBinding      RoutingHeaderType = "binding"
)

type RoutingHeader struct {
	ID string `orm:"primary-key"`

	VNamespace string

	HeaderType RoutingHeaderType

	ExchangeID     string `orm:"unique-compound:0"`
	QueueID        string `orm:"unique-compound:0"`
	QueueMessageID string `orm:"unique-compound:0"`
	BindingID      string `orm:"unique-compound:0"`

	Key   string `orm:"unique-compound:0"`
	Value string `orm:"data-only"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (RoutingHeader) TableName() string {
	return "routing_headers"
}
