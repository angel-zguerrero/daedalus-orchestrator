package models

import "time"

type VNamespace struct {
	ID   string `orm:"primary-key"`
	Name string `orm:"unique"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (VNamespace) TableName() string {
	return "v_namespaces"
}
