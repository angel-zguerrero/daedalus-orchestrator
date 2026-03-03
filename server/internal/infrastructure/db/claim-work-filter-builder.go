package db

import (
	"fmt"
	"strings"

	"deadalus-orch/shared/models"
)

// ClaimWorkFilterQuery is the result of translating a ClaimWorkFilter into DB-layer constructs.
// The DB query string encodes inclusion rules, exact exclusions, and NOT LIKE pattern exclusions.
type ClaimWorkFilterQuery struct {
	// DBQuery is the filter string ready to be passed to Repository.Find.
	DBQuery string
}

// BuildTenantFilterQuery converts the tenant-relevant parts of a ClaimWorkFilter into a
// ClaimWorkFilterQuery whose DBQuery can be passed directly to TenantInMasterRepository.Find.
func BuildTenantFilterQuery(f models.ClaimWorkFilter) ClaimWorkFilterQuery {
	var inclusionClauses []string

	// Exact-match allowlist
	for _, code := range f.TenantCodes {
		inclusionClauses = append(inclusionClauses, fmt.Sprintf("Code = %s", code))
	}
	// Pattern allowlist (glob → LIKE)
	for _, pat := range f.TenantPatterns {
		inclusionClauses = append(inclusionClauses, fmt.Sprintf("Code LIKE %s", pat))
	}

	var exclusionClauses []string
	for _, code := range f.ExcludeTenantCodes {
		exclusionClauses = append(exclusionClauses, fmt.Sprintf("Code != %s", code))
	}
	for _, pat := range f.ExcludeTenantPatterns {
		exclusionClauses = append(exclusionClauses, fmt.Sprintf("Code NOT LIKE %s", pat))
	}

	return buildFilterQuery(inclusionClauses, exclusionClauses)
}

// BuildVNamespaceFilterQuery converts the vnamespace-relevant parts of a ClaimWorkFilter.
func BuildVNamespaceFilterQuery(f models.ClaimWorkFilter) ClaimWorkFilterQuery {
	var inclusionClauses []string
	for _, ns := range f.VNamespaces {
		inclusionClauses = append(inclusionClauses, fmt.Sprintf("Name = %s", ns))
	}
	for _, pat := range f.VNamespacePatterns {
		inclusionClauses = append(inclusionClauses, fmt.Sprintf("Name LIKE %s", pat))
	}

	var exclusionClauses []string
	for _, ns := range f.ExcludeVNamespaces {
		exclusionClauses = append(exclusionClauses, fmt.Sprintf("Name != %s", ns))
	}
	for _, pat := range f.ExcludeVNamespacePatterns {
		exclusionClauses = append(exclusionClauses, fmt.Sprintf("Name NOT LIKE %s", pat))
	}

	return buildFilterQuery(inclusionClauses, exclusionClauses)
}

// BuildQueueFilterQuery converts the queue-relevant parts of a ClaimWorkFilter.
// It also enforces MessagesCount > 0 at the DB level to skip empty queues.
func BuildQueueFilterQuery(f models.ClaimWorkFilter, vNamespace string) ClaimWorkFilterQuery {
	var inclusionClauses []string
	for _, code := range f.QueueCodes {
		inclusionClauses = append(inclusionClauses, fmt.Sprintf("Code = %s", code))
	}
	for _, pat := range f.QueuePatterns {
		inclusionClauses = append(inclusionClauses, fmt.Sprintf("Code LIKE %s", pat))
	}

	var exclusionClauses []string
	for _, code := range f.ExcludeQueueCodes {
		exclusionClauses = append(exclusionClauses, fmt.Sprintf("Code != %s", code))
	}
	for _, pat := range f.ExcludeQueuePatterns {
		exclusionClauses = append(exclusionClauses, fmt.Sprintf("Code NOT LIKE %s", pat))
	}

	fq := buildFilterQuery(inclusionClauses, exclusionClauses)

	// Always restrict to queues that have pending messages
	var extraClauses []string
	extraClauses = append(extraClauses, "MessagesCount > 0")
	if vNamespace != "" {
		extraClauses = append(extraClauses, fmt.Sprintf("VNamespace = %s", vNamespace))
	}

	if fq.DBQuery == "ID != 0" {
		fq.DBQuery = strings.Join(extraClauses, " & ")
	} else {
		fq.DBQuery = fq.DBQuery + " & " + strings.Join(extraClauses, " & ")
	}

	return fq
}

// buildFilterQuery is the generic query assembler shared by the three entity builders above.
// inclusionClauses – OR-combined; if empty, all records are included.
// exclusionClauses – AND-combined; always appended (includes both != and NOT LIKE clauses).
func buildFilterQuery(inclusionClauses, exclusionClauses []string) ClaimWorkFilterQuery {
	var parts []string

	// Inclusion block (OR)
	if len(inclusionClauses) > 0 {
		if len(inclusionClauses) == 1 {
			parts = append(parts, inclusionClauses[0])
		} else {
			parts = append(parts, "("+strings.Join(inclusionClauses, " | ")+")")
		}
	}

	// Exclusion block (AND)
	for _, exc := range exclusionClauses {
		parts = append(parts, exc)
	}

	var query string
	if len(parts) == 0 {
		query = "ID != 0" // match-all workaround
	} else {
		query = strings.Join(parts, " & ")
	}

	return ClaimWorkFilterQuery{
		DBQuery: query,
	}
}
