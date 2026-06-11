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

	// Strictly require HasMessages == true to only iterate active tenants
	exclusionClauses = append(exclusionClauses, "HasMessages = true")

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
// Now it also applies VNamespace filters directly on the Queue since Queue has a VNamespace property.
func BuildQueueFilterQuery(f models.ClaimWorkFilter) ClaimWorkFilterQuery {
	// 1. Queue Code inclusions
	var qInclusionClauses []string
	for _, code := range f.QueueCodes {
		qInclusionClauses = append(qInclusionClauses, fmt.Sprintf("Code = %s", code))
	}
	for _, pat := range f.QueuePatterns {
		qInclusionClauses = append(qInclusionClauses, fmt.Sprintf("Code LIKE %s", pat))
	}

	// 2. VNamespace inclusions
	var vnsInclusionClauses []string
	for _, ns := range f.VNamespaces {
		vnsInclusionClauses = append(vnsInclusionClauses, fmt.Sprintf("VNamespace = %s", ns))
	}
	for _, pat := range f.VNamespacePatterns {
		vnsInclusionClauses = append(vnsInclusionClauses, fmt.Sprintf("VNamespace LIKE %s", pat))
	}

	// 3. Exclusions (Queue and VNamespace)
	var exclusionClauses []string
	for _, code := range f.ExcludeQueueCodes {
		exclusionClauses = append(exclusionClauses, fmt.Sprintf("Code != %s", code))
	}
	for _, pat := range f.ExcludeQueuePatterns {
		exclusionClauses = append(exclusionClauses, fmt.Sprintf("Code NOT LIKE %s", pat))
	}
	for _, ns := range f.ExcludeVNamespaces {
		exclusionClauses = append(exclusionClauses, fmt.Sprintf("VNamespace != %s", ns))
	}
	for _, pat := range f.ExcludeVNamespacePatterns {
		exclusionClauses = append(exclusionClauses, fmt.Sprintf("VNamespace NOT LIKE %s", pat))
	}

	// Build the final query parts
	var parts []string

	// Queue Inclusions (OR)
	if len(qInclusionClauses) > 0 {
		if len(qInclusionClauses) == 1 {
			parts = append(parts, qInclusionClauses[0])
		} else {
			parts = append(parts, "("+strings.Join(qInclusionClauses, " | ")+")")
		}
	}

	// VNamespace Inclusions (OR)
	if len(vnsInclusionClauses) > 0 {
		if len(vnsInclusionClauses) == 1 {
			parts = append(parts, vnsInclusionClauses[0])
		} else {
			parts = append(parts, "("+strings.Join(vnsInclusionClauses, " | ")+")")
		}
	}

	// Exclusions (AND)
	for _, exc := range exclusionClauses {
		parts = append(parts, exc)
	}

	// Always restrict to queues that have pending messages
	parts = append(parts, "MessagesCount > 0")

	var query string
	if len(parts) == 0 {
		query = "ID != 0" // match-all workaround, though it will never hit this because of MessagesCount
	} else {
		query = strings.Join(parts, " & ")
	}

	return ClaimWorkFilterQuery{
		DBQuery: query,
	}
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
