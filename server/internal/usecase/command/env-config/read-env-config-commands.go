package env_config

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
)

func init() {
	gob.Register(GetEnvGroupCommand{})
	gob.Register(ListEnvGroupsCommand{})
	gob.Register(GetEnvVarsCommand{})
	gob.Register(db.FindResult[models.EnvGroup]{})
	gob.Register(models.EnvGroup{})
	gob.Register([]models.EnvVar{})
}

type GetEnvGroupCommand struct {
	GroupID string
	CF      string
	CFS     string
}

func (cmd *GetEnvGroupCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.GroupID == "" {
		commandResult.Error = "GroupID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewEnvGroupRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	group, err := repo.GetEnvGroupByID(cmd.GroupID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to get env group: %v", err)
		return *commandResult
	}

	if group != nil {
		commandResult.Result = *group
	} else {
		commandResult.Result = nil
	}
	return *commandResult
}

type ListEnvGroupsCommand struct {
	Scope    string
	TenantID string
	PageSize int
	Cursor   string
	CF       string
	CFS      string
}

func (cmd *ListEnvGroupsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewEnvGroupRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	res, err := repo.ListEnvGroups(cmd.Scope, cmd.TenantID, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to list env groups: %v", err)
		return *commandResult
	}

	if res != nil {
		commandResult.Result = *res
	} else {
		commandResult.Result = db.FindResult[models.EnvGroup]{Entities: []models.EnvGroup{}}
	}
	return *commandResult
}

type GetEnvVarsCommand struct {
	GroupID string
	CF      string
	CFS     string
}

func (cmd *GetEnvVarsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.GroupID == "" {
		commandResult.Error = "GroupID is required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	repo, err := db.NewEnvVarRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	vars, err := repo.GetEnvVarsByGroupID(cmd.GroupID, now)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to get env vars: %v", err)
		return *commandResult
	}

	if vars != nil {
		commandResult.Result = vars
	} else {
		commandResult.Result = []models.EnvVar{}
	}
	return *commandResult
}
