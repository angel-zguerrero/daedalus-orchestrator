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

	ExchangeID string
	QueueID    string

	RoutingKey string //used only for direct exchanges
	Pattern    string //used only for topic exchanges

	XMatch XMatchType //used for headers exchanges

	BindingType BindingType

	// Virtual fields - not stored in database
	ExchangeCode string            `orm:"virtual"`
	Exchange     *Exchange         `orm:"virtual"`
	QueueCode    string            `orm:"virtual"`
	Queue        *Queue            `orm:"virtual"`
	Headers      map[string]string `orm:"virtual"` // Headers for routing, used only for Headers exchange type

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Binding) TableName() string {
	return "bindings"
}
