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

type Exchange struct {
	ID   string `orm:"primary-key"`
	Name string `orm:"unique"`

	Type ExchangeType

	VNamespace string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (Exchange) TableName() string {
	return "exchanges"
}
