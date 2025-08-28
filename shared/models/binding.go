package models

import "time"

type BindingType string

const (
	BindingTypeClassic BindingType = "classic"
	BindingTypeDynamic BindingType = "dynamic"
)

type XMatchType string

const (
	XMatchTypeAll XMatchType = "all"
	XMatchTypeAny XMatchType = "any"
)

type Binding struct {
	ID string `orm:"primary-key"`

	VNamespace string

	ExchangeID string `orm:"unique-compound:0"`
	QueueID    string `orm:"unique-compound:0"`

	RoutingKey string //used only for direct exchanges
	Pattern    string //used only for topic exchanges

	XMatch XMatchType //used for headers exchanges

	BindingType BindingType

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Binding) TableName() string {
	return "bindings"
}
