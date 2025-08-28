package business_logic

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"fmt"

	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	binding_command "deadalus-orch/server/internal/usecase/command/binding"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"errors"
	"strings"
	"time"
)

type BindingBO struct {
	Config *common.ServerConfing
}

func NewBindingBO(Config *common.ServerConfing) *BindingBO {
	return &BindingBO{
		Config: Config,
	}
}

func (bo *BindingBO) CreateBinding(ctx context.Context, queueCode, exchangeCode, vnamespace, routingKey, pattern string, xMatch models.XMatchType, bindingType models.BindingType, cf, cfs string) (models.Binding, error) {
	assertBindingCommand := &binding_command.AssertBindingCommand{
		QueueCode:    queueCode,
		ExchangeCode: exchangeCode,
		VNamespace:   vnamespace,
		RoutingKey:   routingKey,
		Pattern:      pattern,
		XMatch:       xMatch,
		BindingType:  bindingType,
		CF:           cf,
		CFS:          cfs,
	}

	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer writeCancel()

	fsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  assertBindingCommand,
	}

	result, err := bo.Config.TenantNodesDictionary[cfs].Write(writeCtx, fsmCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Failed to create binding")
		return models.Binding{}, fmt.Errorf("failed to create binding: %w", err)
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Binding creation command returned unexpected result type")
		return models.Binding{}, fmt.Errorf("binding creation command returned decode error: %w", err)
	}

	if parsedResult.Error != "" {
		return models.Binding{}, fmt.Errorf("binding creation failed: %s", parsedResult.Error)
	}

	created := parsedResult.Result.(models.Binding)
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

func (bo *BindingBO) DeleteBinding(ctx context.Context, exchangeCode, queueCode, vnamespace, cf, cfs string) error {
	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer writeCancel()

	deleteBindingCommand := &binding_command.DeleteBindingCommand{
		ExchangeCode: exchangeCode,
		QueueCode:    queueCode,
		VNamespace:   vnamespace,
		CF:           cf,
		CFS:          cfs,
	}

	fsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  deleteBindingCommand,
	}

	result, err := bo.Config.TenantNodesDictionary[cfs].Write(writeCtx, fsmCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Str("ExchangeCode", exchangeCode).Str("QueueCode", queueCode).Str("VNamespace", vnamespace).Msg("Failed to delete binding")
		return errors.New("Failed to delete binding: " + err.Error())
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Str("ExchangeCode", exchangeCode).Str("QueueCode", queueCode).Str("VNamespace", vnamespace).Msg("Binding deletion command returned unexpected result type")
		return errors.New("Binding deletion command returned unexpected error")
	}

	if parsedResult.Error != "" {
		return errors.New("Failed to delete binding error: " + parsedResult.Error)
	}

	bo.Config.Logger.Info().Str("ExchangeCode", exchangeCode).Str("QueueCode", queueCode).Str("VNamespace", vnamespace).Msg("binding deleted successfully")
	return nil
}

func (bo *BindingBO) GetBindings(ctx context.Context, q string, cursor string, pageSize int, vNamespace string, cf, cfs string) (db.FindResult[models.Binding], error) {
	paginateBindingsCommand := &binding_command.PaginateBindingsCommand{
		Query:      q,
		Cursor:     cursor,
		PageSize:   pageSize,
		VNamespace: vNamespace,
		CF:         cf,
		CFS:        cfs,
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
