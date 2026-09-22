package models

const (
	ScopeTenantAdmin  = "tenants:admin"
	ScopeTenantCreate = "tenants:create"
	ScopeTenantDelete = "tenants:delete"
	ScopeTenantEdit   = "tenants:edit"
	ScopeTenantList   = "tenants:list"

	ScopeUserAdmin  = "users:admin"
	ScopeUserCreate = "users:create"
	ScopeUserDelete = "users:delete"
	ScopeUserEdit   = "users:edit"
	ScopeUserList   = "users:list"

	ScopeQueueAdmin  = "queues:admin"
	ScopeQueueCreate = "queues:create"
	ScopeQueueDelete = "queues:delete"
	ScopeQueueEdit   = "queues:edit"
	ScopeQueueList   = "queues:list"

	ScopeExchangeAdmin  = "exchanges:admin"
	ScopeExchangeCreate = "exchanges:create"
	ScopeExchangeDelete = "exchanges:delete"
	ScopeExchangeEdit   = "exchanges:edit"
	ScopeExchangeList   = "exchanges:list"

	ScopeBindingAdmin  = "bindings:admin"
	ScopeBindingCreate = "bindings:create"
	ScopeBindingDelete = "bindings:delete"
	ScopeBindingEdit   = "bindings:edit"
	ScopeBindingList   = "bindings:list"

	ScopeWorkflowAdmin  = "workflows:admin"
	ScopeWorkflowCreate = "workflows:create"
	ScopeWorkflowDelete = "workflows:delete"
	ScopeWorkflowEdit   = "workflows:edit"
	ScopeWorkflowList   = "workflows:list"
)

// AllScopes is the canonical ordered list consumed by UI checkbox grids.
var AllScopes = []string{
	ScopeTenantAdmin, ScopeTenantCreate, ScopeTenantDelete, ScopeTenantEdit, ScopeTenantList,
	ScopeUserAdmin, ScopeUserCreate, ScopeUserDelete, ScopeUserEdit, ScopeUserList,
	ScopeQueueAdmin, ScopeQueueCreate, ScopeQueueDelete, ScopeQueueEdit, ScopeQueueList,
	ScopeExchangeAdmin, ScopeExchangeCreate, ScopeExchangeDelete, ScopeExchangeEdit, ScopeExchangeList,
	ScopeBindingAdmin, ScopeBindingCreate, ScopeBindingDelete, ScopeBindingEdit, ScopeBindingList,
	ScopeWorkflowAdmin, ScopeWorkflowCreate, ScopeWorkflowDelete, ScopeWorkflowEdit, ScopeWorkflowList,
}
