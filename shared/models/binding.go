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
	ID string `json:"id"`

	VNamespace string `json:"vnamespace"`
	Code       string `json:"code"`

	ExchangeID   string    `json:"exchangeId"`
	ExchangeCode string    `json:"exchangeCode"`
	Exchange     *Exchange `json:"exchange,omitempty"`

	QueueID   string `json:"queueId"`
	QueueCode string `json:"queueCode"`
	Queue     *Queue `json:"queue,omitempty"`

	RoutingKey string `json:"routingKey"` //used only for direct exchanges
	Pattern    string `json:"pattern"`    //used only for topic exchanges

	XMatch XMatchType `json:"xMatch"` //used for headers exchanges

	BindingType BindingType `json:"bindingType"`

	Headers map[string]string `json:"headers,omitempty"` // Headers for routing, used only for Headers exchange type

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Binding) TableName() string {
	return "bindings"
}
