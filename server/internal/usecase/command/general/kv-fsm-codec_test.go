package general_command

import (
	"bytes"
	"deadalus-orch/server/internal/infrastructure/db"
	commands "deadalus-orch/server/internal/usecase/command"
	"reflect"
	"testing"
	"time"
)

func TestFSMCommand_BinaryCodec_RW_Write(t *testing.T) {
	cmd := FSM_Command{
		Now:  time.Now().UnixNano(),
		Type: RW,
		CMD: RWK_Command{
			Op: Write,
			CMD: WK_Command{
				Op:                 PutOp,
				Key:                "test-key-123",
				Value:              []byte("hello-world-payload"),
				ColumnFamilyName:   "queues",
				ColumnFamilySector: "default",
				TTL:                60,
			},
		},
	}

	var buf bytes.Buffer
	err := cmd.EncodeTo(&buf)
	if err != nil {
		t.Fatalf("EncodeTo failed: %v", err)
	}

	var decoded FSM_Command
	err = decoded.DecodeFrom(buf.Bytes())
	if err != nil {
		t.Fatalf("DecodeFrom failed: %v", err)
	}

	if decoded.Now != cmd.Now {
		t.Errorf("Now mismatch: got %d, want %d", decoded.Now, cmd.Now)
	}
	if decoded.Type != cmd.Type {
		t.Errorf("Type mismatch: got %v, want %v", decoded.Type, cmd.Type)
	}

	decodedRW, ok := decoded.CMD.(RWK_Command)
	if !ok {
		t.Fatalf("Expected RWK_Command, got %T", decoded.CMD)
	}
	expectedRW := cmd.CMD.(RWK_Command)
	if decodedRW.Op != expectedRW.Op {
		t.Errorf("RW Op mismatch: got %v, want %v", decodedRW.Op, expectedRW.Op)
	}

	decodedW := decodedRW.CMD.(WK_Command)
	expectedW := expectedRW.CMD.(WK_Command)
	if !reflect.DeepEqual(decodedW, expectedW) {
		t.Errorf("WK_Command mismatch: got %+v, want %+v", decodedW, expectedW)
	}
}

func TestFSMCommand_BinaryCodec_RW_Read(t *testing.T) {
	cmd := FSM_Command{
		Now:  time.Now().UnixNano(),
		Type: RW,
		CMD: RWK_Command{
			Op: Read,
			CMD: RK_Command{
				Op:                 Search,
				Key:                "key-1",
				KeyPattern:         "key-*",
				Cursor:             "cursor-xyz",
				Limit:              100,
				ColumnFamilyName:   "messages",
				ColumnFamilySector: "sector1",
				TTL:                3600,
			},
		},
	}

	var buf bytes.Buffer
	err := cmd.EncodeTo(&buf)
	if err != nil {
		t.Fatalf("EncodeTo failed: %v", err)
	}

	var decoded FSM_Command
	err = decoded.DecodeFrom(buf.Bytes())
	if err != nil {
		t.Fatalf("DecodeFrom failed: %v", err)
	}

	if !reflect.DeepEqual(decoded, cmd) {
		t.Errorf("Decoded mismatch: got %+v, want %+v", decoded, cmd)
	}
}

func TestFSMCommand_BinaryCodec_DDL(t *testing.T) {
	cmd := FSM_Command{
		Now:  123456789,
		Type: DDL_FC,
		CMD: DDL_Command{
			Op:               Add_CF_Op,
			ColumnFamilyName: "new-cf",
		},
	}

	var buf bytes.Buffer
	if err := cmd.EncodeTo(&buf); err != nil {
		t.Fatalf("EncodeTo failed: %v", err)
	}

	var decoded FSM_Command
	if err := decoded.DecodeFrom(buf.Bytes()); err != nil {
		t.Fatalf("DecodeFrom failed: %v", err)
	}

	if !reflect.DeepEqual(decoded, cmd) {
		t.Errorf("Decoded mismatch: got %+v, want %+v", decoded, cmd)
	}
}

func TestQueryCommand_BinaryCodec(t *testing.T) {
	q := Query_Command{
		Now: 987654321,
		Command: RK_Command{
			Op:                 GetOp,
			Key:                "my-query-key",
			ColumnFamilyName:   "cf1",
			ColumnFamilySector: "sec1",
		},
	}

	var buf bytes.Buffer
	if err := q.EncodeTo(&buf); err != nil {
		t.Fatalf("EncodeTo failed: %v", err)
	}

	var decoded Query_Command
	if err := decoded.DecodeFrom(buf.Bytes()); err != nil {
		t.Fatalf("DecodeFrom failed: %v", err)
	}

	if !reflect.DeepEqual(decoded, q) {
		t.Errorf("Decoded mismatch: got %+v, want %+v", decoded, q)
	}
}

func BenchmarkFSMCommand_BinaryCodec_Encode(b *testing.B) {
	cmd := FSM_Command{
		Now:  time.Now().UnixNano(),
		Type: RW,
		CMD: RWK_Command{
			Op: Write,
			CMD: WK_Command{
				Op:                 PutOp,
				Key:                "test-key-123",
				Value:              []byte("hello-world-payload-data-for-benchmarking"),
				ColumnFamilyName:   "queues",
				ColumnFamilySector: "default",
				TTL:                60,
			},
		},
	}

	var buf bytes.Buffer
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = cmd.EncodeTo(&buf)
	}
}

func BenchmarkFSMCommand_BinaryCodec_Decode(b *testing.B) {
	cmd := FSM_Command{
		Now:  time.Now().UnixNano(),
		Type: RW,
		CMD: RWK_Command{
			Op: Write,
			CMD: WK_Command{
				Op:                 PutOp,
				Key:                "test-key-123",
				Value:              []byte("hello-world-payload-data-for-benchmarking"),
				ColumnFamilyName:   "queues",
				ColumnFamilySector: "default",
				TTL:                60,
			},
		},
	}

	var buf bytes.Buffer
	_ = cmd.EncodeTo(&buf)
	data := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var decoded FSM_Command
		_ = decoded.DecodeFrom(data)
	}
}

type dummyTestRepoCmd struct {
	CFS string
}

func (d dummyTestRepoCmd) Execute(uow *db.UnitOfWork, now time.Time) commands.CommandResult {
	return commands.CommandResult{}
}

func init() {
	RegisterRepoCommand("dummyTestRepoCmd", func() commands.Command {
		return &dummyTestRepoCmd{}
	})
}

func TestQueryCommand_RepositoryCommand_EncodeDecode(t *testing.T) {
	q := Query_Command{
		Now: time.Now().UnixNano(),
		Command: &Repository_Command{
			CMD: dummyTestRepoCmd{CFS: "test-cfs"},
		},
	}

	var buf bytes.Buffer
	if err := q.EncodeTo(&buf); err != nil {
		t.Fatalf("EncodeTo failed: %v", err)
	}

	var decoded Query_Command
	if err := decoded.DecodeFrom(buf.Bytes()); err != nil {
		t.Fatalf("DecodeFrom failed: %v", err)
	}

	if decoded.Now != q.Now {
		t.Errorf("Now mismatch: got %d, want %d", decoded.Now, q.Now)
	}

	repoCmd, ok := decoded.Command.(Repository_Command)
	if !ok {
		t.Fatalf("Expected Repository_Command, got %T", decoded.Command)
	}

	dummyPtr, ok := repoCmd.CMD.(*dummyTestRepoCmd)
	if !ok {
		dummyVal, okVal := repoCmd.CMD.(dummyTestRepoCmd)
		if !okVal {
			t.Fatalf("Expected dummyTestRepoCmd, got %T", repoCmd.CMD)
		}
		dummyPtr = &dummyVal
	}

	if dummyPtr.CFS != "test-cfs" {
		t.Errorf("CFS mismatch: got %s, want test-cfs", dummyPtr.CFS)
	}
}

func TestFSMCommand_RepositoryCommand_EncodeDecode(t *testing.T) {
	cmd := FSM_Command{
		Now:  time.Now().UnixNano(),
		Type: REPOSITORY_COMMAND,
		CMD:  dummyTestRepoCmd{CFS: "test-fsm-repo"},
	}

	var buf bytes.Buffer
	if err := cmd.EncodeTo(&buf); err != nil {
		t.Fatalf("EncodeTo failed: %v", err)
	}

	var decoded FSM_Command
	if err := decoded.DecodeFrom(buf.Bytes()); err != nil {
		t.Fatalf("DecodeFrom failed: %v", err)
	}

	if decoded.Now != cmd.Now {
		t.Errorf("Now mismatch: got %d, want %d", decoded.Now, cmd.Now)
	}

	if decoded.Type != REPOSITORY_COMMAND {
		t.Errorf("Type mismatch: got %v, want REPOSITORY_COMMAND", decoded.Type)
	}

	dummyPtr, ok := decoded.CMD.(*dummyTestRepoCmd)
	if !ok {
		dummyVal, okVal := decoded.CMD.(dummyTestRepoCmd)
		if !okVal {
			t.Fatalf("Expected dummyTestRepoCmd, got %T", decoded.CMD)
		}
		dummyPtr = &dummyVal
	}

	if dummyPtr.CFS != "test-fsm-repo" {
		t.Errorf("CFS mismatch: got %s, want test-fsm-repo", dummyPtr.CFS)
	}
}
