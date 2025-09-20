package business_logic

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"

	"deadalus-orch/server/internal/pkg/config"
	commands "deadalus-orch/server/internal/usecase/command"
	binding_command "deadalus-orch/server/internal/usecase/command/binding"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type BindingBO struct {
	Config *common.ServerConfing
}

func NewBindingBO(Config *common.ServerConfing) *BindingBO {
	return &BindingBO{
		Config: Config,
	}
}

func (bo *BindingBO) CreateBinding(ctx context.Context, code, queueCode, exchangeCode, targetExchangeCode, alternateExchangeCode, vnamespace, routingKey, pattern string, xMatch models.XMatchType, bindingType models.BindingType, targetExchangeType models.TargetExchangeType, headers map[string]string, cf, cfs string) (models.Binding, error) {
	assertBindingCommand := &binding_command.AssertBindingCommand{
		NewBindingID:          strings.ReplaceAll(uuid.New().String(), "-", ""),
		Code:                  code,
		QueueCode:             queueCode,
		ExchangeCode:          exchangeCode,
		TargetExchangeCode:    targetExchangeCode,
		AlternateExchangeCode: alternateExchangeCode,
		VNamespace:            vnamespace,
		RoutingKey:            routingKey,
		Pattern:               pattern,
		XMatch:                xMatch,
		BindingType:           bindingType,
		TargetExchangeType:    targetExchangeType,
		Headers:               headers,
		CF:                    cf,
		CFS:                   cfs,
	}

	created, err := dragonboat.ExecuteRepositoryCommand[models.Binding](
		bo.Config.TenantNodesDictionary[cfs],
		ctx,
		assertBindingCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"create binding",
	)
	if err != nil {
		return models.Binding{}, err
	}

	return created, nil
}

func (bo *BindingBO) GetBinding(ctx context.Context, exchangeCode, queueCode, vnamespace, cf, cfs string) (models.Binding, error) {
	findBindingCommand := &binding_command.FindBindingCommand{
		ExchangeCode: exchangeCode,
		QueueCode:    queueCode,
		VNamespace:   vnamespace,
		CF:           cf,
		CFS:          cfs,
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: findBindingCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.TenantNodesDictionary[cfs].Read(readCtx, *queryCommand)
	if err != nil {
		if strings.Contains(err.Error(), "cannot encode nil pointer of type") {
			return models.Binding{}, errors.New("Binding not found")
		}
		bo.Config.Logger.Error().Err(err).Msg("Find binding command failed")
		return models.Binding{}, errors.New("Find binding command failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Find binding command failed")
		return models.Binding{}, errors.New("Find binding command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find binding command failed")
		return models.Binding{}, errors.New("Find binding command failed")
	}

	if parsedResult.Result == nil {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find binding command failed")
		return models.Binding{}, errors.New("Binding not found")
	}

	binding := parsedResult.Result.(models.Binding)
	return binding, nil
}

func (bo *BindingBO) DeleteBinding(ctx context.Context, code, vnamespace, cf, cfs string) error {
	deleteBindingCommand := &binding_command.DeleteBindingCommand{
		Code:       code,
		VNamespace: vnamespace,
		CF:         cf,
		CFS:        cfs,
	}

	_, err := dragonboat.ExecuteRepositoryCommand[interface{}](
		bo.Config.TenantNodesDictionary[cfs],
		ctx,
		deleteBindingCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"delete binding",
	)
	return err
}

func (bo *BindingBO) GetBindings(ctx context.Context, q string, cursor string, pageSize int, vNamespace string, includeObjects bool, cf, cfs string) (db.FindResult[models.Binding], error) {
	paginateBindingsCommand := &binding_command.PaginateBindingsCommand{
		Query:          q,
		Cursor:         cursor,
		PageSize:       pageSize,
		VNamespace:     vNamespace,
		IncludeObjects: includeObjects,
		CF:             cf,
		CFS:            cfs,
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: paginateBindingsCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.TenantNodesDictionary[cfs].Read(readCtx, *queryCommand)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate bindings command failed")
		return db.FindResult[models.Binding]{}, errors.New("Paginate bindings failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate bindings command failed")
		return db.FindResult[models.Binding]{}, errors.New("Paginate bindings command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Paginate bindings command failed")
		return db.FindResult[models.Binding]{}, errors.New("Paginate bindings command failed")
	}

	findResult := parsedResult.Result.(db.FindResult[models.Binding])
	if findResult.Entities == nil {
		findResult.Entities = []models.Binding{}
	}

	return findResult, nil
}
