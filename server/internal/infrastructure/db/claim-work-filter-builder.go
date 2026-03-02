package db

import (
	"fmt"
	"path"
	"strings"

	"deadalus-orch/shared/models"
)

// ClaimWorkFilterQuery is the result of translating a ClaimWorkFilter into DB-layer constructs.
// The DB query string already encodes the inclusion and exact-exclusion rules.
// The ExcludePatterns slice must be applied as an in-memory post-filter by the caller.
type ClaimWorkFilterQuery struct {
	// DBQuery is the filter string ready to be passed to Repository.Find.
	DBQuery string
	// ExcludePatterns are glob patterns that could not be expressed as DB queries (no NOT LIKE support).
	// The caller should drop any record whose relevant field matches one of these globs.
	ExcludePatterns []string
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
	// Exact-match exclusions can be expressed at DB level with !=
	for _, code := range f.ExcludeTenantCodes {
		exclusionClauses = append(exclusionClauses, fmt.Sprintf("Code != %s", code))
	}

	return buildFilterQuery(inclusionClauses, exclusionClauses, f.ExcludeTenantPatterns)
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

	return buildFilterQuery(inclusionClauses, exclusionClauses, f.ExcludeVNamespacePatterns)
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

	fq := buildFilterQuery(inclusionClauses, exclusionClauses, f.ExcludeQueuePatterns)

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
// exclusionClauses – AND-combined; always appended.
// excludePatterns  – returned as-is for in-memory post-filtering (no NOT LIKE in the DSL).
func buildFilterQuery(inclusionClauses, exclusionClauses, excludePatterns []string) ClaimWorkFilterQuery {
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
		DBQuery:         query,
		ExcludePatterns: excludePatterns,
	}
}

// MatchesExcludePatterns returns true when value matches any of the glob patterns in the slice.
// Uses path.Match semantics (* ? [range]).
func MatchesExcludePatterns(value string, patterns []string) bool {
	for _, pat := range patterns {
		if ok, _ := path.Match(pat, value); ok {
			return true
		}
	}
	return false
}
