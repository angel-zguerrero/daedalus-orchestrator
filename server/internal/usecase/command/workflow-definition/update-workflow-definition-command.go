package workflow_definition

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/bpmn"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/server/internal/usecase/command/workflow-execution"
	models "deadalus-orch/shared/models"
)

func init() {
	gob.Register(UpdateWorkflowDefinitionCommand{})
}

type UpdateWorkflowDefinitionCommand struct {
	WorkflowDefinition models.WorkflowDefinition
	CF                 string
	CFS                string
}

func (cmd *UpdateWorkflowDefinitionCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.WorkflowDefinition.ID == "" {
		commandResult.Error = "ID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewWorkflowDefinitionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	existing, err := repo.GetWorkflowDefinitionByID(cmd.WorkflowDefinition.ID, now)
	if err != nil || existing == nil {
		commandResult.Error = "workflow definition not found"
		return *commandResult
	}

	existing.Name = cmd.WorkflowDefinition.Name
	existing.Description = cmd.WorkflowDefinition.Description
	if cmd.WorkflowDefinition.OnVersionChange != "" {
		existing.OnVersionChange = cmd.WorkflowDefinition.OnVersionChange
	}
	if existing.OnVersionChange == "" {
		existing.OnVersionChange = models.VersionChangePolicyDefinedInExecution
	}

	isStructuralChange := false
	var newStructuralHash string
	oldPayload := existing.Payload
	oldVerNum := existing.Version

	if len(cmd.WorkflowDefinition.Payload) > 0 {
		oldHash, _ := bpmn.ComputeStructuralHash(existing.Payload)
		newHash, _ := bpmn.ComputeStructuralHash(cmd.WorkflowDefinition.Payload)

		if oldHash == "" || oldHash != newHash {
			isStructuralChange = true
			newStructuralHash = newHash
			existing.Version++
		}
		existing.Payload = cmd.WorkflowDefinition.Payload
	}

	if cmd.WorkflowDefinition.PayloadFormat != "" {
		existing.PayloadFormat = cmd.WorkflowDefinition.PayloadFormat
	}
	existing.MaxDurationSeconds = cmd.WorkflowDefinition.MaxDurationSeconds
	existing.IsActive = cmd.WorkflowDefinition.IsActive
	existing.HasDesignErrors = cmd.WorkflowDefinition.HasDesignErrors
	existing.DesignErrorMessages = cmd.WorkflowDefinition.DesignErrorMessages
	if cmd.WorkflowDefinition.VNamespace != "" {
		existing.VNamespace = cmd.WorkflowDefinition.VNamespace
	}

	ok, err := repo.UpdateWorkflowDefinition(existing, now)
	if err != nil || !ok {
		commandResult.Error = fmt.Sprintf("failed to update workflow definition: %v", err)
		return *commandResult
	}

	if isStructuralChange {
		versionRepo, err := db.NewWorkflowDefinitionVersionRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if err == nil {
			if oldVerNum > 0 {
				prevVer, _ := versionRepo.GetVersionByNumber(existing.ID, oldVerNum, now)
				if prevVer == nil && len(oldPayload) > 0 {
					oldHash, _ := bpmn.ComputeStructuralHash(oldPayload)
					createdAt := existing.UpdatedAt
					if createdAt.IsZero() {
						createdAt = existing.CreatedAt
					}
					if createdAt.IsZero() {
						createdAt = now
					}
					prevRecord := &models.WorkflowDefinitionVersion{
						ID:                   fmt.Sprintf("%s-v%d", existing.ID, oldVerNum),
						WorkflowDefinitionID: existing.ID,
						Version:              oldVerNum,
						VNamespace:           existing.VNamespace,
						Payload:              oldPayload,
						PayloadFormat:        existing.PayloadFormat,
						StructuralHash:       oldHash,
						CreatedAt:            createdAt,
					}
					_, _ = versionRepo.CreateVersion(prevRecord, now)
				}
			}

			versionRecord := &models.WorkflowDefinitionVersion{
				ID:                   fmt.Sprintf("%s-v%d", existing.ID, existing.Version),
				WorkflowDefinitionID: existing.ID,
				Version:              existing.Version,
				VNamespace:           existing.VNamespace,
				Payload:              existing.Payload,
				PayloadFormat:        existing.PayloadFormat,
				StructuralHash:       newStructuralHash,
				CreatedAt:            now,
			}
			_, _ = versionRepo.CreateVersion(versionRecord, now)
		}

		execRepo, err := db.NewWorkflowExecutionRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if err == nil {
			execList, err := execRepo.ListWorkflowExecutions(existing.VNamespace, existing.ID, "", 1000, "", now)
			if err == nil && execList != nil {
				for _, exec := range execList.Entities {
					if exec.Status == models.WorkflowExecutionStatusRunning || exec.Status == models.WorkflowExecutionStatusPending {
						effectivePolicy := existing.OnVersionChange
						if effectivePolicy == "" || effectivePolicy == models.VersionChangePolicyDefinedInExecution {
							effectivePolicy = exec.OnVersionChange
						}

						if effectivePolicy == models.VersionChangePolicyRestart {
							exec.Status = models.WorkflowExecutionStatusTerminated
							exec.Error = "Terminated due to workflow version upgrade (restarted)"
							exec.UpdatedAt = now
							_, _ = execRepo.UpdateWorkflowExecution(&exec, now)

							startCmd := &workflow_execution.StartWorkflowExecutionCommand{
								WorkflowDefinitionID: existing.ID,
								ExecutionKey:         exec.ExecutionKey,
								OnVersionChange:      exec.OnVersionChange,
								Input:                exec.Input,
								VNamespace:           exec.VNamespace,
								CF:                   cmd.CF,
								CFS:                  cmd.CFS,
							}
							_ = startCmd.Execute(uow, now)
						}
					}
				}
			}
		}
	}

	commandResult.Result = *existing
	return *commandResult
}
