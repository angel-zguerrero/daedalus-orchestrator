package general_command

import (
	commands "deadalus-orch/server/internal/usecase/command"
	auth_command "deadalus-orch/server/internal/usecase/command/auth"
	exchange_command "deadalus-orch/server/internal/usecase/command/exchange"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	tenant_summary_command "deadalus-orch/server/internal/usecase/command/tenant-summary"
	tentant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
	"testing"
	"time"
)

func TestEncodeDecodeRepoCommand_LoginCommand(t *testing.T) {
	cmd := &auth_command.LoginCommand{
		UsernameOrEmail: "admin",
		Password:        "secret-password",
	}

	typeName, jsonBytes, err := EncodeRepoCommand(cmd)
	if err != nil {
		t.Fatalf("EncodeRepoCommand failed: %v", err)
	}

	if typeName != "LoginCommand" {
		t.Errorf("Expected typeName 'LoginCommand', got '%s'", typeName)
	}

	decodedCmd, err := DecodeRepoCommand(typeName, jsonBytes)
	if err != nil {
		t.Fatalf("DecodeRepoCommand failed: %v", err)
	}

	loginCmd, ok := decodedCmd.(*auth_command.LoginCommand)
	if !ok {
		t.Fatalf("Expected *auth_command.LoginCommand, got %T", decodedCmd)
	}

	if loginCmd.UsernameOrEmail != "admin" || loginCmd.Password != "secret-password" {
		t.Errorf("Decoded content mismatch: %+v", loginCmd)
	}
}

func TestEncodeDecodeRepoCommand_EnqueueCommand(t *testing.T) {
	cmd := &queue_command.EnqueueCommand{
		CF:  "cf-1",
		CFS: "cfs-1",
		Messages: []models.QueueMessage{
			{ID: "msg-123", QueueID: "test-queue"},
		},
	}

	typeName, jsonBytes, err := EncodeRepoCommand(cmd)
	if err != nil {
		t.Fatalf("EncodeRepoCommand failed: %v", err)
	}

	if typeName != "EnqueueCommand" {
		t.Errorf("Expected typeName 'EnqueueCommand', got '%s'", typeName)
	}

	decodedCmd, err := DecodeRepoCommand(typeName, jsonBytes)
	if err != nil {
		t.Fatalf("DecodeRepoCommand failed: %v", err)
	}

	enqueueCmd, ok := decodedCmd.(*queue_command.EnqueueCommand)
	if !ok {
		t.Fatalf("Expected *queue_command.EnqueueCommand, got %T", decodedCmd)
	}

	if enqueueCmd.CF != "cf-1" || len(enqueueCmd.Messages) != 1 {
		t.Errorf("Decoded content mismatch: %+v", enqueueCmd)
	}
}

func TestEncodeDecodeRepoCommand_GetOutboxEvents(t *testing.T) {
	cmd := &tentant_command.GetOutboxEventsCommand{
		CFS: "tenant-cfs-1",
	}

	typeName, jsonBytes, err := EncodeRepoCommand(cmd)
	if err != nil {
		t.Fatalf("EncodeRepoCommand failed: %v", err)
	}

	if typeName != "GetOutboxEventsCommand" {
		t.Errorf("Expected typeName 'GetOutboxEventsCommand', got '%s'", typeName)
	}

	decodedCmd, err := DecodeRepoCommand(typeName, jsonBytes)
	if err != nil {
		t.Fatalf("DecodeRepoCommand failed: %v", err)
	}

	outboxCmd, ok := decodedCmd.(*tentant_command.GetOutboxEventsCommand)
	if !ok {
		t.Fatalf("Expected *tentant_command.GetOutboxEventsCommand, got %T", decodedCmd)
	}

	if outboxCmd.CFS != "tenant-cfs-1" {
		t.Errorf("Decoded content mismatch: %+v", outboxCmd)
	}
}

func TestEncodeDecodeRepoCommand_DeleteExchange(t *testing.T) {
	cmd := &exchange_command.DeleteExchangeCommand{
		Code:       "ex-1",
		VNamespace: "vns-1",
	}

	typeName, jsonBytes, err := EncodeRepoCommand(cmd)
	if err != nil {
		t.Fatalf("EncodeRepoCommand failed: %v", err)
	}

	if typeName != "DeleteExchangeCommand" {
		t.Errorf("Expected typeName 'DeleteExchangeCommand', got '%s'", typeName)
	}

	decodedCmd, err := DecodeRepoCommand(typeName, jsonBytes)
	if err != nil {
		t.Fatalf("DecodeRepoCommand failed: %v", err)
	}

	delExCmd, ok := decodedCmd.(*exchange_command.DeleteExchangeCommand)
	if !ok {
		t.Fatalf("Expected *exchange_command.DeleteExchangeCommand, got %T", decodedCmd)
	}

	if delExCmd.Code != "ex-1" || delExCmd.VNamespace != "vns-1" {
		t.Errorf("Decoded content mismatch: %+v", delExCmd)
	}
}

func TestDecodeRepoCommand_UnregisteredCommand(t *testing.T) {
	_, err := DecodeRepoCommand("NonExistentCommand", []byte("{}"))
	if err == nil {
		t.Fatal("Expected error for unregistered command, got nil")
	}
}

func TestEncodeDecodeCommandResult_SuccessAndError(t *testing.T) {
	// Test Struct Payload
	resultPayload := queue_command.AckMessageResult{
		Success: true,
	}

	res := commands.CommandResult{
		Result: resultPayload,
	}

	encoded, err := commands.EncodeCommandResult(res)
	if err != nil {
		t.Fatalf("EncodeCommandResult failed: %v", err)
	}

	decoded, err := commands.DecodeCommandResult[queue_command.AckMessageResult](encoded)
	if err != nil {
		t.Fatalf("DecodeCommandResult failed: %v", err)
	}

	if !decoded.Success {
		t.Errorf("Expected Success=true, got false")
	}

	// Test Error Payload
	errRes := commands.CommandResult{
		Error: "queue not found",
	}

	errEncoded, err := commands.EncodeCommandResult(errRes)
	if err != nil {
		t.Fatalf("EncodeCommandResult failed for error: %v", err)
	}

	_, errDec := commands.DecodeCommandResult[queue_command.AckMessageResult](errEncoded)
	if errDec == nil || errDec.Error() != "queue not found" {
		t.Errorf("Expected 'queue not found' error, got %v", errDec)
	}
}

func TestEncodeDecodeRepoCommand_WrappedInRepositoryCommand(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	cmd := Repository_Command{
		CMD: &tenant_summary_command.RefreshLastUpdateAtFromCommand{
			LastUpdateAtFrom: now,
		},
	}

	typeName, jsonBytes, err := EncodeRepoCommand(cmd)
	if err != nil {
		t.Fatalf("EncodeRepoCommand failed for wrapped command: %v", err)
	}

	if typeName != "RefreshLastUpdateAtFromCommand" {
		t.Errorf("Expected typeName 'RefreshLastUpdateAtFromCommand', got '%s'", typeName)
	}

	decodedCmd, err := DecodeRepoCommand(typeName, jsonBytes)
	if err != nil {
		t.Fatalf("DecodeRepoCommand failed: %v", err)
	}

	refreshCmd, ok := decodedCmd.(*tenant_summary_command.RefreshLastUpdateAtFromCommand)
	if !ok {
		t.Fatalf("Expected *tenant_summary_command.RefreshLastUpdateAtFromCommand, got %T", decodedCmd)
	}

	if !refreshCmd.LastUpdateAtFrom.Equal(now) {
		t.Errorf("Decoded content mismatch: got %v, want %v", refreshCmd.LastUpdateAtFrom, now)
	}
}

func TestEncodeDecodeRepoCommand_UpdateTenantSummaryInMasterCommand(t *testing.T) {
	cmd := &tentant_command.UpdateTenantSummaryInMasterCommand{
		TenantSummaries: []models.TenantSummary{
			{
				ID:             "tenant-123",
				ExchangesCount: 5,
				QueuesCount:    10,
				MessagesCount:  42,
			},
		},
	}

	typeName, jsonBytes, err := EncodeRepoCommand(cmd)
	if err != nil {
		t.Fatalf("EncodeRepoCommand failed: %v", err)
	}

	if typeName != "UpdateTenantSummaryInMasterCommand" {
		t.Errorf("Expected typeName 'UpdateTenantSummaryInMasterCommand', got '%s'", typeName)
	}

	decodedCmd, err := DecodeRepoCommand(typeName, jsonBytes)
	if err != nil {
		t.Fatalf("DecodeRepoCommand failed: %v", err)
	}

	updateCmd, ok := decodedCmd.(*tentant_command.UpdateTenantSummaryInMasterCommand)
	if !ok {
		t.Fatalf("Expected *tentant_command.UpdateTenantSummaryInMasterCommand, got %T", decodedCmd)
	}

	if len(updateCmd.TenantSummaries) != 1 || updateCmd.TenantSummaries[0].ID != "tenant-123" {
		t.Errorf("Decoded content mismatch: %+v", updateCmd)
	}
}

