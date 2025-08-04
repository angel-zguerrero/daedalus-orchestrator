package models

import "time"

type ExchangeType string

const (
	Direct     ExchangeType = "direct"
	Fanout     ExchangeType = "fanout"
	Topic      ExchangeType = "topic"
	Headers    ExchangeType = "headers"
	DeadLetter ExchangeType = "dead-letter"
)

// Exchange represents a message exchange with both unique code and compound uniqueness constraint.
// The Code field must be unique across all exchanges.
// Additionally, the combination of Name + VNamespace must be unique across all exchanges.
// This allows multiple exchanges with the same name as long as they are in different namespaces.
type Exchange struct {
	ID string `orm:"primary-key"`

	// Code is unique identifier for the exchange, used for upsert operations
	Code string `orm:"unique"`

	// Name is part of the compound uniqueness constraint with index 0
	Name string `orm:"unique-compound:0"`

	Type ExchangeType

	// VNamespace is part of the compound uniqueness constraint with index 0
	// Together with Name, they form a unique constraint: (Name, VNamespace)
	VNamespace string `orm:"unique-compound:0"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Exchange) TableName() string {
	return "exchanges"
}
