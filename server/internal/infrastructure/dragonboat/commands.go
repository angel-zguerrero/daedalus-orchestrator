package dragonboat

import "encoding/gob"

func init() {
	gob.Register(Command{})
	gob.Register(WK_Command{})
	gob.Register(RK_Command{})
	gob.Register(DDL_Command{})
	gob.Register(RWK_Command{})
	gob.Register(MCLK_Command{})
}

// ----BEGINNING RW Type ---- //
type RW_Type int
type W_Type int
type R_Type int
type MCL_Type int

const (
	Read RW_Type = iota
	Write
)

const (
	PutOp W_Type = iota
	PutOpTTL
	DeleteOp
	DeleteOpTTL
)

const (
	GetOp R_Type = iota
	GetOpTTL
	Search
	SearchTTL
)

const (
	ClearExpiredTTL MCL_Type = iota
)

type WK_Command struct {
	Key              string
	Value            []byte
	ColumnFamilyName string
	TTL              int
	Op               W_Type
}

type RK_Command struct {
	Key              string
	KeyPatter        string
	cursor           string
	limit            int64
	ColumnFamilyName string
	TTL              int64
	Op               R_Type
}

type RWK_Command struct {
	Op  RW_Type
	CMD any
}

// ----END RW Type ---- //

// ----            ---- //
type MCLK_Command struct {
	Op  MCL_Type
	CMD any
}

// ---- END      --------//

// ----BEGINNING DDL Type ---- //
type DDL_FC_Type int

const (
	Add_CF_Op DDL_FC_Type = iota
	Remove_CF_Op
	Add_TTL_CF_Op
	Remove_TTL_CF_Op
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
	MCL // Maintenance Control Language
)

type Command struct {
	Type Command_Type
	CMD  any
}
