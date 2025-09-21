package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"fmt"

	"deadalus-orch/server/internal/pkg/config"
	binding_command "deadalus-orch/server/internal/usecase/command/binding"
	"deadalus-orch/shared/models"
	"errors"
	"strings"

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

func (bo *BindingBO) CreateBinding(ctx context.Context, code, queueCode, exchangeCode, targetExchangeCode, alternateExchangeCode, vnamespace, routingKey, pattern string, xMatch models.XMatchType, bindingType models.BindingType, targetExchangeType models.TargetExchangeType, headers map[string]string, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) (models.Binding, error) {
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
		tenantNode,
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

func (bo *BindingBO) GetBinding(ctx context.Context, exchangeCode, queueCode, vnamespace, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) (models.Binding, error) {
	findBindingCommand := &binding_command.FindBindingCommand{
		ExchangeCode: exchangeCode,
		QueueCode:    queueCode,
		VNamespace:   vnamespace,
		CF:           cf,
		CFS:          cfs,
	}

	binding, err := dragonboat.ExecuteRepositoryQuery[models.Binding](
		tenantNode,
		ctx,
		findBindingCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"find binding",
	)
	if err != nil {
		if strings.Contains(err.Error(), "entity not found") {
			return models.Binding{}, errors.New("Binding not found")
		}
		return models.Binding{}, fmt.Errorf("find binding failed: %w", err)
	}

	return binding, nil
}

func (bo *BindingBO) DeleteBinding(ctx context.Context, code, vnamespace, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) error {
	deleteBindingCommand := &binding_command.DeleteBindingCommand{
		Code:       code,
		VNamespace: vnamespace,
		CF:         cf,
		CFS:        cfs,
	}

	_, err := dragonboat.ExecuteRepositoryCommand[interface{}](
		tenantNode,
		ctx,
		deleteBindingCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"delete binding",
	)
	return err
}

func (bo *BindingBO) GetBindings(ctx context.Context, q string, cursor string, pageSize int, vNamespace string, includeObjects bool, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) (db.FindResult[models.Binding], error) {
	paginateBindingsCommand := &binding_command.PaginateBindingsCommand{
		Query:          q,
		Cursor:         cursor,
		PageSize:       pageSize,
		VNamespace:     vNamespace,
		IncludeObjects: includeObjects,
		CF:             cf,
		CFS:            cfs,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.Binding]](
		tenantNode,
		ctx,
		paginateBindingsCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"paginate bindings",
	)
	if err != nil {
		return db.FindResult[models.Binding]{}, fmt.Errorf("paginate bindings failed: %w", err)
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.Binding{}
	}

	return findResult, nil
}
