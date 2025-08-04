package business_logic

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	vnamespace_command "deadalus-orch/server/internal/usecase/command/vnamespace"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"errors"
	"time"
)

type VNamespaceBO struct {
	Config *common.ServerConfing
}

func NewVNamespaceBO(config *common.ServerConfing) *VNamespaceBO {
	return &VNamespaceBO{
		Config: config,
	}
}

func (bo *VNamespaceBO) GetVNamespaces(ctx context.Context, q string, cursor string, pageSize int, cf, cfs string) (db.FindResult[models.VNamespace], error) {
	paginateVNamespacesCommand := &vnamespace_command.PaginateVNamespacesCommand{
		Query:    q,
		Cursor:   cursor,
		PageSize: pageSize,
		CF:       cf,
		CFS:      cfs,
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: paginateVNamespacesCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.TenantNodesDictionary[cfs].Read(readCtx, *queryCommand)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate vnamespaces command failed")
		return db.FindResult[models.VNamespace]{}, errors.New("Paginate vnamespaces failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate vnamespaces command failed")
		return db.FindResult[models.VNamespace]{}, errors.New("Paginate vnamespaces command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Paginate vnamespaces command failed")
		return db.FindResult[models.VNamespace]{}, errors.New("Paginate vnamespaces command failed")
	}

	findResult := parsedResult.Result.(db.FindResult[models.VNamespace])
	if findResult.Entities == nil {
		findResult.Entities = []models.VNamespace{}
	}

	return findResult, nil
}
