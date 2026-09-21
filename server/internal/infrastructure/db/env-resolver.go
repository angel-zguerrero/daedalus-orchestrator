package db

import (
	"fmt"
	"time"

	"deadalus-orch/server/internal/pkg/crypto"
	models "deadalus-orch/shared/models"
)

type DBEnvResolver struct {
	UOW        *UnitOfWork
	TenantCF   string
	TenantCFS  string
	VNamespace string
	Now        time.Time
}

func (r *DBEnvResolver) ResolveEnvVar(groupType string, scope string, groupRef string, varKey string) (string, error) {
	if r.UOW == nil {
		return "", fmt.Errorf("UnitOfWork is required for DBEnvResolver")
	}

	targetCF := r.TenantCF
	targetCFS := r.TenantCFS
	targetVNamespace := r.VNamespace

	if scope == string(models.EnvGroupScopeGlobal) {
		targetCF = AdminFC
		targetCFS = AdminFCSector
		targetVNamespace = "default"
	}

	idFactory := &DeterministicIDGeneratorFactory{}
	groupRepo, err := NewEnvGroupRepository(r.UOW, idFactory, targetCF, targetCFS)
	if err != nil {
		return "", fmt.Errorf("failed to create env group repository: %w", err)
	}

	group, err := groupRepo.GetEnvGroupByCode(groupRef, targetVNamespace, r.Now)
	if err != nil || group == nil {
		group, err = groupRepo.GetEnvGroupByName(groupRef, targetVNamespace, r.Now)
	}

	if group == nil {
		return "", fmt.Errorf("environment group %q (scope: %s, type: %s) not found", groupRef, scope, groupType)
	}

	if string(group.Type) != groupType {
		return "", fmt.Errorf("environment group %q is of type %s, but expression requested %s", groupRef, group.Type, groupType)
	}

	if string(group.Scope) != scope {
		return "", fmt.Errorf("environment group %q has scope %s, but expression requested %s", groupRef, group.Scope, scope)
	}

	varRepo, err := NewEnvVarRepository(r.UOW, idFactory, targetCF, targetCFS)
	if err != nil {
		return "", fmt.Errorf("failed to create env var repository: %w", err)
	}

	envVar, err := varRepo.GetEnvVarByKey(group.ID, varKey, r.Now)
	if err != nil || envVar == nil || envVar.Key == "" {
		return "", fmt.Errorf("environment variable %q not found in group %q", varKey, groupRef)
	}

	val := envVar.Value
	if group.Type == models.EnvGroupTypeSecret || groupType == string(models.EnvGroupTypeSecret) {
		decrypted, err := crypto.Decrypt(val)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt secret variable %q: %w", varKey, err)
		}
		val = decrypted
	}

	return val, nil
}
