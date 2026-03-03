package db_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"
)

// TestBuildTenantFilterQuery covers all ClaimWorkFilter possibilities used by
// GetTenantsWithFilter via PaginateWithClaimWorkFilter on TenantInMasterRepository.
func TestBuildTenantFilterQuery(t *testing.T) {
	tests := []struct {
		name        string
		filter      models.ClaimWorkFilter
		wantDBQuery string
	}{
		{
			name:        "empty filter",
			filter:      models.ClaimWorkFilter{},
			wantDBQuery: "ID != 0",
		},
		{
			name:        "single TenantCode",
			filter:      models.ClaimWorkFilter{TenantCodes: []string{"acme"}},
			wantDBQuery: "Code = acme",
		},
		{
			name:        "multiple TenantCodes",
			filter:      models.ClaimWorkFilter{TenantCodes: []string{"acme", "globex"}},
			wantDBQuery: "(Code = acme | Code = globex)",
		},
		{
			name:        "single TenantPattern",
			filter:      models.ClaimWorkFilter{TenantPatterns: []string{"acme-*"}},
			wantDBQuery: "Code LIKE acme-*",
		},
		{
			name:        "multiple TenantPatterns",
			filter:      models.ClaimWorkFilter{TenantPatterns: []string{"acme-*", "beta-*"}},
			wantDBQuery: "(Code LIKE acme-* | Code LIKE beta-*)",
		},
		{
			name: "TenantCodes and TenantPatterns combined",
			filter: models.ClaimWorkFilter{
				TenantCodes:    []string{"acme"},
				TenantPatterns: []string{"beta-*"},
			},
			wantDBQuery: "(Code = acme | Code LIKE beta-*)",
		},
		{
			name:        "single ExcludeTenantCode",
			filter:      models.ClaimWorkFilter{ExcludeTenantCodes: []string{"evil-corp"}},
			wantDBQuery: "Code != evil-corp",
		},
		{
			name:        "multiple ExcludeTenantCodes",
			filter:      models.ClaimWorkFilter{ExcludeTenantCodes: []string{"evil-corp", "bad-co"}},
			wantDBQuery: "Code != evil-corp & Code != bad-co",
		},
		{
			name:        "single ExcludeTenantPattern uses NOT LIKE",
			filter:      models.ClaimWorkFilter{ExcludeTenantPatterns: []string{"internal-*"}},
			wantDBQuery: "Code NOT LIKE internal-*",
		},
		{
			name:        "multiple ExcludeTenantPatterns use NOT LIKE",
			filter:      models.ClaimWorkFilter{ExcludeTenantPatterns: []string{"internal-*", "test-*"}},
			wantDBQuery: "Code NOT LIKE internal-* & Code NOT LIKE test-*",
		},
		{
			name: "TenantCodes and ExcludeTenantCodes",
			filter: models.ClaimWorkFilter{
				TenantCodes:        []string{"acme", "globex"},
				ExcludeTenantCodes: []string{"globex"},
			},
			wantDBQuery: "(Code = acme | Code = globex) & Code != globex",
		},
		{
			name: "TenantCodes and ExcludeTenantPatterns",
			filter: models.ClaimWorkFilter{
				TenantCodes:           []string{"acme"},
				ExcludeTenantPatterns: []string{"internal-*"},
			},
			wantDBQuery: "Code = acme & Code NOT LIKE internal-*",
		},
		{
			name: "TenantPatterns and ExcludeTenantCodes",
			filter: models.ClaimWorkFilter{
				TenantPatterns:     []string{"prod-*"},
				ExcludeTenantCodes: []string{"prod-legacy"},
			},
			wantDBQuery: "Code LIKE prod-* & Code != prod-legacy",
		},
		{
			name: "TenantPatterns and ExcludeTenantPatterns",
			filter: models.ClaimWorkFilter{
				TenantPatterns:        []string{"prod-*"},
				ExcludeTenantPatterns: []string{"prod-internal-*"},
			},
			wantDBQuery: "Code LIKE prod-* & Code NOT LIKE prod-internal-*",
		},
		{
			name: "all tenant fields combined",
			filter: models.ClaimWorkFilter{
				TenantCodes:           []string{"acme"},
				TenantPatterns:        []string{"prod-*"},
				ExcludeTenantCodes:    []string{"prod-legacy"},
				ExcludeTenantPatterns: []string{"prod-internal-*"},
			},
			wantDBQuery: "(Code = acme | Code LIKE prod-*) & Code != prod-legacy & Code NOT LIKE prod-internal-*",
		},
		{
			name:        "ExcludeTenantCodes only",
			filter:      models.ClaimWorkFilter{ExcludeTenantCodes: []string{"acme"}},
			wantDBQuery: "Code != acme",
		},
		{
			name: "ExcludeTenantCodes and ExcludeTenantPatterns",
			filter: models.ClaimWorkFilter{
				ExcludeTenantCodes:    []string{"evil-corp"},
				ExcludeTenantPatterns: []string{"test-*"},
			},
			wantDBQuery: "Code != evil-corp & Code NOT LIKE test-*",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := db.BuildTenantFilterQuery(tc.filter)
			assert.Equal(t, tc.wantDBQuery, got.DBQuery)
		})
	}
}

// TestBuildVNamespaceFilterQuery covers all ClaimWorkFilter possibilities used by
// GetVNamespacesWithFilter via PaginateWithClaimWorkFilter on VNamespaceRepository.
func TestBuildVNamespaceFilterQuery(t *testing.T) {
	tests := []struct {
		name        string
		filter      models.ClaimWorkFilter
		wantDBQuery string
	}{
		{
			name:        "empty filter",
			filter:      models.ClaimWorkFilter{},
			wantDBQuery: "ID != 0",
		},
		{
			name:        "single VNamespace",
			filter:      models.ClaimWorkFilter{VNamespaces: []string{"default"}},
			wantDBQuery: "Name = default",
		},
		{
			name:        "multiple VNamespaces",
			filter:      models.ClaimWorkFilter{VNamespaces: []string{"default", "staging"}},
			wantDBQuery: "(Name = default | Name = staging)",
		},
		{
			name:        "single VNamespacePattern",
			filter:      models.ClaimWorkFilter{VNamespacePatterns: []string{"prod-*"}},
			wantDBQuery: "Name LIKE prod-*",
		},
		{
			name:        "multiple VNamespacePatterns",
			filter:      models.ClaimWorkFilter{VNamespacePatterns: []string{"prod-*", "staging-*"}},
			wantDBQuery: "(Name LIKE prod-* | Name LIKE staging-*)",
		},
		{
			name: "VNamespaces and VNamespacePatterns combined",
			filter: models.ClaimWorkFilter{
				VNamespaces:        []string{"default"},
				VNamespacePatterns: []string{"prod-*"},
			},
			wantDBQuery: "(Name = default | Name LIKE prod-*)",
		},
		{
			name:        "single ExcludeVNamespace",
			filter:      models.ClaimWorkFilter{ExcludeVNamespaces: []string{"internal"}},
			wantDBQuery: "Name != internal",
		},
		{
			name:        "multiple ExcludeVNamespaces",
			filter:      models.ClaimWorkFilter{ExcludeVNamespaces: []string{"internal", "legacy"}},
			wantDBQuery: "Name != internal & Name != legacy",
		},
		{
			name:        "single ExcludeVNamespacePattern uses NOT LIKE",
			filter:      models.ClaimWorkFilter{ExcludeVNamespacePatterns: []string{"internal-*"}},
			wantDBQuery: "Name NOT LIKE internal-*",
		},
		{
			name:        "multiple ExcludeVNamespacePatterns use NOT LIKE",
			filter:      models.ClaimWorkFilter{ExcludeVNamespacePatterns: []string{"internal-*", "test-*"}},
			wantDBQuery: "Name NOT LIKE internal-* & Name NOT LIKE test-*",
		},
		{
			name: "VNamespaces and ExcludeVNamespaces",
			filter: models.ClaimWorkFilter{
				VNamespaces:        []string{"default", "staging"},
				ExcludeVNamespaces: []string{"staging"},
			},
			wantDBQuery: "(Name = default | Name = staging) & Name != staging",
		},
		{
			name: "VNamespaces and ExcludeVNamespacePatterns",
			filter: models.ClaimWorkFilter{
				VNamespaces:               []string{"default"},
				ExcludeVNamespacePatterns: []string{"internal-*"},
			},
			wantDBQuery: "Name = default & Name NOT LIKE internal-*",
		},
		{
			name: "VNamespacePatterns and ExcludeVNamespaces",
			filter: models.ClaimWorkFilter{
				VNamespacePatterns: []string{"prod-*"},
				ExcludeVNamespaces: []string{"prod-legacy"},
			},
			wantDBQuery: "Name LIKE prod-* & Name != prod-legacy",
		},
		{
			name: "VNamespacePatterns and ExcludeVNamespacePatterns",
			filter: models.ClaimWorkFilter{
				VNamespacePatterns:        []string{"prod-*"},
				ExcludeVNamespacePatterns: []string{"prod-internal-*"},
			},
			wantDBQuery: "Name LIKE prod-* & Name NOT LIKE prod-internal-*",
		},
		{
			name: "all vnamespace fields combined",
			filter: models.ClaimWorkFilter{
				VNamespaces:               []string{"default"},
				VNamespacePatterns:        []string{"prod-*"},
				ExcludeVNamespaces:        []string{"prod-legacy"},
				ExcludeVNamespacePatterns: []string{"prod-internal-*"},
			},
			wantDBQuery: "(Name = default | Name LIKE prod-*) & Name != prod-legacy & Name NOT LIKE prod-internal-*",
		},
		{
			name:        "ExcludeVNamespaces only",
			filter:      models.ClaimWorkFilter{ExcludeVNamespaces: []string{"internal"}},
			wantDBQuery: "Name != internal",
		},
		{
			name: "ExcludeVNamespaces and ExcludeVNamespacePatterns",
			filter: models.ClaimWorkFilter{
				ExcludeVNamespaces:        []string{"internal"},
				ExcludeVNamespacePatterns: []string{"test-*"},
			},
			wantDBQuery: "Name != internal & Name NOT LIKE test-*",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := db.BuildVNamespaceFilterQuery(tc.filter)
			assert.Equal(t, tc.wantDBQuery, got.DBQuery)
		})
	}
}

// TestBuildQueueFilterQuery covers all ClaimWorkFilter possibilities used by
// GetQueuesWithFilter via PaginateWithClaimWorkFilter on QueueRepository.
// MessagesCount > 0 is always appended; vNamespace is appended when non-empty.
func TestBuildQueueFilterQuery(t *testing.T) {
	tests := []struct {
		name        string
		filter      models.ClaimWorkFilter
		vNamespace  string
		wantDBQuery string
	}{
		{
			name:        "empty filter no vNamespace",
			filter:      models.ClaimWorkFilter{},
			wantDBQuery: "MessagesCount > 0",
		},
		{
			name:        "empty filter with vNamespace",
			filter:      models.ClaimWorkFilter{},
			vNamespace:  "prod",
			wantDBQuery: "MessagesCount > 0 & VNamespace = prod",
		},
		{
			name:        "single QueueCode",
			filter:      models.ClaimWorkFilter{QueueCodes: []string{"orders"}},
			wantDBQuery: "Code = orders & MessagesCount > 0",
		},
		{
			name:        "multiple QueueCodes",
			filter:      models.ClaimWorkFilter{QueueCodes: []string{"orders", "payments"}},
			wantDBQuery: "(Code = orders | Code = payments) & MessagesCount > 0",
		},
		{
			name:        "single QueuePattern",
			filter:      models.ClaimWorkFilter{QueuePatterns: []string{"order-*"}},
			wantDBQuery: "Code LIKE order-* & MessagesCount > 0",
		},
		{
			name:        "multiple QueuePatterns",
			filter:      models.ClaimWorkFilter{QueuePatterns: []string{"order-*", "payment-*"}},
			wantDBQuery: "(Code LIKE order-* | Code LIKE payment-*) & MessagesCount > 0",
		},
		{
			name: "QueueCodes and QueuePatterns combined",
			filter: models.ClaimWorkFilter{
				QueueCodes:    []string{"orders"},
				QueuePatterns: []string{"payment-*"},
			},
			wantDBQuery: "(Code = orders | Code LIKE payment-*) & MessagesCount > 0",
		},
		{
			name:        "single ExcludeQueueCode",
			filter:      models.ClaimWorkFilter{ExcludeQueueCodes: []string{"dead-letters"}},
			wantDBQuery: "Code != dead-letters & MessagesCount > 0",
		},
		{
			name:        "multiple ExcludeQueueCodes",
			filter:      models.ClaimWorkFilter{ExcludeQueueCodes: []string{"dead-letters", "retry"}},
			wantDBQuery: "Code != dead-letters & Code != retry & MessagesCount > 0",
		},
		{
			name:        "single ExcludeQueuePattern uses NOT LIKE",
			filter:      models.ClaimWorkFilter{ExcludeQueuePatterns: []string{"internal-*"}},
			wantDBQuery: "Code NOT LIKE internal-* & MessagesCount > 0",
		},
		{
			name:        "multiple ExcludeQueuePatterns use NOT LIKE",
			filter:      models.ClaimWorkFilter{ExcludeQueuePatterns: []string{"internal-*", "test-*"}},
			wantDBQuery: "Code NOT LIKE internal-* & Code NOT LIKE test-* & MessagesCount > 0",
		},
		{
			name: "QueueCodes and ExcludeQueueCodes",
			filter: models.ClaimWorkFilter{
				QueueCodes:        []string{"orders", "payments"},
				ExcludeQueueCodes: []string{"payments"},
			},
			wantDBQuery: "(Code = orders | Code = payments) & Code != payments & MessagesCount > 0",
		},
		{
			name: "QueueCodes and ExcludeQueuePatterns",
			filter: models.ClaimWorkFilter{
				QueueCodes:           []string{"orders"},
				ExcludeQueuePatterns: []string{"internal-*"},
			},
			wantDBQuery: "Code = orders & Code NOT LIKE internal-* & MessagesCount > 0",
		},
		{
			name: "QueuePatterns and ExcludeQueueCodes with vNamespace",
			filter: models.ClaimWorkFilter{
				QueuePatterns:     []string{"order-*"},
				ExcludeQueueCodes: []string{"order-archive"},
			},
			vNamespace:  "production",
			wantDBQuery: "Code LIKE order-* & Code != order-archive & MessagesCount > 0 & VNamespace = production",
		},
		{
			name: "QueuePatterns and ExcludeQueuePatterns",
			filter: models.ClaimWorkFilter{
				QueuePatterns:        []string{"prod-*"},
				ExcludeQueuePatterns: []string{"prod-internal-*"},
			},
			wantDBQuery: "Code LIKE prod-* & Code NOT LIKE prod-internal-* & MessagesCount > 0",
		},
		{
			name: "all queue fields combined with vNamespace",
			filter: models.ClaimWorkFilter{
				QueueCodes:           []string{"orders"},
				QueuePatterns:        []string{"payment-*"},
				ExcludeQueueCodes:    []string{"payment-archive"},
				ExcludeQueuePatterns: []string{"payment-internal-*"},
			},
			vNamespace:  "prod",
			wantDBQuery: "(Code = orders | Code LIKE payment-*) & Code != payment-archive & Code NOT LIKE payment-internal-* & MessagesCount > 0 & VNamespace = prod",
		},
		{
			name:        "ExcludeQueueCodes only",
			filter:      models.ClaimWorkFilter{ExcludeQueueCodes: []string{"dead-letters"}},
			wantDBQuery: "Code != dead-letters & MessagesCount > 0",
		},
		{
			name: "ExcludeQueueCodes and ExcludeQueuePatterns",
			filter: models.ClaimWorkFilter{
				ExcludeQueueCodes:    []string{"dead-letters"},
				ExcludeQueuePatterns: []string{"test-*"},
			},
			wantDBQuery: "Code != dead-letters & Code NOT LIKE test-* & MessagesCount > 0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := db.BuildQueueFilterQuery(tc.filter, tc.vNamespace)
			assert.Equal(t, tc.wantDBQuery, got.DBQuery)
		})
	}
}
