package db_test

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"
)

var (
	cwfTenants = []string{
		"alpha-01", "alpha-02", "alpha-03", "alpha-04", "alpha-05",
		"beta-01", "beta-02", "beta-03", "beta-04", "beta-05",
	}
	cwfVNamespaces = []string{
		"prod-api", "prod-rpc", "prod-events",
		"staging-api", "staging-rpc",
		"dev-api", "dev-rpc",
		"internal-jobs", "internal-batch", "internal-audit",
	}
	cwfQueues = []string{
		"orders", "payments", "notifications", "emails", "sms",
		"audit-log", "audit-access", "audit-changes",
		"batch-jobs", "batch-export",
	}
)

const (
	cwfNSSector    = "cwf-ns-sector"
	cwfQueueSector = "cwf-queue-sector"
	cwfPageSize    = 200
)

func setupCWFStore(t *testing.T) db.KVStore {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := db.CreatePebbleStore(tmpDir, []string{db.AdminFC}, []string{})
	require.NoError(t, err, "create pebble store")
	t.Cleanup(func() { _ = store.Close() })
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// tenants
	{
		uow := db.NewUnitOfWork(store, nil)
		repo, err := db.NewTenantInMasterRepository(uow, &db.DefaultIDGeneratorFactory{})
		require.NoError(t, err)
		for _, code := range cwfTenants {
			_, err := repo.CreateTenantInMaster(&models.TenantInMaster{
				Code:        code,
				Name:        fmt.Sprintf("Tenant %s", code),
				HasMessages: true,
			}, now)
			require.NoError(t, err, "create tenant %s", code)
		}
		require.NoError(t, uow.Commit())
	}
	// vnamespaces
	{
		uow := db.NewUnitOfWork(store, nil)
		repo, err := db.NewVNamespaceRepository(uow, &db.DefaultIDGeneratorFactory{}, db.AdminFC, cwfNSSector)
		require.NoError(t, err)
		for _, name := range cwfVNamespaces {
			_, err := repo.CreateVNamespace(&models.VNamespace{Name: name}, now)
			require.NoError(t, err, "create vnamespace %s", name)
		}
		require.NoError(t, uow.Commit())
	}
	// queues
	{
		uow := db.NewUnitOfWork(store, nil)
		repo, err := db.NewQueueRepository(uow, &db.DefaultIDGeneratorFactory{}, db.AdminFC, cwfQueueSector)
		require.NoError(t, err)
		for _, ns := range cwfVNamespaces {
			for _, code := range cwfQueues {
				_, err := repo.CreateQueue(&models.Queue{
					Code:          code,
					Name:          code,
					VNamespace:    ns,
					Type:          models.StandardQueue,
					State:         models.QueueActive,
					MessagesCount: 5,
					MaxAttempts:   1,
				}, now)
				require.NoError(t, err, "create queue %s/%s", ns, code)
			}
		}
		require.NoError(t, uow.Commit())
	}
	return store
}

func cwfTenantRepo(t *testing.T, store db.KVStore) *db.TenantInMasterRepository {
	t.Helper()
	uow := db.NewUnitOfWork(store, nil)
	repo, err := db.NewTenantInMasterRepository(uow, &db.DefaultIDGeneratorFactory{})
	require.NoError(t, err)
	return repo
}

func cwfNSRepo(t *testing.T, store db.KVStore) *db.VNamespaceRepository {
	t.Helper()
	uow := db.NewUnitOfWork(store, nil)
	repo, err := db.NewVNamespaceRepository(uow, &db.DefaultIDGeneratorFactory{}, db.AdminFC, cwfNSSector)
	require.NoError(t, err)
	return repo
}

func cwfQueueRepo(t *testing.T, store db.KVStore) *db.QueueRepository {
	t.Helper()
	uow := db.NewUnitOfWork(store, nil)
	repo, err := db.NewQueueRepository(uow, &db.DefaultIDGeneratorFactory{}, db.AdminFC, cwfQueueSector)
	require.NoError(t, err)
	return repo
}

func sortedTenantCodes(r *db.FindResult[models.TenantInMaster]) []string {
	out := make([]string, len(r.Entities))
	for i, e := range r.Entities {
		out[i] = e.Code
	}
	sort.Strings(out)
	return out
}

func sortedNSNames(r *db.FindResult[models.VNamespace]) []string {
	out := make([]string, len(r.Entities))
	for i, e := range r.Entities {
		out[i] = e.Name
	}
	sort.Strings(out)
	return out
}

// =============================================================================
// TENANT FILTER TESTS
// =============================================================================

func TestPebbleClaimWorkFilter_Tenants(t *testing.T) {
	store := setupCWFStore(t)
	now := time.Now()
	type tc struct {
		name         string
		filter       models.ClaimWorkFilter
		wantCount    int
		wantContains []string
		wantExcludes []string
	}
	tests := []tc{
		{
			name:         "empty filter returns all 10 tenants",
			filter:       models.ClaimWorkFilter{},
			wantCount:    10,
			wantContains: []string{"alpha-01", "beta-05"},
		},
		{
			name:         "single TenantCode",
			filter:       models.ClaimWorkFilter{TenantCodes: []string{"alpha-01"}},
			wantCount:    1,
			wantContains: []string{"alpha-01"},
			wantExcludes: []string{"alpha-02", "beta-01"},
		},
		{
			name:         "two TenantCodes",
			filter:       models.ClaimWorkFilter{TenantCodes: []string{"alpha-01", "beta-01"}},
			wantCount:    2,
			wantContains: []string{"alpha-01", "beta-01"},
			wantExcludes: []string{"alpha-02"},
		},
		{
			name:         "TenantPattern alpha-*",
			filter:       models.ClaimWorkFilter{TenantPatterns: []string{"alpha-*"}},
			wantCount:    5,
			wantContains: []string{"alpha-01", "alpha-02", "alpha-03", "alpha-04", "alpha-05"},
			wantExcludes: []string{"beta-01"},
		},
		{
			name:         "TenantPattern beta-*",
			filter:       models.ClaimWorkFilter{TenantPatterns: []string{"beta-*"}},
			wantCount:    5,
			wantContains: []string{"beta-01", "beta-02", "beta-03", "beta-04", "beta-05"},
			wantExcludes: []string{"alpha-01"},
		},
		{
			name:         "TenantPatterns alpha-* beta-* combined returns all 10",
			filter:       models.ClaimWorkFilter{TenantPatterns: []string{"alpha-*", "beta-*"}},
			wantCount:    10,
			wantContains: []string{"alpha-01", "beta-05"},
		},
		{
			name:         "TenantCodes + TenantPatterns union",
			filter:       models.ClaimWorkFilter{TenantCodes: []string{"beta-01"}, TenantPatterns: []string{"alpha-*"}},
			wantCount:    6,
			wantContains: []string{"alpha-01", "alpha-05", "beta-01"},
			wantExcludes: []string{"beta-02"},
		},
		{
			name:         "single ExcludeTenantCode",
			filter:       models.ClaimWorkFilter{ExcludeTenantCodes: []string{"alpha-01"}},
			wantCount:    9,
			wantContains: []string{"alpha-02", "beta-01"},
			wantExcludes: []string{"alpha-01"},
		},
		{
			name: "ExcludeTenantCodes removes whole alpha group",
			filter: models.ClaimWorkFilter{
				ExcludeTenantCodes: []string{"alpha-01", "alpha-02", "alpha-03", "alpha-04", "alpha-05"},
			},
			wantCount:    5,
			wantContains: []string{"beta-01", "beta-02", "beta-03", "beta-04", "beta-05"},
			wantExcludes: []string{"alpha-01"},
		},
		{
			name:         "ExcludeTenantPattern alpha-* keeps only betas",
			filter:       models.ClaimWorkFilter{ExcludeTenantPatterns: []string{"alpha-*"}},
			wantCount:    5,
			wantContains: []string{"beta-01", "beta-02", "beta-03", "beta-04", "beta-05"},
			wantExcludes: []string{"alpha-01"},
		},
		{
			name:         "ExcludeTenantPattern beta-* keeps only alphas",
			filter:       models.ClaimWorkFilter{ExcludeTenantPatterns: []string{"beta-*"}},
			wantCount:    5,
			wantContains: []string{"alpha-01", "alpha-02", "alpha-03", "alpha-04", "alpha-05"},
			wantExcludes: []string{"beta-01"},
		},
		{
			name: "TenantPatterns alpha-* + ExcludeTenantCodes alpha-01 alpha-02",
			filter: models.ClaimWorkFilter{
				TenantPatterns:     []string{"alpha-*"},
				ExcludeTenantCodes: []string{"alpha-01", "alpha-02"},
			},
			wantCount:    3,
			wantContains: []string{"alpha-03", "alpha-04", "alpha-05"},
			wantExcludes: []string{"alpha-01", "alpha-02", "beta-01"},
		},
		{
			name: "TenantPatterns alpha-* + ExcludeTenantPatterns *-05",
			filter: models.ClaimWorkFilter{
				TenantPatterns:        []string{"alpha-*"},
				ExcludeTenantPatterns: []string{"*-05"},
			},
			wantCount:    4,
			wantContains: []string{"alpha-01", "alpha-02", "alpha-03", "alpha-04"},
			wantExcludes: []string{"alpha-05", "beta-01"},
		},
		{
			name: "TenantCode + ExcludeSameCode yields empty",
			filter: models.ClaimWorkFilter{
				TenantCodes:        []string{"alpha-01"},
				ExcludeTenantCodes: []string{"alpha-01"},
			},
			wantCount: 0,
		},
		{
			name: "all fields combined",
			filter: models.ClaimWorkFilter{
				TenantCodes:           []string{"beta-01"},
				TenantPatterns:        []string{"alpha-*"},
				ExcludeTenantCodes:    []string{"alpha-01"},
				ExcludeTenantPatterns: []string{"*-05"},
			},
			wantCount:    4,
			wantContains: []string{"alpha-02", "alpha-03", "alpha-04", "beta-01"},
			wantExcludes: []string{"alpha-01", "alpha-05", "beta-02"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := cwfTenantRepo(t, store)
			result, err := repo.PaginateWithClaimWorkFilter(tc.filter, cwfPageSize, "", now)
			require.NoError(t, err)
			got := sortedTenantCodes(result)
			assert.Len(t, got, tc.wantCount, "got codes: %v", got)
			for _, code := range tc.wantContains {
				assert.Contains(t, got, code)
			}
			for _, code := range tc.wantExcludes {
				assert.NotContains(t, got, code)
			}
		})
	}
}

// =============================================================================
// VNAMESPACE FILTER TESTS
// =============================================================================

func TestPebbleClaimWorkFilter_VNamespaces(t *testing.T) {
	store := setupCWFStore(t)
	now := time.Now()
	type tc struct {
		name         string
		filter       models.ClaimWorkFilter
		wantCount    int
		wantContains []string
		wantExcludes []string
	}
	tests := []tc{
		{
			name:         "empty filter returns all 10 vnamespaces",
			filter:       models.ClaimWorkFilter{},
			wantCount:    10,
			wantContains: []string{"prod-api", "internal-audit"},
		},
		{
			name:         "single VNamespace exact match",
			filter:       models.ClaimWorkFilter{VNamespaces: []string{"prod-api"}},
			wantCount:    1,
			wantContains: []string{"prod-api"},
			wantExcludes: []string{"prod-rpc", "staging-api"},
		},
		{
			name:         "two VNamespaces exact match",
			filter:       models.ClaimWorkFilter{VNamespaces: []string{"prod-api", "staging-api"}},
			wantCount:    2,
			wantContains: []string{"prod-api", "staging-api"},
			wantExcludes: []string{"prod-rpc", "dev-api"},
		},
		{
			name:         "VNamespacePattern prod-*",
			filter:       models.ClaimWorkFilter{VNamespacePatterns: []string{"prod-*"}},
			wantCount:    3,
			wantContains: []string{"prod-api", "prod-rpc", "prod-events"},
			wantExcludes: []string{"staging-api", "dev-api"},
		},
		{
			name:         "VNamespacePattern staging-*",
			filter:       models.ClaimWorkFilter{VNamespacePatterns: []string{"staging-*"}},
			wantCount:    2,
			wantContains: []string{"staging-api", "staging-rpc"},
			wantExcludes: []string{"prod-api", "dev-api"},
		},
		{
			name: "VNamespacePatterns prod-* staging-* combined",
			filter: models.ClaimWorkFilter{
				VNamespacePatterns: []string{"prod-*", "staging-*"},
			},
			wantCount:    5,
			wantContains: []string{"prod-api", "prod-rpc", "prod-events", "staging-api", "staging-rpc"},
			wantExcludes: []string{"dev-api", "internal-jobs"},
		},
		{
			name: "VNamespaces + VNamespacePattern union",
			filter: models.ClaimWorkFilter{
				VNamespaces:        []string{"dev-api"},
				VNamespacePatterns: []string{"prod-*"},
			},
			wantCount:    4,
			wantContains: []string{"prod-api", "prod-rpc", "prod-events", "dev-api"},
			wantExcludes: []string{"dev-rpc"},
		},
		{
			name:         "single ExcludeVNamespace",
			filter:       models.ClaimWorkFilter{ExcludeVNamespaces: []string{"prod-api"}},
			wantCount:    9,
			wantContains: []string{"prod-rpc"},
			wantExcludes: []string{"prod-api"},
		},
		{
			name:         "ExcludeVNamespaces dev set removed",
			filter:       models.ClaimWorkFilter{ExcludeVNamespaces: []string{"dev-api", "dev-rpc"}},
			wantCount:    8,
			wantContains: []string{"prod-api"},
			wantExcludes: []string{"dev-api", "dev-rpc"},
		},
		{
			name: "ExcludeVNamespacePattern internal-* removes 3",
			filter: models.ClaimWorkFilter{
				ExcludeVNamespacePatterns: []string{"internal-*"},
			},
			wantCount:    7,
			wantContains: []string{"prod-api", "staging-api", "dev-api"},
			wantExcludes: []string{"internal-jobs", "internal-batch", "internal-audit"},
		},
		{
			name: "VNamespacePattern prod-* + ExcludeVNamespace prod-events",
			filter: models.ClaimWorkFilter{
				VNamespacePatterns: []string{"prod-*"},
				ExcludeVNamespaces: []string{"prod-events"},
			},
			wantCount:    2,
			wantContains: []string{"prod-api", "prod-rpc"},
			wantExcludes: []string{"prod-events"},
		},
		{
			name: "VNamespacePatterns prod-* staging-* + ExcludeVNamespacePattern *-rpc",
			filter: models.ClaimWorkFilter{
				VNamespacePatterns:        []string{"prod-*", "staging-*"},
				ExcludeVNamespacePatterns: []string{"*-rpc"},
			},
			wantCount:    3,
			wantContains: []string{"prod-api", "prod-events", "staging-api"},
			wantExcludes: []string{"prod-rpc", "staging-rpc"},
		},
		{
			name: "VNamespace exact + ExcludeVNamespacePattern *-api",
			filter: models.ClaimWorkFilter{
				VNamespaces:               []string{"prod-api", "prod-rpc", "staging-api"},
				ExcludeVNamespacePatterns: []string{"*-api"},
			},
			wantCount:    1,
			wantContains: []string{"prod-rpc"},
			wantExcludes: []string{"prod-api", "staging-api"},
		},
		{
			name: "VNamespace + ExcludeSame yields empty",
			filter: models.ClaimWorkFilter{
				VNamespaces:        []string{"prod-api"},
				ExcludeVNamespaces: []string{"prod-api"},
			},
			wantCount: 0,
		},
		{
			name: "all fields combined",
			filter: models.ClaimWorkFilter{
				VNamespaces:               []string{"dev-api"},
				VNamespacePatterns:        []string{"prod-*"},
				ExcludeVNamespaces:        []string{"prod-api"},
				ExcludeVNamespacePatterns: []string{"*-events"},
			},
			wantCount:    2,
			wantContains: []string{"prod-rpc", "dev-api"},
			wantExcludes: []string{"prod-api", "prod-events"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := cwfNSRepo(t, store)
			result, err := repo.PaginateWithClaimWorkFilter(tc.filter, cwfPageSize, "", now)
			require.NoError(t, err)
			got := sortedNSNames(result)
			assert.Len(t, got, tc.wantCount, "got names: %v", got)
			for _, name := range tc.wantContains {
				assert.Contains(t, got, name)
			}
			for _, name := range tc.wantExcludes {
				assert.NotContains(t, got, name)
			}
		})
	}
}

// =============================================================================
// QUEUE FILTER TESTS
// =============================================================================

func TestPebbleClaimWorkFilter_Queues(t *testing.T) {
	store := setupCWFStore(t)
	now := time.Now()
	uniqueCodes := func(r *db.FindResult[models.Queue]) []string {
		seen := map[string]struct{}{}
		for _, q := range r.Entities {
			seen[q.Code] = struct{}{}
		}
		out := make([]string, 0, len(seen))
		for c := range seen {
			out = append(out, c)
		}
		sort.Strings(out)
		return out
	}
	type tc struct {
		name            string
		filter          models.ClaimWorkFilter
		wantTotalCount  int
		wantUniqCodes   []string
		wantAbsentCodes []string
	}
	tests := []tc{
		{
			name:           "empty filter no vns returns all 100",
			filter:         models.ClaimWorkFilter{},
			wantTotalCount: 100,
			wantUniqCodes:  cwfQueues,
		},
		{
			name:           "empty filter vns=prod-api returns 10",
			filter:         models.ClaimWorkFilter{VNamespaces: []string{"prod-api"}},
			wantTotalCount: 10,
			wantUniqCodes:  cwfQueues,
		},
		{
			name:            "single QueueCode orders no vns returns 10",
			filter:          models.ClaimWorkFilter{QueueCodes: []string{"orders"}},
			wantTotalCount:  10,
			wantUniqCodes:   []string{"orders"},
			wantAbsentCodes: []string{"payments", "emails"},
		},
		{
			name:           "single QueueCode orders + vns=prod-api returns 1",
			filter:         models.ClaimWorkFilter{QueueCodes: []string{"orders"}, VNamespaces: []string{"prod-api"}},
			wantTotalCount: 1,
			wantUniqCodes:  []string{"orders"},
		},
		{
			name:            "two QueueCodes no vns returns 20",
			filter:          models.ClaimWorkFilter{QueueCodes: []string{"orders", "payments"}},
			wantTotalCount:  20,
			wantUniqCodes:   []string{"orders", "payments"},
			wantAbsentCodes: []string{"notifications"},
		},
		{
			name:            "QueuePattern audit-* no vns returns 30",
			filter:          models.ClaimWorkFilter{QueuePatterns: []string{"audit-*"}},
			wantTotalCount:  30,
			wantUniqCodes:   []string{"audit-log", "audit-access", "audit-changes"},
			wantAbsentCodes: []string{"orders", "batch-jobs"},
		},
		{
			name: "QueuePattern batch-* no vns returns 20",
			filter: models.ClaimWorkFilter{
				QueuePatterns: []string{"batch-*"},
			},
			wantTotalCount:  20,
			wantUniqCodes:   []string{"batch-jobs", "batch-export"},
			wantAbsentCodes: []string{"orders", "payments"},
		},
		{
			name: "QueueCodes + QueuePatterns union returns 30",
			filter: models.ClaimWorkFilter{
				QueueCodes:    []string{"orders"},
				QueuePatterns: []string{"batch-*"},
			},
			wantTotalCount:  30,
			wantUniqCodes:   []string{"orders", "batch-jobs", "batch-export"},
			wantAbsentCodes: []string{"payments", "audit-log"},
		},
		{
			name: "ExcludeQueueCode orders no vns returns 90",
			filter: models.ClaimWorkFilter{
				ExcludeQueueCodes: []string{"orders"},
			},
			wantTotalCount:  90,
			wantUniqCodes:   []string{"payments", "batch-jobs"},
			wantAbsentCodes: []string{"orders"},
		},
		{
			name: "ExcludeQueueCodes orders payments returns 80",
			filter: models.ClaimWorkFilter{
				ExcludeQueueCodes: []string{"orders", "payments"},
			},
			wantTotalCount:  80,
			wantAbsentCodes: []string{"orders", "payments"},
			wantUniqCodes:   []string{"notifications", "batch-export"},
		},
		{
			name: "ExcludeQueuePattern batch-* no vns returns 80",
			filter: models.ClaimWorkFilter{
				ExcludeQueuePatterns: []string{"batch-*"},
			},
			wantTotalCount:  80,
			wantAbsentCodes: []string{"batch-jobs", "batch-export"},
			wantUniqCodes:   []string{"orders", "audit-log"},
		},
		{
			name: "ExcludeQueuePattern audit-* no vns returns 70",
			filter: models.ClaimWorkFilter{
				ExcludeQueuePatterns: []string{"audit-*"},
			},
			wantTotalCount:  70,
			wantAbsentCodes: []string{"audit-log", "audit-access", "audit-changes"},
			wantUniqCodes:   []string{"orders", "batch-jobs"},
		},
		{
			name: "QueueCode + ExcludeSame returns 0",
			filter: models.ClaimWorkFilter{
				QueueCodes:        []string{"orders"},
				ExcludeQueueCodes: []string{"orders"},
			},
			wantTotalCount: 0,
		},
		{
			name: "QueuePattern audit-* + ExcludeQueueCode audit-log returns 20",
			filter: models.ClaimWorkFilter{
				QueuePatterns:     []string{"audit-*"},
				ExcludeQueueCodes: []string{"audit-log"},
			},
			wantTotalCount:  20,
			wantUniqCodes:   []string{"audit-access", "audit-changes"},
			wantAbsentCodes: []string{"audit-log", "orders"},
		},
		{
			name: "QueuePattern audit-* + ExcludeQueuePattern *-log returns 20",
			filter: models.ClaimWorkFilter{
				QueuePatterns:        []string{"audit-*"},
				ExcludeQueuePatterns: []string{"*-log"},
			},
			wantTotalCount:  20,
			wantUniqCodes:   []string{"audit-access", "audit-changes"},
			wantAbsentCodes: []string{"audit-log"},
		},
		{
			name: "vns=prod-api + ExcludeQueuePattern batch-* returns 8",
			filter: models.ClaimWorkFilter{
				ExcludeQueuePatterns: []string{"batch-*"},
				VNamespaces:          []string{"prod-api"},
			},
			wantTotalCount:  8,
			wantAbsentCodes: []string{"batch-jobs", "batch-export"},
			wantUniqCodes:   []string{"orders", "audit-log"},
		},
		{
			name: "QueueCodes orders+payments vns=staging-api returns 2",
			filter: models.ClaimWorkFilter{
				QueueCodes:  []string{"orders", "payments"},
				VNamespaces: []string{"staging-api"},
			},
			wantTotalCount: 2,
			wantUniqCodes:  []string{"orders", "payments"},
		},
		{
			name: "all fields combined",
			filter: models.ClaimWorkFilter{
				QueueCodes:           []string{"orders"},
				QueuePatterns:        []string{"audit-*"},
				ExcludeQueueCodes:    []string{"audit-log"},
				ExcludeQueuePatterns: []string{"audit-changes"},
			},
			wantTotalCount:  20,
			wantUniqCodes:   []string{"orders", "audit-access"},
			wantAbsentCodes: []string{"audit-log", "audit-changes", "payments"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := cwfQueueRepo(t, store)
			result, err := repo.PaginateWithClaimWorkFilter(tc.filter, cwfPageSize, "", now)
			require.NoError(t, err)
			assert.Len(t, result.Entities, tc.wantTotalCount, "unique codes: %v", uniqueCodes(result))
			uniq := uniqueCodes(result)
			for _, code := range tc.wantUniqCodes {
				assert.Contains(t, uniq, code)
			}
			for _, code := range tc.wantAbsentCodes {
				assert.NotContains(t, uniq, code)
			}
		})
	}
}

func TestPebbleClaimWorkFilter_Queues_ZeroMessagesExcluded(t *testing.T) {
	store := setupCWFStore(t)
	now := time.Now()
	{
		uow := db.NewUnitOfWork(store, nil)
		repo, err := db.NewQueueRepository(uow, &db.DefaultIDGeneratorFactory{}, db.AdminFC, cwfQueueSector)
		require.NoError(t, err)
		_, err = repo.CreateQueue(&models.Queue{
			Code:          "empty-queue",
			Name:          "empty-queue",
			VNamespace:    "prod-api",
			Type:          models.StandardQueue,
			State:         models.QueueActive,
			MessagesCount: 0,
			MaxAttempts:   1,
		}, now)
		require.NoError(t, err)
		require.NoError(t, uow.Commit())
	}
	repo := cwfQueueRepo(t, store)
	result, err := repo.PaginateWithClaimWorkFilter(
		models.ClaimWorkFilter{
			QueueCodes:  []string{"empty-queue"},
			VNamespaces: []string{"prod-api"},
		},
		cwfPageSize, "", now,
	)
	require.NoError(t, err)
	assert.Empty(t, result.Entities, "MessagesCount=0 queue must not be returned")
}
