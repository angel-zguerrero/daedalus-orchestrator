package business_logic

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"fmt"

	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	exchange_command "deadalus-orch/server/internal/usecase/command/exchange"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ExchangeBO struct {
	Config *common.ServerConfing
}

func NewExchangeBO(Config *common.ServerConfing) *ExchangeBO {
	return &ExchangeBO{
		Config: Config,
	}
}

func (bo *ExchangeBO) AssertExchange(ctx context.Context, tenantID, name string, exchangeType models.ExchangeType) (models.Exchange, error) {
	exchange := &models.Exchange{
		ID:       strings.ReplaceAll(uuid.New().String(), "-", ""),
		Name:     name,
		Type:     exchangeType,
		TenantID: tenantID,
	}

	createdList, err := bo.AssertExchanges(ctx, []*models.Exchange{exchange})
	if err != nil {
		return models.Exchange{}, err
	}
	return createdList[0], nil
}

func (bo *ExchangeBO) AssertExchanges(ctx context.Context, exchanges []*models.Exchange) ([]models.Exchange, error) {
	if len(exchanges) == 0 {
		return nil, errors.New("no exchanges provided")
	}

	// Asegurar IDs válidos
	for _, t := range exchanges {
		if t.ID == "" {
			t.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
		}
	}

	asseertExchangeCommand := &exchange_command.AssertExchangeCommand{
		Exchanges: make([]models.Exchange, len(exchanges)),
	}
	for i, t := range exchanges {
		asseertExchangeCommand.Exchanges[i] = *t
	}

	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout*time.Duration(len(exchanges)))
	defer writeCancel()

	fsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  asseertExchangeCommand,
	}

	result, err := bo.Config.MasterNode.Write(writeCtx, fsmCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Failed to assert exchanges (bulk)")
		return nil, fmt.Errorf("failed to assert exchanges (bulk): %w", err)
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Bulk exchange creation command returned unexpected result type")
		return nil, fmt.Errorf("bulk exchange creation command returned decode error: %w", err)
	}

	if parsedResult.Error != "" {
		return nil, fmt.Errorf("bulk exchange creation failed: %s", parsedResult.Error)
	}

	created := parsedResult.Result.([]models.Exchange)

	return created, nil
}

/* func (bo *ExchangeBO) GetExchange(ctx context.Context, exchangeID string) (models.Exchange, *dragonboat.RaftNode, *db4.NodeHostInfo, error) {
	findExchangeCommand := &exchange_command.FindExchangeCommand{
		ExchangeID: exchangeID,
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: findExchangeCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.MasterNode.Read(readCtx, *queryCommand)
	if err != nil {
		if strings.Contains(err.Error(), "cannot encode nil pointer of type") {
			return models.Exchange{}, nil, nil, errors.New("Exchange not found")
		}
		bo.Config.Logger.Error().Err(err).Msg("Find exchanges command failed")
		return models.Exchange{}, nil, nil, errors.New("Find exchanges command failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Find exchanges command failed")
		return models.Exchange{}, nil, nil, errors.New("Find exchanges command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find exchanges command failed")
		return models.Exchange{}, nil, nil, errors.New("Find exchanges command failed")
	}

	if parsedResult.Result == nil {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find exchanges command failed")
		return models.Exchange{}, nil, nil, errors.New("Exchange not found")
	}

	exchange := parsedResult.Result.(models.Exchange)
	node := bo.Config.ExchangeNodesDictionary[exchange.ID]

	if node == nil {
		return exchange, nil, nil, nil
	}

	nodeHostInfoOption := &db4.NodeHostInfoOption{SkipLogInfo: true}
	nodeHostInfo := node.NH.GetNodeHostInfo(*nodeHostInfoOption)
	return exchange, node, nodeHostInfo, nil
}

func (bo *ExchangeBO) DeleteExchange(ctx context.Context, exchangeID string) error {
	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer writeCancel()

	markToDeletionExchangeCommand := &exchange_command.MarkToDeletionExchangeCommand{
		ExchangeId: exchangeID,
	}

	atstCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  markToDeletionExchangeCommand,
	}

	result, err := bo.Config.MasterNode.Write(writeCtx, atstCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Str("ExchangeID", exchangeID).Msg("Failed to delete exchange")
		return errors.New("Failed to delete exchange: " + err.Error())
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Str("ExchangeID", exchangeID).Msg("Exchange deletion command returned unexpected result type")
		return errors.New("Exchange deletion command returned unexpected error")
	}

	if parsedResult.Error != "" {
		return errors.New("Failed to delete exchange error: " + parsedResult.Error)
	}

	deleteColumnFamilyCommand := &general_command.DeleteColumnFamilyCommand{
		Name: exchangeID,
	}

	ccfCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  deleteColumnFamilyCommand,
	}

	exchangeNode := bo.Config.ExchangeNodesDictionary[exchangeID]
	if exchangeNode == nil {
		return errors.New("Failed to delete exchange error: Exchange node not found")
	}

	_, err = exchangeNode.Write(writeCtx, ccfCmd)
	if err != nil {
		return errors.New("Failed to delete exchange error: " + err.Error())
	}

	deleteExchangeCommand := &exchange_command.DeleteExchangeCommand{
		ExchangeId: exchangeID,
	}

	atstCmd = general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  deleteExchangeCommand,
	}

	_, err = bo.Config.MasterNode.Write(writeCtx, atstCmd)
	if err != nil {
		return errors.New("Failed to delete exchange error: " + err.Error())
	}

	bo.Config.Logger.Info().Str("ExchangeID", exchangeID).Msg("new exchange deleted successfully")
	return nil
}

func (bo *ExchangeBO) GetExchanges(ctx context.Context, q string, cursor string, pageSize int) (db.FindResult[models.Exchange], error) {
	paginateExchangesCommand := &exchange_command.PaginateExchangesCommand{
		Cursor:   cursor,
		PageSize: pageSize,
		Q:        q,
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: paginateExchangesCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.MasterNode.Read(readCtx, *queryCommand)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate exchanges command failed")
		return db.FindResult[models.Exchange]{}, errors.New("Login failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate exchanges command failed")
		return db.FindResult[models.Exchange]{}, errors.New("Paginate exchanges command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Paginate exchanges command failed")
		return db.FindResult[models.Exchange]{}, errors.New("Paginate exchanges command failed")
	}

	findResult := parsedResult.Result.(db.FindResult[models.Exchange])
	if findResult.Entities == nil {
		findResult.Entities = []models.Exchange{}
	}

	return findResult, nil
}
*/
