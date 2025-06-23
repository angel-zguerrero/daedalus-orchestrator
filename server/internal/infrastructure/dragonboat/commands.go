package dragonboat

import "encoding/gob"

// init registers command structs with the gob package for serialization.
// This is necessary for them to be transported over the network or persisted.
func init() {
	gob.Register(Command{})
	gob.Register(WK_Command{})
	gob.Register(RK_Command{})
	gob.Register(DDL_Command{})
	gob.Register(RWK_Command{})
	gob.Register(MCLK_Command{})
}

// ---- Read/Write (RW) Command Types ----

// RW_Type defines the category of a read/write key command.
type RW_Type int

// W_Type defines the specific type of a write operation.
type W_Type int

// R_Type defines the specific type of a read operation.
type R_Type int

// MCL_Type defines the specific type of a maintenance control language operation.
type MCL_Type int

const (
	// Read indicates a read operation.
	Read RW_Type = iota
	// Write indicates a write operation.
	Write
)

const (
	// PutOp represents a standard put (key-value store) operation.
	PutOp W_Type = iota
	// PutOpTTL represents a put operation with a Time-To-Live (TTL).
	PutOpTTL
	// DeleteOp represents a standard delete operation.
	DeleteOp
	// DeleteOpTTL represents a delete operation for a key that might have TTL.
	DeleteOpTTL
)

const (
	// GetOp represents a standard get (key lookup) operation.
	GetOp R_Type = iota
	// GetOpTTL represents a get operation for a key that might have TTL.
	GetOpTTL
	// Search represents a search operation (e.g., by pattern).
	Search
	// SearchTTL represents a search operation for keys that might have TTL.
	SearchTTL
)

const (
	// ClearExpiredTTL represents an operation to clear expired TTL entries.
	ClearExpiredTTL MCL_Type = iota
)

// WK_Command (Write Key Command) defines the structure for commands that modify a single key-value pair.
type WK_Command struct {
	Key              string // The key to write.
	Value            []byte // The value to write.
	ColumnFamilyName string // The name of the column family to operate on.
	TTL              int    // Time-To-Live for the key in seconds (used with PutOpTTL).
	Op               W_Type // The specific write operation type (PutOp, PutOpTTL, DeleteOp, DeleteOpTTL).
}

// RK_Command (Read Key Command) defines the structure for commands that read a single key or search keys.

type Query_Command struct {
	Now     []byte
	Command interface{}
}

type RK_Command struct {
	Key              string // The specific key to read (used with GetOp, GetOpTTL).
	KeyPattern       string // The pattern to search for (used with Search, SearchTTL).
	Cursor           string // The cursor for paginated search results.
	Limit            int64  // The maximum number of results for paginated search.
	ColumnFamilyName string // The name of the column family to operate on.
	TTL              int64  // Expected TTL, or for filtering by TTL (usage may vary).
	Op               R_Type // The specific read operation type (GetOp, GetOpTTL, Search, SearchTTL).
}

// RWK_Command (Read/Write Key Command) is a wrapper command that encapsulates either a read or a write key command.
type RWK_Command struct {
	Op  RW_Type // Indicates if the encapsulated command is a Read or Write.
	CMD any     // The actual command, expected to be either WK_Command or RK_Command.
}

// ---- Maintenance Control Language (MCL) Command Types ----

// MCLK_Command (Maintenance Control Language Key Command) defines the structure for maintenance operations.
type MCLK_Command struct {
	Op  MCL_Type // The specific maintenance operation type (e.g., ClearExpiredTTL).
	CMD any      // The actual command payload for the maintenance operation (can be specific to Op).
}

// ---- Data Definition Language (DDL) Command Types for Column Families (FC) ----

// DDL_FC_Type defines the specific type of a DDL operation related to column families.
type DDL_FC_Type int

const (
	// Add_CF_Op represents an operation to add a new regular column family.
	Add_CF_Op DDL_FC_Type = iota
	// Remove_CF_Op represents an operation to remove an existing column family.
	Remove_CF_Op
	// Add_TTL_CF_Op represents an operation to add a new TTL-enabled column family.
	Add_TTL_CF_Op
	// Remove_TTL_CF_Op represents an operation to remove an existing TTL-enabled column family.
	Remove_TTL_CF_Op
)

// DDL_Command (Data Definition Language Command) defines the structure for DDL operations, primarily for managing column families.
type DDL_Command struct {
	ColumnFamilyName string      // The name of the column family to be added or removed.
	Op               DDL_FC_Type // The specific DDL operation type.
}

// ---- General Command Type ----

// Command_Type defines the top-level category of a command (Read/Write, DDL, or Maintenance).
type Command_Type int

const (
	// RW indicates a Read/Write command, further detailed by RWK_Command.
	RW Command_Type = iota
	// DDL_FC indicates a Data Definition Language command for Column Families, detailed by DDL_Command.
	DDL_FC
	// MCL indicates a Maintenance Control Language command, detailed by MCLK_Command.
	MCL

	SPECIALIZED
)

// Command is the generic wrapper for any type of command sent through the Raft consensus.
// It allows different command structures to be handled uniformly at a higher level.
type Command struct {
	Type Command_Type // The overall type of the command.
	CMD  any          // The actual specific command payload (e.g., RWK_Command, DDL_Command, MCLK_Command).
}
