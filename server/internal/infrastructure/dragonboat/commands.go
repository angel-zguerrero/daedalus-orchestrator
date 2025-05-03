package dragonboat

import "encoding/gob"

func init() {
	gob.Register(Command{})
	gob.Register(RWK_Command{})
}

type RW_Type int

const (
	PutOp RW_Type = iota
	DeleteOp
)

type RWK_Command struct {
	Key              string
	Value            []byte
	ColumnFamilyName string
	Op               RW_Type
}

type Command_Type int

const (
	RW Command_Type = iota
)

type Command struct {
	Type Command_Type
	CMD  any
}
