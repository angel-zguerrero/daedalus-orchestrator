package common

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/shared/models"
	"fmt"
)

// TenantContext contiene información del tenant y su nodo asociado para inyectar en el contexto
type TenantContext struct {
	Tenant *models.TenantInMaster
	Node   *dragonboat.RaftNode
	CF     string // Column Family prefix
	CFS    string // Column Family suffix (tenant ID)
}

// Claves para el contexto
type contextKey string

const (
	TenantContextKey contextKey = "tenant_context"
)

// GetTenantContext extrae el TenantContext del contexto
func GetTenantContext(ctx context.Context) (*TenantContext, bool) {
	tenantCtx, ok := ctx.Value(TenantContextKey).(*TenantContext)
	return tenantCtx, ok
}

// SetTenantContext inyecta el TenantContext en el contexto
func SetTenantContext(ctx context.Context, tenantCtx *TenantContext) context.Context {
	return context.WithValue(ctx, TenantContextKey, tenantCtx)
}

// MustGetTenantContext extrae el TenantContext del contexto y entra en pánico si no existe
func MustGetTenantContext(ctx context.Context) *TenantContext {
	tenantCtx, ok := GetTenantContext(ctx)
	if !ok {
		panic("tenant context not found - make sure tenant interceptor is configured")
	}
	return tenantCtx
}

// GetTenantData es una función conveniente que retorna los datos más usados del tenant
func GetTenantData(ctx context.Context) (*models.TenantInMaster, *dragonboat.RaftNode, string, string, error) {
	tenantCtx, ok := GetTenantContext(ctx)
	if !ok {
		return nil, nil, "", "", fmt.Errorf("tenant context not found")
	}
	return tenantCtx.Tenant, tenantCtx.Node, tenantCtx.CF, tenantCtx.CFS, nil
}

// MustGetTenantData es como GetTenantData pero entra en pánico si no encuentra el contexto
func MustGetTenantData(ctx context.Context) (*models.TenantInMaster, *dragonboat.RaftNode, string, string) {
	tenant, node, cf, cfs, err := GetTenantData(ctx)
	if err != nil {
		panic(err)
	}
	return tenant, node, cf, cfs
}
