package dragonboat

import "encoding/gob"

func init() {
	gob.Register(Command{})
	gob.Register(RWK_Command{})
	gob.Register(DDL_Command{})
}

// ----BEGINNING RW Type ---- //
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

// ----END RW Type ---- //

// ----BEGINNING DDL Type ---- //
type DDL_FC_Type int

const (
	Add_CF_Op DDL_FC_Type = iota
	Remove_CF_Op
)

type DDL_Command struct {
	ColumnFamilyName string
	Op               DDL_FC_Type
}

// ----END DDL Type ---- //

type Command_Type int

const (
	RW Command_Type = iota
	DLL_FC
)

type Command struct {
	Type Command_Type
	CMD  any
}
