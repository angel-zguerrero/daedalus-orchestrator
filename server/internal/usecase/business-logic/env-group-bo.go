package business_logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/crypto"
	env_config_command "deadalus-orch/server/internal/usecase/command/env-config"
	"deadalus-orch/shared/models"

	"github.com/google/uuid"
)

func toKebabCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var result strings.Builder
	var prevIsDash bool
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
			prevIsDash = false
		} else if r >= 'A' && r <= 'Z' {
			if result.Len() > 0 && !prevIsDash {
				result.WriteByte('-')
			}
			result.WriteRune(r + ('a' - 'A'))
			prevIsDash = false
		} else if r == ' ' || r == '-' || r == '_' || r == '.' || r == '/' || r == '\\' {
			if result.Len() > 0 && !prevIsDash {
				result.WriteByte('-')
				prevIsDash = true
			}
		}
	}
	return strings.Trim(result.String(), "-")
}

type EnvGroupBO struct {
	Config *common.ServerConfing
}

func NewEnvGroupBO(config *common.ServerConfing) *EnvGroupBO {
	return &EnvGroupBO{
		Config: config,
	}
}

func (bo *EnvGroupBO) resolveRaftNode(scope models.EnvGroupScope, tenantNode *dragonboat.RaftNode) (*dragonboat.RaftNode, string, string, error) {
	if scope == models.EnvGroupScopeGlobal {
		return bo.Config.MasterNode, db.AdminFC, db.AdminFCSector, nil
	}
	if tenantNode == nil {
		return nil, "", "", errors.New("tenant node is required for tenant scope")
	}
	return tenantNode, "", "", nil
}

func (bo *EnvGroupBO) CreateGroup(
	ctx context.Context,
	scope models.EnvGroupScope,
	tenantID string,
	code string,
	name string,
	description string,
	typeStr models.EnvGroupType,
	vnamespace string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (models.EnvGroup, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)

	if code == "" && name == "" {
		return models.EnvGroup{}, errors.New("group name or code is required")
	}

	if code == "" {
		code = toKebabCase(name)
	}
	if name == "" {
		name = code
	}
	if typeStr != models.EnvGroupTypeConfig && typeStr != models.EnvGroupTypeSecret {
		typeStr = models.EnvGroupTypeConfig
	}
	if vnamespace == "" {
		vnamespace = "default"
	}

	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return models.EnvGroup{}, err
	}
	if scope == models.EnvGroupScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	// Generating ID outside command as required!
	groupID := uuid.New().String()

	group := models.EnvGroup{
		ID:          groupID,
		Code:        code,
		Name:        name,
		Description: description,
		Type:        typeStr,
		Scope:       scope,
		TenantID:    tenantID,
		VNamespace:  vnamespace,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	cmd := &env_config_command.CreateEnvGroupCommand{
		EnvGroup: group,
		CF:       targetCF,
		CFS:      targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	created, err := dragonboat.ExecuteRepositoryCommand[models.EnvGroup](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"create env group",
	)
	if err != nil {
		return models.EnvGroup{}, err
	}

	return created, nil
}

func (bo *EnvGroupBO) UpdateGroup(
	ctx context.Context,
	scope models.EnvGroupScope,
	id string,
	name string,
	description string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (models.EnvGroup, error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return models.EnvGroup{}, err
	}
	if scope == models.EnvGroupScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	group := models.EnvGroup{
		ID:          id,
		Name:        name,
		Description: description,
	}

	cmd := &env_config_command.UpdateEnvGroupCommand{
		EnvGroup: group,
		CF:       targetCF,
		CFS:      targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	updated, err := dragonboat.ExecuteRepositoryCommand[models.EnvGroup](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"update env group",
	)
	if err != nil {
		return models.EnvGroup{}, err
	}

	return updated, nil
}

func (bo *EnvGroupBO) DeleteGroup(
	ctx context.Context,
	scope models.EnvGroupScope,
	groupID string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) error {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return err
	}
	if scope == models.EnvGroupScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &env_config_command.DeleteEnvGroupCommand{
		GroupID: groupID,
		CF:      targetCF,
		CFS:     targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	_, err = dragonboat.ExecuteRepositoryCommand[bool](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"delete env group",
	)
	return err
}

func (bo *EnvGroupBO) GetGroup(
	ctx context.Context,
	scope models.EnvGroupScope,
	groupID string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (*models.EnvGroup, error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return nil, err
	}
	if scope == models.EnvGroupScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &env_config_command.GetEnvGroupCommand{
		GroupID: groupID,
		CF:      targetCF,
		CFS:     targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	group, err := dragonboat.ExecuteRepositoryQuery[models.EnvGroup](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"get env group",
	)
	if err != nil {
		return nil, err
	}
	if group.ID == "" {
		return nil, nil
	}
	return &group, nil
}

func (bo *EnvGroupBO) ListGroups(
	ctx context.Context,
	scope models.EnvGroupScope,
	tenantID string,
	pageSize int,
	cursor string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (*db.FindResult[models.EnvGroup], error) {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return nil, err
	}
	if scope == models.EnvGroupScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &env_config_command.ListEnvGroupsCommand{
		Scope:    string(scope),
		TenantID: tenantID,
		PageSize: pageSize,
		Cursor:   cursor,
		CF:       targetCF,
		CFS:      targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	res, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.EnvGroup]](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"list env groups",
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (bo *EnvGroupBO) GetGroupVars(
	ctx context.Context,
	scope models.EnvGroupScope,
	groupID string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) ([]models.EnvVar, error) {
	group, err := bo.GetGroup(ctx, scope, groupID, cf, cfs, tenantNode)
	if err != nil || group == nil {
		return nil, fmt.Errorf("group not found: %v", err)
	}

	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return nil, err
	}
	if scope == models.EnvGroupScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &env_config_command.GetEnvVarsCommand{
		GroupID: groupID,
		CF:      targetCF,
		CFS:     targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	vars, err := dragonboat.ExecuteRepositoryQuery[[]models.EnvVar](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"get env vars",
	)
	if err != nil {
		return nil, err
	}

	// Decrypt values if group is a secret!
	if group.Type == models.EnvGroupTypeSecret {
		for i := range vars {
			decrypted, err := crypto.Decrypt(vars[i].Value)
			if err == nil {
				vars[i].Value = decrypted
			}
		}
	}

	return vars, nil
}

func (bo *EnvGroupBO) SaveGroupVar(
	ctx context.Context,
	scope models.EnvGroupScope,
	groupID string,
	varID string,
	key string,
	value string,
	description string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (models.EnvVar, error) {
	group, err := bo.GetGroup(ctx, scope, groupID, cf, cfs, tenantNode)
	if err != nil || group == nil {
		return models.EnvVar{}, errors.New("group not found")
	}

	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return models.EnvVar{}, err
	}
	if scope == models.EnvGroupScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	// Encrypt if group is secret!
	valToSave := value
	if group.Type == models.EnvGroupTypeSecret && value != "" {
		encrypted, err := crypto.Encrypt(value)
		if err != nil {
			return models.EnvVar{}, fmt.Errorf("failed to encrypt secret value: %w", err)
		}
		valToSave = encrypted
	}

	// Generating ID outside command as required!
	if varID == "" {
		varID = uuid.New().String()
	}

	envVar := models.EnvVar{
		ID:          varID,
		GroupID:     groupID,
		Key:         key,
		Value:       valToSave,
		Description: description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	cmd := &env_config_command.SaveEnvVarCommand{
		EnvVar: envVar,
		CF:     targetCF,
		CFS:    targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	saved, err := dragonboat.ExecuteRepositoryCommand[models.EnvVar](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"save env var",
	)
	if err != nil {
		return models.EnvVar{}, err
	}

	// Return plaintext value in response
	saved.Value = value
	return saved, nil
}

func (bo *EnvGroupBO) DeleteGroupVar(
	ctx context.Context,
	scope models.EnvGroupScope,
	varID string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) error {
	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return err
	}
	if scope == models.EnvGroupScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	cmd := &env_config_command.DeleteEnvVarCommand{
		VarID: varID,
		CF:    targetCF,
		CFS:   targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	_, err = dragonboat.ExecuteRepositoryCommand[bool](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"delete env var",
	)
	return err
}

func (bo *EnvGroupBO) BulkSaveGroupVars(
	ctx context.Context,
	scope models.EnvGroupScope,
	groupID string,
	inputVars []models.EnvVar,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) ([]models.EnvVar, error) {
	group, err := bo.GetGroup(ctx, scope, groupID, cf, cfs, tenantNode)
	if err != nil || group == nil {
		return nil, errors.New("group not found")
	}

	node, targetCF, targetCFS, err := bo.resolveRaftNode(scope, tenantNode)
	if err != nil {
		return nil, err
	}
	if scope == models.EnvGroupScopeTenant {
		targetCF = cf
		targetCFS = cfs
	}

	varsToSave := make([]models.EnvVar, 0, len(inputVars))
	for _, v := range inputVars {
		if v.Key == "" {
			continue
		}
		valToSave := v.Value
		if group.Type == models.EnvGroupTypeSecret && v.Value != "" {
			enc, err := crypto.Encrypt(v.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt key %s: %w", v.Key, err)
			}
			valToSave = enc
		}

		// Generating ID outside command as required!
		varID := v.ID
		if varID == "" {
			varID = uuid.New().String()
		}

		varsToSave = append(varsToSave, models.EnvVar{
			ID:          varID,
			GroupID:     groupID,
			Key:         v.Key,
			Value:       valToSave,
			Description: v.Description,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		})
	}

	cmd := &env_config_command.BulkSaveEnvVarsCommand{
		GroupID: groupID,
		EnvVars: varsToSave,
		CF:      targetCF,
		CFS:     targetCFS,
	}

	timeout := config.GlobalConfiguration.ApiRaftTimeout
	saved, err := dragonboat.ExecuteRepositoryCommand[[]models.EnvVar](
		node,
		ctx,
		cmd,
		timeout,
		bo.Config.Logger,
		"bulk save env vars",
	)
	if err != nil {
		return nil, err
	}

	// Return plaintext values for user response
	for i := range saved {
		for _, orig := range inputVars {
			if orig.Key == saved[i].Key {
				saved[i].Value = orig.Value
				break
			}
		}
	}

	return saved, nil
}
