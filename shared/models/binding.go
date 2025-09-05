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

	VNamespace string `orm:"unique-compound:1"`
	Code       string `orm:"unique-compound:1"`

	ExchangeID string `orm:"unique-compound:0"`
	QueueID    string `orm:"unique-compound:0"`

	RoutingKey string //used only for direct exchanges
	Pattern    string //used only for topic exchanges

	XMatch XMatchType //used for headers exchanges

	BindingType BindingType

	CreatedAt time.Time
	UpdatedAt time.Time
}

// BindingWithObjects representa un binding con los objetos Exchange y Queue incluidos
type BindingWithObjects struct {
	ID string

	VNamespace string

	ExchangeID   string
	ExchangeCode string
	Exchange     *Exchange

	QueueID   string
	QueueCode string
	Queue     *Queue

	RoutingKey string //used only for direct exchanges
	Pattern    string //used only for topic exchanges

	XMatch XMatchType

	BindingType BindingType

	Headers   map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Binding) TableName() string {
	return "bindings"
}
