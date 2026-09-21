package models

// Scope format: "<resource>:<action>"
//
// Resources: tenants, users, queues, exchanges, bindings, workflows
// Actions:   admin (grants all actions on the resource), create, delete, edit, list
//
// When a token carries "<resource>:admin", it implicitly covers all other actions
// for that resource. The requireScope middleware accepts both the specific action
// and the admin scope as valid:
//
//   requireScope("queues:create", "queues:admin")
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

// AllScopes is the canonical ordered list of every valid scope.
// It is consumed by the Angular UI to render the scope-selection checkbox grid.
// Grouped by resource, with "admin" first in each group.
var AllScopes = []string{
	ScopeTenantAdmin, ScopeTenantCreate, ScopeTenantDelete, ScopeTenantEdit, ScopeTenantList,
	ScopeUserAdmin, ScopeUserCreate, ScopeUserDelete, ScopeUserEdit, ScopeUserList,
	ScopeQueueAdmin, ScopeQueueCreate, ScopeQueueDelete, ScopeQueueEdit, ScopeQueueList,
	ScopeExchangeAdmin, ScopeExchangeCreate, ScopeExchangeDelete, ScopeExchangeEdit, ScopeExchangeList,
	ScopeBindingAdmin, ScopeBindingCreate, ScopeBindingDelete, ScopeBindingEdit, ScopeBindingList,
	ScopeWorkflowAdmin, ScopeWorkflowCreate, ScopeWorkflowDelete, ScopeWorkflowEdit, ScopeWorkflowList,
}
