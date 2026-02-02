package models

import (
	"encoding/gob"
	"time"
)

type NodeSchedulerBalancingStatus string

const (
	WaitingForNodeSchedulers NodeSchedulerBalancingStatus = "waiting-for-node-schedulers"
	Balanced                 NodeSchedulerBalancingStatus = "balanced"
)

func init() {
	gob.Register(NodeSchedulerBalancingState{})
}

type NodeSchedulerBalancingState struct {
	ID     string `orm:"primary-key"`
	Status NodeSchedulerBalancingStatus

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (NodeSchedulerBalancingState) TableName() string {
	return "node-scheduler-balancing-states"
}

const NodeSchedulerBalancingStateID = "singleton-id-node-scheduler-balancing-state"
