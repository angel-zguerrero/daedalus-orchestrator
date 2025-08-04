// Package db provides a generic repository implementation for interacting with a key-value store.
// It supports operations like Create, Update, Delete, Find, and FindByField.
// The repository also includes logic for tokenizing and parsing filter expressions.
package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// IDGeneratorFactory defines the interface for generating unique IDs.
type IDGeneratorFactory interface {
	// GenerateID creates a new unique ID.
	GenerateID() string
}

// ORMEntity defines the interface that entities managed by the repository must implement.
type ORMEntity interface {
	// TableName returns the name of the table corresponding to the entity.
	TableName() string
}

// FieldDefinition describes the properties of a field in an entity.
type FieldDefinition struct {
	// Name is the name of the field.
	Name string
	// Type is the data type of the field (e.g., "string", "int").
	Type string
	// Unique indicates whether the field must have unique values across all entities.
	Unique bool
	// Primary indicates whether the field is the primary key for the entity.
	Primary bool
	// MaxLength specifies the maximum length for string fields. It's nil if not applicable.
	MaxLength *int

	TTL bool

	// IgnoreIsTrueFieldName stores the name of the boolean field that, if true, bypasses the uniqueness check.
	IgnoreIsTrueFieldName string
	// HasConditionalUniqueness indicates if the 'ignore-is-true' tag is used for this field.
	HasConditionalUniqueness bool

	// UniqueCompoundIndex indicates the index of this field in a compound uniqueness constraint.
	// If this field is part of a compound uniqueness constraint, this value is >= 0.
	// If this field is not part of a compound uniqueness constraint, this value is -1.
	UniqueCompoundIndex int
	// IsUniqueCompound indicates whether this field is part of a compound uniqueness constraint.
	IsUniqueCompound bool
}

// TableDefinition describes the schema of a table in the key-value store.
type TableDefinition struct {
	// ColumnFamily is the name of the column family where the table data is stored.
	ColumnFamily string
	// ColumnFamilySector is the name of the column family sector where the table data is stored.
	ColumnFamilySector string
	// Schema is the namespace or schema name for the table.
	Schema string
	// Name is the name of the table.
	Name string
	// Fields is a map of field names to their definitions.
	Fields map[string]FieldDefinition
	// UniqueCompoundGroups maps compound index numbers to the field names that are part of that compound constraint.
	// For example, if fields Name and VNamespace both have unique-compound:0, then UniqueCompoundGroups[0] = ["Name", "VNamespace"]
	UniqueCompoundGroups map[int][]string
}

// Repository provides a generic implementation for interacting with a key-value store.
// T is the type of the entity being managed, and it must implement the ORMEntity interface.
type Repository[T ORMEntity] struct {
	definition         *TableDefinition
	kvStore            KVStore
	idGeneratorFactory IDGeneratorFactory
}

// FindResult represents the result of a Find operation.
// T is the type of the entities returned.
type FindResult[T any] struct {
	// Entities is a slice of entities that match the filter criteria.
	Entities []T
	// Cursor is a token that can be used to retrieve the next page of results.
	// It's an empty string if there are no more results.
	Cursor string
}
type tokenType int

const (
	tokUnknown tokenType = iota
	tokLParen
	tokRParen
	tokOperator
	tokCondition
)

type token struct {
	typ tokenType
	val string
}

var operatorPrecedence = map[string]int{
	"|": 1,
	"&": 2,
}

type exprNode struct {
	op      string // "&" or "|" or "COND"
	left    *exprNode
	right   *exprNode
	condStr string
}

func tokenize(input string) ([]token, error) {
	tokens := []token{}
	buffer := ""
	depth := 0
	flush := func() {
		if strings.TrimSpace(buffer) != "" {
			tokens = append(tokens, token{typ: tokCondition, val: strings.TrimSpace(buffer)})
			buffer = ""
		}
	}
	for i := 0; i < len(input); i++ {
		c := input[i]
		switch c {
		case '(':
			flush()
			tokens = append(tokens, token{typ: tokLParen, val: "("})
			depth++
		case ')':
			flush()
			tokens = append(tokens, token{typ: tokRParen, val: ")"})
			depth--
		case '&', '|':
			flush()
			tokens = append(tokens, token{typ: tokOperator, val: string(c)})
		default:
			buffer += string(c)
		}
	}
	flush()
	if depth != 0 {
		return nil, errors.New("mismatched parentheses")
	}
	return tokens, nil
}
func parse(tokens []token) (*exprNode, error) {
	output := []*exprNode{}
	ops := []token{}
	popOp := func() error {
		if len(ops) == 0 {
			return errors.New("invalid expression")
		}
		op := ops[len(ops)-1]
		ops = ops[:len(ops)-1]
		if len(output) < 2 {
			return errors.New("invalid expression tree")
		}
		right := output[len(output)-1]
		left := output[len(output)-2]
		output = output[:len(output)-2]
		n := &exprNode{op: op.val, left: left, right: right}
		output = append(output, n)
		return nil
	}
	for _, tok := range tokens {
		switch tok.typ {
		case tokCondition:
			output = append(output, &exprNode{op: "COND", condStr: tok.val})
		case tokOperator:
			for len(ops) > 0 && ops[len(ops)-1].typ == tokOperator &&
				operatorPrecedence[ops[len(ops)-1].val] >= operatorPrecedence[tok.val] {
				if err := popOp(); err != nil {
					return nil, err
				}
			}
			ops = append(ops, tok)
		case tokLParen:
			ops = append(ops, tok)
		case tokRParen:
			for len(ops) > 0 && ops[len(ops)-1].typ != tokLParen {
				if err := popOp(); err != nil {
					return nil, err
				}
			}
			if len(ops) == 0 || ops[len(ops)-1].typ != tokLParen {
				return nil, errors.New("mismatched parentheses")
			}
			ops = ops[:len(ops)-1] // pop (
		default:
			return nil, fmt.Errorf("unexpected token: %v", tok)
		}
	}
	for len(ops) > 0 {
		if err := popOp(); err != nil {
			return nil, err
		}
	}
	if len(output) != 1 {
		return nil, errors.New("expression reduced to multiple roots")
	}
	return output[0], nil
}
func (r *Repository[T]) evalCondition(condStr string, limit int, now time.Time) (map[string]bool, error) {
	// Regular field condition parsing
	conditionRegex := regexp.MustCompile(`(?i)^([\w.]+)\s*(=|!=|<=|>=|<|>|LIKE|BETWEEN)\s*(.+)$`)
	parts := conditionRegex.FindStringSubmatch(strings.TrimSpace(condStr))
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid condition: %s", condStr)
	}

	field := strings.TrimSpace(parts[1])
	def := r.definition.Fields[field]
	if def == (FieldDefinition{}) {
		return nil, fmt.Errorf("Unknown field %s", field)
	}
	if def.TTL {
		return nil, fmt.Errorf("TTL columns are not supported in query operations")
	}
	operator := strings.ToUpper(strings.TrimSpace(parts[2]))
	value := strings.TrimSpace(strings.Trim(parts[3], "'"))

	allIDs := make(map[string]bool)
	prefix := fmt.Sprintf("%s:%s:idx:%s:", r.definition.Schema, r.definition.Name, field)
	cursorInner := ""

	switch operator {
	case "=":
		pattern := prefix + value + ":*"
		for {
			items, next, err := r.kvStore.SearchByPatternPaginatedKV(r.definition.ColumnFamily, r.definition.ColumnFamilySector, pattern, cursorInner, limit, now)
			if err != nil {
				return nil, err
			}
			for _, item := range items {
				allIDs[string(item.Value)] = true
			}
			if next == "" {
				break
			}
			cursorInner = next
		}
	case "LIKE":
		likeRegex := strings.ReplaceAll(value, "*", ".*")
		regex, err := regexp.Compile("^" + likeRegex + "$")
		if err != nil {
			return nil, fmt.Errorf("invalid LIKE pattern: %s", value)
		}
		for {
			items, next, err := r.kvStore.SearchByPatternPaginatedKV(r.definition.ColumnFamily, r.definition.ColumnFamilySector, prefix+"*", cursorInner, limit, now)
			if err != nil {
				return nil, err
			}
			for _, item := range items {
				parts := strings.Split(string(item.Key), ":")
				if len(parts) < 5 {
					continue
				}
				indexedVal := parts[4]
				if regex.MatchString(indexedVal) {
					allIDs[string(item.Value)] = true
				}
			}
			if next == "" {
				break
			}
			cursorInner = next
		}
	case "<", "<=", ">", ">=", "!=", "BETWEEN":
		for {
			items, next, err := r.kvStore.SearchByPatternPaginatedKV(r.definition.ColumnFamily, r.definition.ColumnFamilySector, prefix+"*", cursorInner, limit, now)
			if err != nil {
				return nil, err
			}
			for _, item := range items {
				parts := strings.Split(string(item.Key), ":")
				if len(parts) < 5 {
					continue
				}
				indexedVal := parts[4]
				include := false
				switch operator {
				case "<":
					include = indexedVal < value
				case "<=":
					include = indexedVal <= value
				case ">":
					include = indexedVal > value
				case ">=":
					include = indexedVal >= value
				case "!=":
					include = indexedVal != value
				case "BETWEEN":
					bounds := strings.Split(value, "AND")
					if len(bounds) == 2 {
						lo := strings.TrimSpace(bounds[0])
						hi := strings.TrimSpace(bounds[1])
						include = indexedVal >= lo && indexedVal <= hi
					}
				}
				if include {
					allIDs[string(item.Value)] = true
				}
			}
			if next == "" {
				break
			}
			cursorInner = next
		}
	default:
		return nil, fmt.Errorf("unsupported operator: %s", operator)
	}

	return allIDs, nil
}
func (r *Repository[T]) evalExpr(node *exprNode, limit int, now time.Time) (map[string]bool, error) {
	if node.op == "COND" {
		return r.evalCondition(node.condStr, limit, now)
	}
	left, err := r.evalExpr(node.left, limit, now)
	if err != nil {
		return nil, err
	}
	right, err := r.evalExpr(node.right, limit, now)
	if err != nil {
		return nil, err
	}
	res := map[string]bool{}
	if node.op == "&" {
		for k := range left {
			if right[k] {
				res[k] = true
			}
		}
	} else if node.op == "|" {
		for k := range left {
			res[k] = true
		}
		for k := range right {
			res[k] = true
		}
	} else {
		return nil, fmt.Errorf("unsupported operator: %s", node.op)
	}
	return res, nil
}

// Find retrieves entities from the repository based on a filter expression.
// It supports pagination using a cursor.
//
// The filter syntax is as follows:
//   - Conditions are specified as `field operator value`.
//   - Supported operators: `=`, `!=`, `<`, `<=`, `>`, `>=`, `LIKE`, `BETWEEN`.
//   - String values in conditions should be enclosed in single quotes (e.g., `name = 'John Doe'`).
//   - Conditions can be combined using `&` (AND) and `|` (OR) operators.
//   - Parentheses `()` can be used to group conditions.
//   - Example: `(field1 = 'value1' & field2 LIKE 'pattern*') | field3 BETWEEN 'start' AND 'end'`
//
// Parameters:
//   - filter: The filter string to apply.
//   - limit: The maximum number of entities to return.
//   - cursor: A cursor token for pagination. Pass an empty string for the first page.
//
// Returns:
//   - A pointer to a FindResult struct containing the matched entities and the next cursor.
//   - An error if the operation fails.
//
// tryCompoundQuery attempts to detect and optimize queries that match compound unique constraints.
// It analyzes the filter string to find patterns like "field1='value1' & field2='value2'" that
// correspond to compound unique constraints.
//
// Returns:
//   - entity: The found entity if the query matches a compound constraint
//   - isCompound: true if this was identified as a compound query
//   - error: any error that occurred during processing
func (r *Repository[T]) tryCompoundQuery(filter string, now time.Time) (*T, bool, error) {
	// Parse the filter to extract field=value conditions connected by &
	fieldValues, isCompoundPattern := r.parseCompoundPattern(filter)
	if !isCompoundPattern {
		return nil, false, nil
	}

	// Check if the extracted fields form a compound unique constraint
	compoundIndex, isCompoundConstraint := r.isCompoundUniqueConstraint(fieldValues)
	if !isCompoundConstraint {
		return nil, false, nil
	}

	// Perform compound field lookup directly
	expectedFieldNames := r.definition.UniqueCompoundGroups[compoundIndex]

	// Build composite key parts using the same ordering as used during creation
	var compositeKeyParts []string
	for _, fieldName := range expectedFieldNames { // expectedFieldNames is already sorted
		fieldValue := fieldValues[fieldName]
		compositeKeyParts = append(compositeKeyParts, fmt.Sprintf("%s:%s", fieldName, fieldValue))
	}

	// Create composite value by joining all field:value pairs
	compositeValue := strings.Join(compositeKeyParts, "|")

	// Create compound unique index key
	compoundIdxKey := fmt.Sprintf("%s:%s:idx-uc:%d:%s", r.definition.Schema, r.definition.Name, compoundIndex, compositeValue)

	// Get the ID from the compound index
	idBytes, err := r.kvStore.Get(r.definition.ColumnFamily, r.definition.ColumnFamilySector, compoundIdxKey, now)
	if err != nil {
		return nil, true, fmt.Errorf("error searching compound index: %w", err)
	}

	if idBytes == nil {
		return nil, true, nil // Not found, but it was a compound query
	}

	// Get the actual entity data
	dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, string(idBytes))
	dataBytes, err := r.kvStore.Get(r.definition.ColumnFamily, r.definition.ColumnFamilySector, dataKey, now)
	if err != nil {
		return nil, true, fmt.Errorf("error getting entity data: %w", err)
	}

	if dataBytes == nil {
		return nil, true, nil // Data not found (inconsistent state), but it was a compound query
	}

	var result T
	err = json.Unmarshal(dataBytes, &result)
	if err != nil {
		return nil, true, fmt.Errorf("error unmarshaling entity: %w", err)
	}

	return &result, true, nil
}

// parseCompoundPattern analyzes a filter string to detect patterns that could be compound queries.
// It looks for patterns like "field1='value1' & field2='value2'" (with any number of fields).
// Returns the extracted field-value pairs and whether this looks like a compound pattern.
func (r *Repository[T]) parseCompoundPattern(filter string) (map[string]string, bool) {
	// Split by & to get individual conditions
	conditions := strings.Split(filter, "&")
	if len(conditions) < 2 {
		return nil, false // Need at least 2 conditions for compound
	}

	fieldValues := make(map[string]string)
	equalityRegex := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_.]*)\s*=\s*'([^']*)'\s*$`)

	for _, condition := range conditions {
		condition = strings.TrimSpace(condition)

		// Check if this is a simple equality condition
		matches := equalityRegex.FindStringSubmatch(condition)
		if len(matches) != 3 {
			return nil, false // Not a simple equality, can't optimize
		}

		fieldName := matches[1]
		fieldValue := matches[2]

		// Verify the field exists in our table definition
		if _, exists := r.definition.Fields[fieldName]; !exists {
			return nil, false // Unknown field
		}

		fieldValues[fieldName] = fieldValue
	}

	return fieldValues, len(fieldValues) >= 2
}

// isCompoundUniqueConstraint checks if the provided field values match a compound unique constraint
// Returns the compound index and true if it matches, or -1 and false if it doesn't match
func (r *Repository[T]) isCompoundUniqueConstraint(fieldValues map[string]string) (int, bool) {
	// Extract field names from the provided map
	providedFields := make(map[string]bool)
	for fieldName := range fieldValues {
		providedFields[fieldName] = true
	}

	// Check each compound unique group
	for compoundIndex, expectedFields := range r.definition.UniqueCompoundGroups {
		// Check if the number of fields matches
		if len(expectedFields) != len(providedFields) {
			continue
		}

		// Check if all expected fields are present
		allFieldsMatch := true
		for _, expectedField := range expectedFields {
			if !providedFields[expectedField] {
				allFieldsMatch = false
				break
			}
		}

		if allFieldsMatch {
			return compoundIndex, true
		}
	}

	return -1, false
}

func (r *Repository[T]) Find(filter string, limit int, cursor string, now time.Time) (*FindResult[T], error) {
	// First, try to optimize the query by detecting compound unique constraints
	compoundResult, isCompoundQuery, err := r.tryCompoundQuery(filter, now)
	if err != nil {
		return nil, err
	}
	if isCompoundQuery {
		// Convert single entity result to FindResult format
		results := []T{}
		if compoundResult != nil {
			results = append(results, *compoundResult)
		}

		return &FindResult[T]{
			Entities: results,
			Cursor:   "", // No pagination needed for single result
		}, nil
	}

	// If not a compound query, process normally
	tokens, err := tokenize(filter)
	if err != nil {
		return nil, err
	}
	tree, err := parse(tokens)
	if err != nil {
		return nil, err
	}
	idMap, err := r.evalExpr(tree, limit, now)
	if err != nil {
		return nil, err
	}
	var matchedIDs []string
	for id := range idMap {
		matchedIDs = append(matchedIDs, id)
	}
	sort.Strings(matchedIDs)
	start := 0
	if cursor != "" {
		for i, id := range matchedIDs {
			if id == cursor {
				start = i + 1
				break
			}
		}
	}
	end := start + limit
	if end > len(matchedIDs) {
		end = len(matchedIDs)
	}
	selectedIDs := matchedIDs[start:end]
	results := []T{}
	for _, id := range selectedIDs {
		dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
		dataBytes, err := r.kvStore.Get(r.definition.ColumnFamily, r.definition.ColumnFamilySector, dataKey, now)
		if err != nil {
			return nil, err
		}
		var result T
		err = json.Unmarshal(dataBytes, &result)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	var nextCursor string
	if end < len(matchedIDs) {
		nextCursor = matchedIDs[end-1]
	}
	return &FindResult[T]{
		Entities: results,
		Cursor:   nextCursor,
	}, nil
}

// FindByField retrieves a single entity by the value of a specific field.
// This method is typically used for fields that are indexed or have unique constraints.
//
// Parameters:
//   - field: The name of the field to search by.
//   - value: The value of the field to match.
//
// Returns:
//   - A pointer to the matched entity, or nil if no entity is found.
//   - An error if the operation fails.
func (r *Repository[T]) FindByField(field string, value string, now time.Time) (*T, error) {
	def := r.definition.Fields[field]
	if def == (FieldDefinition{}) {
		return nil, fmt.Errorf("Unknown field %s", field)
	}
	if def.TTL {
		return nil, fmt.Errorf("TTL columns are not supported in query operations")
	}
	var dataKey string
	if def.Unique {
		searchKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, field, value)
		idBytes, err := r.kvStore.Get(r.definition.ColumnFamily, r.definition.ColumnFamilySector, searchKey, now)
		if err != nil || idBytes == nil || len(idBytes) == 0 {
			return nil, err
		}

		dataKey = fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, string(idBytes))
	} else if def.Primary {
		dataKey = fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, value)
	} else {
		searchKey := fmt.Sprintf("%s:%s:idx:%s:%s:*", r.definition.Schema, r.definition.Name, field, value)
		idBytes, _, err := r.kvStore.SearchByPatternPaginatedKV(r.definition.ColumnFamily, r.definition.ColumnFamilySector, searchKey, "", 1, now)
		if err != nil || idBytes == nil || len(idBytes) == 0 {
			return nil, err
		}

		dataKey = fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, string(idBytes[0].Value))
	}
	dataBytes, err := r.kvStore.Get(r.definition.ColumnFamily, r.definition.ColumnFamilySector, dataKey, now)
	if err != nil {
		return nil, err
	}

	if dataBytes == nil {
		return nil, nil
	}

	var result T
	err = json.Unmarshal(dataBytes, &result)
	return &result, err
}

// BulkCreate creates multiple entities in the repository in a single batch operation.
// It assigns a new unique ID to each entity and sets the primary key field.
// It also validates unique constraints across the batch and against existing data.
//
// Parameters:
//   - entities: A slice of pointers to the entities to create.
//
// Returns:
//   - A slice of strings containing the IDs of the newly created entities.
//   - An error if the operation fails (e.g., due to a unique constraint violation).
func (r *Repository[T]) BulkCreate(entities []*T, now time.Time) ([]string, error) {
	var ids []string
	batch := NewWriteBatch()

	type uniqueCheck struct {
		Key       string
		FieldName string
		Value     string
	}

	type uniqueCompoundCheck struct {
		Key            string
		CompoundIndex  int
		FieldValues    map[string]string // fieldName -> value
		CompositeValue string            // concatenated field values for key generation
	}
	var uniqueChecks []uniqueCheck
	var uniqueCompoundChecks []uniqueCompoundCheck

	// Map para detectar duplicados en el batch, clave = campo+valor
	uniqueInBatch := make(map[string]struct{})
	// Map to detect duplicate compound uniqueness in the batch, key = compoundIndex:compositeValue
	uniqueCompoundInBatch := make(map[string]struct{})
	// Map to detect duplicate primary keys in the batch
	primaryKeysInBatch := make(map[string]struct{})

	for _, entity := range entities {
		generatedID := r.idGeneratorFactory.GenerateID()
		var currentEntityIDValue string

		val := reflect.ValueOf(entity)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// Set primary key field
		// The primary key field name is validated to be "ID" and top-level by NewRepository.
		pkField, err := getNestedFieldValue(val, "ID")
		if err != nil {
			return nil, fmt.Errorf("error getting primary key field 'ID' for entity: %w", err)
		}
		if !pkField.IsValid() || !pkField.CanSet() || pkField.Kind() != reflect.String {
			return nil, fmt.Errorf("primary key field 'ID' is not a settable string field or is invalid")
		}

		if generatedID != "" { // ID was generated
			pkField.SetString(generatedID)
			currentEntityIDValue = generatedID
			ids = append(ids, generatedID)
		} else { // ID is provided in the entity
			entityProvidedID := pkField.String()
			if entityProvidedID == "" {
				return nil, fmt.Errorf("primary key field 'ID' cannot be empty when not generated")
			}
			currentEntityIDValue = entityProvidedID
			ids = append(ids, entityProvidedID)
		}

		// Check for duplicate primary key in the batch
		if _, exists := primaryKeysInBatch[currentEntityIDValue]; exists {
			return nil, fmt.Errorf("duplicate primary key in input batch: ID = %s", currentEntityIDValue)
		}
		primaryKeysInBatch[currentEntityIDValue] = struct{}{}

		// Check for duplicate primary key in the database
		// This check must be done before the unique field checks for other fields
		// as it's a more fundamental constraint.
		pkDataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, currentEntityIDValue)
		exists, err := r.kvStore.Exists(r.definition.ColumnFamily, r.definition.ColumnFamilySector, pkDataKey, now)
		if err != nil {
			return nil, fmt.Errorf("error checking existence for primary key %s: %w", currentEntityIDValue, err)
		}
		if exists {
			return nil, fmt.Errorf("duplicate primary key: ID = %s already exists", currentEntityIDValue)
		}

		for _, def := range r.definition.Fields {
			if def.Unique { // This handles non-primary unique fields
				shouldSkipUniqueness := false
				if def.HasConditionalUniqueness {
					boolFieldVal, err := getNestedFieldValue(val, def.IgnoreIsTrueFieldName)
					if err != nil {
						return nil, fmt.Errorf("error getting conditional uniqueness flag field '%s' for entity: %w", def.IgnoreIsTrueFieldName, err)
					}
					if boolFieldVal.Kind() == reflect.Bool && boolFieldVal.Bool() {
						shouldSkipUniqueness = true
					}
				}

				if shouldSkipUniqueness {
					continue // Skip uniqueness operations for this field on this entity
				}

				fieldVal, err := getNestedFieldValue(val, def.Name)
				if err != nil {
					return nil, fmt.Errorf("error getting unique field '%s' for entity: %w", def.Name, err)
				}
				if !fieldVal.IsValid() {
					return nil, fmt.Errorf("unique field '%s' is invalid for entity", def.Name)
				}
				fieldValue := fmt.Sprintf("%v", fieldVal.Interface())
				uniqueIdxKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, def.Name, fieldValue)
				uniqueChecks = append(uniqueChecks, uniqueCheck{
					Key:       uniqueIdxKey,
					FieldName: def.Name,
					Value:     fieldValue,
				})

				// Validación de duplicados en el mismo batch
				batchKey := def.Name + ":" + fieldValue
				if _, exists := uniqueInBatch[batchKey]; exists {
					return nil, fmt.Errorf("duplicate unique field in input batch: %s = %s", def.Name, fieldValue)
				}
				uniqueInBatch[batchKey] = struct{}{}
			}
		}
	}

	// Check unique compound constraints
	for _, entity := range entities {
		val := reflect.ValueOf(entity)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// For each compound constraint group
		for compoundIndex, fieldNames := range r.definition.UniqueCompoundGroups {
			fieldValues := make(map[string]string)
			var compositeKeyParts []string

			// Get all field values for this compound constraint
			for _, fieldName := range fieldNames {
				fieldVal, err := getNestedFieldValue(reflect.ValueOf(entity), fieldName)
				if err != nil {
					return nil, fmt.Errorf("error getting compound field '%s' for entity: %w", fieldName, err)
				}
				if !fieldVal.IsValid() {
					return nil, fmt.Errorf("compound field '%s' is invalid for entity", fieldName)
				}
				fieldValue := fmt.Sprintf("%v", fieldVal.Interface())
				fieldValues[fieldName] = fieldValue
				compositeKeyParts = append(compositeKeyParts, fmt.Sprintf("%s:%s", fieldName, fieldValue))
			}

			// Create composite value by joining all field:value pairs
			compositeValue := strings.Join(compositeKeyParts, "|")

			// Check for duplicates in batch
			batchKey := fmt.Sprintf("%d:%s", compoundIndex, compositeValue)
			if _, exists := uniqueCompoundInBatch[batchKey]; exists {
				return nil, fmt.Errorf("duplicate unique compound constraint in input batch: compound-index=%d, fields=%v", compoundIndex, fieldValues)
			}
			uniqueCompoundInBatch[batchKey] = struct{}{}

			// Create key for database check
			compoundIdxKey := fmt.Sprintf("%s:%s:idx-uc:%d:%s", r.definition.Schema, r.definition.Name, compoundIndex, compositeValue)
			uniqueCompoundChecks = append(uniqueCompoundChecks, uniqueCompoundCheck{
				Key:            compoundIdxKey,
				CompoundIndex:  compoundIndex,
				FieldValues:    fieldValues,
				CompositeValue: compositeValue,
			})
		}
	}

	// Validar duplicados en la base
	// This check inherently respects shouldSkipUniqueness because items are not added to uniqueChecks
	for _, check := range uniqueChecks {
		exists, err := r.kvStore.Exists(r.definition.ColumnFamily, r.definition.ColumnFamilySector, check.Key, now)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("duplicate unique field: %s = %s", check.FieldName, check.Value)
		}
	}

	// Validate unique compound constraints in database
	for _, check := range uniqueCompoundChecks {
		exists, err := r.kvStore.Exists(r.definition.ColumnFamily, r.definition.ColumnFamilySector, check.Key, now)
		if err != nil {
			return nil, fmt.Errorf("error checking compound uniqueness constraint: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("duplicate unique compound constraint: compound-index=%d, fields=%v", check.CompoundIndex, check.FieldValues)
		}
	}

	// Insertar datos y sus índices
	for i, entity := range entities {
		entityPtrVal := reflect.ValueOf(entity) // entity is *T

		id := ids[i]

		// Pass entityPtrVal (which is *T) to checkForTTL
		hasTTL, ttl, err := r.checkForTTL(entityPtrVal)
		if err != nil {
			return nil, fmt.Errorf("error checking TTL for entity with ID '%s': %w", id, err)
		}

		for _, def := range r.definition.Fields {
			if def.TTL { // TTL field itself is not indexed like other fields.
				continue
			}
			// Pass entityPtrVal (which is *T) to getNestedFieldValue
			fieldVal, err := getNestedFieldValue(entityPtrVal, def.Name)
			if err != nil {
				return nil, fmt.Errorf("error getting field '%s' for entity with ID '%s': %w", def.Name, id, err)
			}
			if !fieldVal.IsValid() {
				return nil, fmt.Errorf("field '%s' is invalid for entity with ID '%s'", def.Name, id)
			}
			fieldValue := fmt.Sprintf("%v", fieldVal.Interface())
			idxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, def.Name, fieldValue, id)
			if hasTTL {
				batch.PutTTl(r.definition.ColumnFamily, r.definition.ColumnFamilySector, idxKey, []byte(id), ttl, now)
			} else {
				batch.Put(r.definition.ColumnFamily, r.definition.ColumnFamilySector, idxKey, []byte(id), now)
			}

			if def.Unique {
				shouldSkipUniqueness := false
				if def.HasConditionalUniqueness {
					boolFieldVal, err := getNestedFieldValue(entityPtrVal, def.IgnoreIsTrueFieldName)
					if err != nil {
						return nil, fmt.Errorf("error getting conditional uniqueness flag field '%s' for entity with ID '%s': %w", def.IgnoreIsTrueFieldName, id, err)
					}
					if boolFieldVal.Kind() == reflect.Bool && boolFieldVal.Bool() {
						shouldSkipUniqueness = true
					}
				}

				if shouldSkipUniqueness {
					// Already processed other non-unique indexing for this field if any,
					// so just continue the inner loop over fields to skip unique indexing.
					// Note: If a field was ONLY unique and had no other indexing logic,
					// this continue would skip to the next field def.
					// The current structure has general indexing first, then unique indexing.
					// This `continue` will effectively skip the unique indexing part below for this field.
					// To be absolutely clear, we could place the unique indexing in an else block
					// of `if shouldSkipUniqueness`, but the current flow with continue is fine.
				} else {
					uniqueIdxKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, def.Name, fieldValue)
					if hasTTL {
						batch.PutTTl(r.definition.ColumnFamily, r.definition.ColumnFamilySector, uniqueIdxKey, []byte(id), ttl, now)
					} else {
						batch.Put(r.definition.ColumnFamily, r.definition.ColumnFamilySector, uniqueIdxKey, []byte(id), now)
					}
				}
			}
		}

		// Create compound unique indexes
		for compoundIndex, fieldNames := range r.definition.UniqueCompoundGroups {
			fieldValues := make(map[string]string)
			var compositeKeyParts []string

			// Get all field values for this compound constraint
			for _, fieldName := range fieldNames {
				fieldVal, err := getNestedFieldValue(entityPtrVal, fieldName)
				if err != nil {
					return nil, fmt.Errorf("error getting compound field '%s' for entity with ID '%s': %w", fieldName, id, err)
				}
				if !fieldVal.IsValid() {
					return nil, fmt.Errorf("compound field '%s' is invalid for entity with ID '%s'", fieldName, id)
				}
				fieldValue := fmt.Sprintf("%v", fieldVal.Interface())
				fieldValues[fieldName] = fieldValue
				compositeKeyParts = append(compositeKeyParts, fmt.Sprintf("%s:%s", fieldName, fieldValue))
			}

			// Create composite value by joining all field:value pairs
			compositeValue := strings.Join(compositeKeyParts, "|")

			// Create compound unique index key
			compoundIdxKey := fmt.Sprintf("%s:%s:idx-uc:%d:%s", r.definition.Schema, r.definition.Name, compoundIndex, compositeValue)
			if hasTTL {
				batch.PutTTl(r.definition.ColumnFamily, r.definition.ColumnFamilySector, compoundIdxKey, []byte(id), ttl, now)
			} else {
				batch.Put(r.definition.ColumnFamily, r.definition.ColumnFamilySector, compoundIdxKey, []byte(id), now)
			}
		}

		dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
		dataBytes, err := json.Marshal(entity)
		if err != nil {
			return nil, err
		}
		if hasTTL {
			batch.PutTTl(r.definition.ColumnFamily, r.definition.ColumnFamilySector, dataKey, dataBytes, ttl, now)
		} else {
			batch.Put(r.definition.ColumnFamily, r.definition.ColumnFamilySector, dataKey, dataBytes, now)
		}
	}

	if err := r.kvStore.Write(batch); err != nil {
		return nil, err
	}

	return ids, nil
}

// checkForTTL checks if the entity has a TTL field defined and retrieves its value.
// entityValue is the reflect.Value of the entity (can be pointer or struct).
func (r *Repository[T]) checkForTTL(entityValue reflect.Value) (bool, int, error) {
	var ttlFieldName string
	hasTTLDefinition := false
	for fName, fDef := range r.definition.Fields {
		if fDef.TTL {
			ttlFieldName = fName // This should be "TTL" as per NewRepository validation
			hasTTLDefinition = true
			break
		}
	}

	if !hasTTLDefinition {
		return false, 0, nil // No TTL field defined in schema
	}

	// getNestedFieldValue expects a struct or pointer to struct.
	// ttlFieldName is validated to be top-level "TTL".
	ttlVal, err := getNestedFieldValue(entityValue, ttlFieldName)
	if err != nil {
		return false, 0, fmt.Errorf("error getting TTL field '%s': %w", ttlFieldName, err)
	}

	if !ttlVal.IsValid() {
		// This could happen if the "TTL" field is somehow missing from the struct instance,
		// though schema validation in NewRepository should ensure it exists.
		// Or if entityValue was nil and not handled by getNestedFieldValue (it should error).
		return false, 0, fmt.Errorf("TTL field '%s' is invalid or not found on the entity instance", ttlFieldName)
	}

	var ttl int
	switch ttlVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ttl = int(ttlVal.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		ttl = int(ttlVal.Uint())
	default:
		// This should be caught by schema validation in NewRepository.
		return false, 0, fmt.Errorf("TTL field '%s' must be of integer type, but found %s", ttlFieldName, ttlVal.Kind())
	}

	return true, ttl, nil
}

// Create creates a single entity in the repository.
// It's a convenience wrapper around BulkCreate.
//
// Parameters:
//   - entity: A pointer to the entity to create.
//
// Returns:
//   - The ID of the newly created entity.
//   - An error if the operation fails.
func (r *Repository[T]) Create(entity *T, now time.Time) (string, error) {
	ids, err := r.BulkCreate([]*T{entity}, now)
	if err != nil {
		return "", err
	}
	return ids[0], nil
}

// BulkUpdate updates multiple entities in the repository in a single batch operation.
// It identifies entities by their primary key.
// For each entity, it compares the current values with the new values and updates only the changed fields.
// It handles updates to indexed and unique fields accordingly.
//
// Parameters:
//   - entities: A slice of pointers to the entities to update. The primary key field must be populated.
//
// Returns:
//   - A slice of booleans indicating whether each corresponding entity was updated.
//   - An error if the operation fails (e.g., due to a unique constraint violation).
func (r *Repository[T]) BulkUpdate(entities []*T, now time.Time) ([]bool, error) {
	// Determine the primary key field name (should be "ID" due to NewRepository validation)
	var primaryFieldName string
	for name, fDef := range r.definition.Fields {
		if fDef.Primary {
			primaryFieldName = name
			break
		}
	}
	if primaryFieldName == "" {
		return nil, fmt.Errorf("no primary key defined for the entity type")
	}

	results := make([]bool, len(entities))
	batch := NewWriteBatch()

	// For checking unique constraints within the batch
	uniqueValuesInBatch := make(map[string]map[string]string) // fieldName -> value -> entityID

	for i, entity := range entities {
		if entity == nil {
			results[i] = false
			continue
		}
		entityPtrVal := reflect.ValueOf(entity) // entity is *T

		idFieldVal, err := getNestedFieldValue(entityPtrVal, primaryFieldName)
		if err != nil {
			return nil, fmt.Errorf("error getting ID for entity for batch unique check: %w", err)
		}

		id := fmt.Sprintf("%v", idFieldVal.Interface())

		for _, def := range r.definition.Fields {
			if def.Primary || !def.Unique {
				continue
			}

			shouldSkipForBatchCheck := false
			if def.HasConditionalUniqueness {
				boolFieldVal, err := getNestedFieldValue(entityPtrVal, def.IgnoreIsTrueFieldName)
				if err != nil {
					return nil, fmt.Errorf("error getting conditional uniqueness flag '%s' for entity ID '%s' (batch check): %w", def.IgnoreIsTrueFieldName, id, err)
				}
				if boolFieldVal.Kind() == reflect.Bool && boolFieldVal.Bool() {
					shouldSkipForBatchCheck = true
				}
			}

			if shouldSkipForBatchCheck {
				continue
			}

			fieldVal, err := getNestedFieldValue(entityPtrVal, def.Name)
			if err != nil {
				return nil, fmt.Errorf("error getting unique field '%s' for entity ID '%s' (batch check): %w", def.Name, id, err)
			}
			if !fieldVal.IsValid() {
				// Assuming if a field is not valid/present, it cannot violate uniqueness for this check.
				// More robust error handling might be needed if fields are mandatory.
				continue
			}
			value := fmt.Sprintf("%v", fieldVal.Interface())

			if _, ok := uniqueValuesInBatch[def.Name]; !ok {
				uniqueValuesInBatch[def.Name] = make(map[string]string)
			}

			if existingID, exists := uniqueValuesInBatch[def.Name][value]; exists && existingID != id {
				return nil, fmt.Errorf("duplicate unique field in input batch: %s = %s", def.Name, value)
			}
			uniqueValuesInBatch[def.Name][value] = id
		}
	}

	for i, entity := range entities {
		if entity == nil {
			results[i] = false
			continue
		}
		entityPtrVal := reflect.ValueOf(entity) // entity is *T

		idFieldVal, err := getNestedFieldValue(entityPtrVal, primaryFieldName)
		if err != nil {
			return nil, fmt.Errorf("error getting ID for entity for update: %w", err)
		}
		id := fmt.Sprintf("%v", idFieldVal.Interface())

		// Fetch current entity from DB
		currentEntityStored, err := r.FindByField(primaryFieldName, id, now)
		if err != nil {
			return nil, fmt.Errorf("error fetching current entity with ID '%s' for update: %w", id, err)
		}
		if currentEntityStored == nil {
			results[i] = false // Entity not found, cannot update
			continue
		}

		changed := false
		// currentEntityStored is *T, get its Elem for getNestedFieldValue. Pass pointer to checkForTTL.
		currentEntityReflectVal := reflect.ValueOf(currentEntityStored)
		newEntityReflectVal := entityPtrVal // *T

		// Store compound field values before any changes for comparison
		compoundFieldChanges := make(map[int]struct {
			oldCompositeValue string
			newCompositeValue string
			changed           bool
		})

		// Pre-calculate compound constraint changes
		for compoundIndex, fieldNames := range r.definition.UniqueCompoundGroups {
			var oldCompositeKeyParts []string
			var newCompositeKeyParts []string
			compoundChanged := false
			allFieldsValid := true

			for _, fieldName := range fieldNames {
				currentFieldVal, err := getNestedFieldValue(currentEntityReflectVal, fieldName)
				if err != nil {
					fmt.Printf("Warning: could not get current compound field %s for updating compound index of entity ID %s: %v. Stale index might remain.\n", fieldName, id, err)
					allFieldsValid = false
					break
				}
				newFieldVal, err := getNestedFieldValue(newEntityReflectVal, fieldName)
				if err != nil {
					fmt.Printf("Warning: could not get new compound field %s for updating compound index of entity ID %s: %v. Stale index might remain.\n", fieldName, id, err)
					allFieldsValid = false
					break
				}
				if !currentFieldVal.IsValid() || !newFieldVal.IsValid() {
					fmt.Printf("Warning: compound field %s is invalid for updating compound index of entity ID %s. Stale index might remain.\n", fieldName, id)
					allFieldsValid = false
					break
				}

				oldValue := fmt.Sprintf("%v", currentFieldVal.Interface())
				newValue := fmt.Sprintf("%v", newFieldVal.Interface())

				oldCompositeKeyParts = append(oldCompositeKeyParts, fmt.Sprintf("%s:%s", fieldName, oldValue))
				newCompositeKeyParts = append(newCompositeKeyParts, fmt.Sprintf("%s:%s", fieldName, newValue))

				if oldValue != newValue {
					compoundChanged = true
				}
			}

			if allFieldsValid {
				oldCompositeValue := strings.Join(oldCompositeKeyParts, "|")
				newCompositeValue := strings.Join(newCompositeKeyParts, "|")
				compoundFieldChanges[compoundIndex] = struct {
					oldCompositeValue string
					newCompositeValue string
					changed           bool
				}{
					oldCompositeValue: oldCompositeValue,
					newCompositeValue: newCompositeValue,
					changed:           compoundChanged,
				}
			}
		}

		// Pass pointer to checkForTTL
		hasTTL, ttl, err := r.checkForTTL(newEntityReflectVal)
		if err != nil {
			return nil, err
		}

		forceUdpate := false
		if hasTTL && ttl != 0 {
			forceUdpate = true
		}

		currentEntityDataVal := currentEntityReflectVal.Elem() // For setting fields if changed

		for _, def := range r.definition.Fields {
			if def.Primary || def.TTL { // Primary key and TTL managed separately
				continue
			}

			// Pass pointers to getNestedFieldValue, it will Elem() internally
			currentFieldVal, err := getNestedFieldValue(currentEntityReflectVal, def.Name)
			if err != nil {
				return nil, fmt.Errorf("error getting current field '%s' for entity ID '%s': %w", def.Name, id, err)
			}
			newFieldVal, err := getNestedFieldValue(newEntityReflectVal, def.Name)
			if err != nil {
				return nil, fmt.Errorf("error getting new field '%s' for entity ID '%s': %w", def.Name, id, err)
			}

			if !currentFieldVal.IsValid() || !newFieldVal.IsValid() {
				// This implies a schema mismatch or partially defined entity.
				// If new field is invalid, it might be an issue. If old is invalid, it means it wasn't set.
				// For now, if either is invalid, we cannot compare or update this field.
				// A more robust system might error if a non-nullable new field is invalid.
				fmt.Printf("Warning: field '%s' is invalid on current or new entity (ID '%s'). Skipping update for this field.\n", def.Name, id)
				continue
			}

			oldValue := fmt.Sprintf("%v", currentFieldVal.Interface())
			newValue := fmt.Sprintf("%v", newFieldVal.Interface())

			if oldValue != newValue || forceUdpate {
				if def.Unique {
					// Always delete old unique index if value changed
					oldUIdxKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, def.Name, oldValue)
					batch.Delete(r.definition.ColumnFamily, r.definition.ColumnFamilySector, oldUIdxKey, now)

					shouldSkipUniquenessForNewValue := false
					if def.HasConditionalUniqueness {
						// newEntityReflectVal is entityPtrVal (*T)
						boolFieldVal, err := getNestedFieldValue(newEntityReflectVal, def.IgnoreIsTrueFieldName)
						if err != nil {
							return nil, fmt.Errorf("error getting conditional uniqueness flag '%s' for entity ID '%s' (update check): %w", def.IgnoreIsTrueFieldName, id, err)
						}
						if boolFieldVal.Kind() == reflect.Bool && boolFieldVal.Bool() {
							shouldSkipUniquenessForNewValue = true
						}
					}

					if !shouldSkipUniquenessForNewValue {
						// Check for new value collision in DB
						idxKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, def.Name, newValue)
						existingIDBytes, errDb := r.kvStore.Get(r.definition.ColumnFamily, r.definition.ColumnFamilySector, idxKey, now)
						if errDb != nil {
							return nil, fmt.Errorf("error checking unique constraint for field '%s', value '%s': %w", def.Name, newValue, errDb)
						}
						if len(existingIDBytes) > 0 && string(existingIDBytes) != id {
							return nil, fmt.Errorf("duplicate unique field: %s = %s (conflicts with existing ID %s)", def.Name, newValue, string(existingIDBytes))
						}

						// Add new unique index
						newUIdxKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, def.Name, newValue)
						if hasTTL { // Assuming hasTTL and ttl are determined for the entity
							batch.PutTTl(r.definition.ColumnFamily, r.definition.ColumnFamilySector, newUIdxKey, []byte(id), ttl, now)
						} else {
							batch.Put(r.definition.ColumnFamily, r.definition.ColumnFamilySector, newUIdxKey, []byte(id), now)
						}
					}
				}

				// Delete old regular index
				oldIdxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, def.Name, oldValue, id)
				batch.Delete(r.definition.ColumnFamily, r.definition.ColumnFamilySector, oldIdxKey, now)

				// Add new regular index
				newIdxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, def.Name, newValue, id)
				if hasTTL {
					batch.PutTTl(r.definition.ColumnFamily, r.definition.ColumnFamilySector, newIdxKey, []byte(id), ttl, now)
				} else {
					batch.Put(r.definition.ColumnFamily, r.definition.ColumnFamilySector, newIdxKey, []byte(id), now)
				}

				// Update the field in the currentEntityDataVal model, which will be marshaled
				targetFieldToSet, errSet := getNestedFieldValue(currentEntityDataVal, def.Name)
				if errSet != nil {
					return nil, fmt.Errorf("failed to get field %s for setting on entity ID %s: %w", def.Name, id, errSet)
				}
				if targetFieldToSet.CanSet() {
					targetFieldToSet.Set(newFieldVal)
				} else {
					return nil, fmt.Errorf("cannot set field %s on current entity for update (ID %s)", def.Name, id)
				}
				changed = true
			}
		}

		// Handle compound unique constraints using pre-calculated values
		for compoundIndex, changeInfo := range compoundFieldChanges {
			if changeInfo.changed {
				// Delete old compound index
				oldCompoundIdxKey := fmt.Sprintf("%s:%s:idx-uc:%d:%s", r.definition.Schema, r.definition.Name, compoundIndex, changeInfo.oldCompositeValue)
				batch.Delete(r.definition.ColumnFamily, r.definition.ColumnFamilySector, oldCompoundIdxKey, now)

				// Check for new compound value collision in DB
				newCompoundIdxKey := fmt.Sprintf("%s:%s:idx-uc:%d:%s", r.definition.Schema, r.definition.Name, compoundIndex, changeInfo.newCompositeValue)
				existingIDBytes, errDb := r.kvStore.Get(r.definition.ColumnFamily, r.definition.ColumnFamilySector, newCompoundIdxKey, now)
				if errDb != nil {
					return nil, fmt.Errorf("error checking compound unique constraint for compound-index=%d, composite=%s: %w", compoundIndex, changeInfo.newCompositeValue, errDb)
				}
				if len(existingIDBytes) > 0 && string(existingIDBytes) != id {
					return nil, fmt.Errorf("duplicate compound unique constraint: compound-index=%d, composite=%s (conflicts with existing ID %s)", compoundIndex, changeInfo.newCompositeValue, string(existingIDBytes))
				}

				// Add new compound index
				if hasTTL {
					batch.PutTTl(r.definition.ColumnFamily, r.definition.ColumnFamilySector, newCompoundIdxKey, []byte(id), ttl, now)
				} else {
					batch.Put(r.definition.ColumnFamily, r.definition.ColumnFamilySector, newCompoundIdxKey, []byte(id), now)
				}

				changed = true
			}
		}

		if changed {
			dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
			// Marshal the modified currentEntityStored, which now contains the merged changes
			dataBytes, err := json.Marshal(entity)
			if err != nil {
				return nil, err
			}
			if hasTTL {
				batch.PutTTl(r.definition.ColumnFamily, r.definition.ColumnFamilySector, dataKey, dataBytes, ttl, now)
			} else {
				batch.Put(r.definition.ColumnFamily, r.definition.ColumnFamilySector, dataKey, dataBytes, now)
			}
		}

		results[i] = changed
	}

	if batch.Count() > 0 {
		if err := r.kvStore.Write(batch); err != nil {
			return nil, err
		}
	}

	return results, nil
}

// Parameters:
//   - entity: A pointer to the entity to update. The primary key field must be populated.
//
// Returns:
//   - A boolean indicating whether the entity was updated.
//   - An error if the operation fails.
func (r *Repository[T]) Update(entity *T, now time.Time) (bool, error) {
	results, err := r.BulkUpdate([]*T{entity}, now)
	if err != nil {
		return false, err
	}
	return results[0], nil
}

// BulkDelete deletes multiple entities from the repository by their IDs in a single batch operation.
// It removes the entity data and all associated index entries.
//
// Parameters:
//   - ids: A slice of strings containing the IDs of the entities to delete.
//
// Returns:
//   - A slice of booleans indicating whether each corresponding entity was found and deleted.
//   - An error if the operation fails.
func (r *Repository[T]) BulkDelete(ids []string, now time.Time) ([]bool, error) {
	var primaryFieldName string
	for name, def := range r.definition.Fields {
		if def.Primary {
			primaryFieldName = name
			break
		}
	}
	if primaryFieldName == "" {
		return nil, fmt.Errorf("no primary key defined")
	}

	results := make([]bool, len(ids))
	batch := NewWriteBatch()

	for i, id := range ids {
		entity, err := r.FindByField(primaryFieldName, id, now)
		if err != nil {
			return nil, fmt.Errorf("error finding entity with id %s: %w", id, err)
		}
		if entity == nil {
			results[i] = false
			continue
		}
		results[i] = true

		entityPtrVal := reflect.ValueOf(entity) // entity is *T (result from FindByField)

		// Delete compound unique indexes for this entity
		for compoundIndex, fieldNames := range r.definition.UniqueCompoundGroups {
			var compositeKeyParts []string

			// Get all field values for this compound constraint
			allFieldsValid := true
			for _, fieldName := range fieldNames {
				fieldVal, err := getNestedFieldValue(entityPtrVal, fieldName)
				if err != nil {
					fmt.Printf("Warning: could not get compound field %s for deleting compound index of entity ID %s: %v. Stale index might remain.\n", fieldName, id, err)
					allFieldsValid = false
					break
				}
				if !fieldVal.IsValid() {
					fmt.Printf("Warning: compound field %s is invalid for deleting compound index of entity ID %s. Stale index might remain.\n", fieldName, id)
					allFieldsValid = false
					break
				}
				fieldValue := fmt.Sprintf("%v", fieldVal.Interface())
				compositeKeyParts = append(compositeKeyParts, fmt.Sprintf("%s:%s", fieldName, fieldValue))
			}

			// Only delete compound index if all fields are valid
			if allFieldsValid {
				// Create composite value by joining all field:value pairs
				compositeValue := strings.Join(compositeKeyParts, "|")
				compoundIdxKey := fmt.Sprintf("%s:%s:idx-uc:%d:%s", r.definition.Schema, r.definition.Name, compoundIndex, compositeValue)
				batch.Delete(r.definition.ColumnFamily, r.definition.ColumnFamilySector, compoundIdxKey, now)
			}
		}

		// Delete regular field indexes for this entity
		for _, def := range r.definition.Fields {
			if def.TTL { // TTL fields don't have separate general indexes
				continue
			}
			// Pass pointer, getNestedFieldValue will Elem()
			fieldVal, err := getNestedFieldValue(entityPtrVal, def.Name)
			if err != nil {
				// If a field doesn't exist on the entity (e.g. schema evolution), we can't delete its index based on value.
				// Log this, as it might lead to stale indexes.
				fmt.Printf("Warning: could not get field %s for deleting index of entity ID %s: %v. Stale index might remain.\n", def.Name, id, err)
				continue
			}
			if !fieldVal.IsValid() {
				fmt.Printf("Warning: field %s is invalid for deleting index of entity ID %s. Stale index might remain.\n", def.Name, id)
				continue
			}

			fieldValue := fmt.Sprintf("%v", fieldVal.Interface())

			idxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, def.Name, fieldValue, id)
			batch.Delete(r.definition.ColumnFamily, r.definition.ColumnFamilySector, idxKey, now)

			if def.Unique {
				idxUKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, def.Name, fieldValue)
				batch.Delete(r.definition.ColumnFamily, r.definition.ColumnFamilySector, idxUKey, now)
			}
		}

		dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
		batch.Delete(r.definition.ColumnFamily, r.definition.ColumnFamilySector, dataKey, now)
	}

	if batch.Count() > 0 {
		if err := r.kvStore.Write(batch); err != nil {
			return nil, fmt.Errorf("error applying bulk delete batch: %w", err)
		}
	}

	return results, nil
}

// Delete deletes a single entity from the repository by its ID.
// It's a convenience wrapper around BulkDelete.
//
// Parameters:
//   - id: The ID of the entity to delete.
//
// Returns:
//   - A boolean indicating whether the entity was found and deleted.
//   - An error if the operation fails.
func (r *Repository[T]) Delete(id string, now time.Time) (bool, error) {
	results, err := r.BulkDelete([]string{id}, now)
	if err != nil {
		return false, err
	}
	return results[0], nil
}

// DefaultIDGeneratorFactory is a default implementation of IDGeneratorFactory
// that uses UUIDs to generate IDs.
type DefaultIDGeneratorFactory struct{}

// GenerateID creates a new unique ID using UUID v4.
// It removes hyphens from the UUID string.
func (idG *DefaultIDGeneratorFactory) GenerateID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

type DeterministicIDGeneratorFactory struct{}

func (idG *DeterministicIDGeneratorFactory) GenerateID() string {
	return ""
}

// getNestedFieldValue retrieves the reflect.Value of a potentially nested field.
// entityValue is the reflect.Value of the struct or a pointer to the struct.
// fieldName can be "Field" or "StructField.NestedField".
func getNestedFieldValue(entityValue reflect.Value, fieldName string) (reflect.Value, error) {
	if fieldName == "" {
		return reflect.Value{}, fmt.Errorf("fieldName cannot be empty")
	}

	currentVal := entityValue
	// Dereference if it's a pointer, until we get the actual struct or a non-pointer.
	for currentVal.Kind() == reflect.Ptr {
		if currentVal.IsNil() {
			return reflect.Value{}, fmt.Errorf("cannot get field '%s' from nil pointer", fieldName)
		}
		currentVal = currentVal.Elem()
	}

	if currentVal.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("expected a struct or pointer to struct to get field '%s', but got %s", fieldName, entityValue.Kind())
	}

	parts := strings.Split(fieldName, ".")
	finalField := currentVal

	for i, part := range parts {
		if finalField.Kind() == reflect.Ptr { // Should have been handled by initial loop, but for safety.
			if finalField.IsNil() {
				return reflect.Value{}, fmt.Errorf("encountered nil pointer while traversing path '%s' at part '%s'", fieldName, part)
			}
			finalField = finalField.Elem()
		}

		if finalField.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("field '%s' in path '%s' is not a struct, but %s", strings.Join(parts[:i], "."), fieldName, finalField.Kind())
		}

		finalField = finalField.FieldByName(part)
		if !finalField.IsValid() {
			return reflect.Value{}, fmt.Errorf("field part '%s' not found in path '%s' on struct type %s", part, fieldName, finalField.Type().Name())
		}
	}
	return finalField, nil
}

// NewRepository creates a new instance of the Repository.
// It inspects the type T to determine the table schema, including field definitions,
// primary key, and unique constraints based on struct tags.
//
// The struct T must have a field named 'ID' of type string with the tag `orm:"primary-key"`.
// Other fields can have tags like `orm:"unique"` or `orm:"max-length=N"`.
//
// Parameters:
//   - kvStore: An instance of KVStore to interact with the underlying key-value database.
//   - ColumnFamily: The name of the column family to use for this repository.
//   - schema: The schema or namespace for the tables managed by this repository.
//   - idGeneratorFactory: A factory for generating unique IDs for new entities.
//
// Returns:
//   - A pointer to the initialized Repository.
//   - An error if the repository initialization fails (e.g., due to invalid struct tags or schema).
func NewRepository[T ORMEntity](kvStore KVStore, ColumnFamily, columnFamilySector string, schema string, idGeneratorFactory IDGeneratorFactory) (*Repository[T], error) {
	t := reflect.TypeOf(new(T)).Elem()

	var tableName string
	var zero T
	if tn, ok := any(zero).(interface{ TableName() string }); ok {
		tableName = tn.TableName()
	} else {
		tableName = t.Name()
	}

	table := &TableDefinition{
		ColumnFamily:         ColumnFamily,
		ColumnFamilySector:   columnFamilySector,
		Schema:               schema,
		Name:                 tableName,
		Fields:               map[string]FieldDefinition{},
		UniqueCompoundGroups: map[int][]string{},
	}

	fields, err := extractFieldsRecursively(t, "")
	if err != nil {
		return nil, fmt.Errorf("error extracting fields from struct %s: %w", t.Name(), err)
	}

	hasPrimaryKey := false
	hasTTL := false
	for _, def := range fields {
		table.Fields[def.Name] = def
		if def.Primary {
			if hasPrimaryKey {
				return nil, fmt.Errorf("multiple primary keys defined in struct %s", t.Name())
			}
			hasPrimaryKey = true
		}
		if def.TTL {
			if hasTTL {
				return nil, fmt.Errorf("multiple TTL fields defined in struct %s", t.Name())
			}
			hasTTL = true
		}
	}

	if !hasPrimaryKey {
		return nil, fmt.Errorf("struct %s must have a string field named 'ID' with `orm:\"primary-key\"`", t.Name())
	}

	// Build unique compound groups
	for fieldName, fieldDef := range table.Fields {
		if fieldDef.IsUniqueCompound {
			index := fieldDef.UniqueCompoundIndex
			if _, exists := table.UniqueCompoundGroups[index]; !exists {
				table.UniqueCompoundGroups[index] = []string{}
			}
			table.UniqueCompoundGroups[index] = append(table.UniqueCompoundGroups[index], fieldName)
		}
	}

	// Validate unique compound groups - each group must have at least 2 fields
	for index, fieldNames := range table.UniqueCompoundGroups {
		if len(fieldNames) < 2 {
			return nil, fmt.Errorf("unique-compound:%d must have at least 2 fields, found only %d field(s): %v", index, len(fieldNames), fieldNames)
		}
		// Sort field names for consistent ordering
		sort.Strings(fieldNames)
		table.UniqueCompoundGroups[index] = fieldNames
	}

	// Validate conditional uniqueness fields
	for fieldName, fieldDef := range table.Fields {
		if fieldDef.HasConditionalUniqueness {
			if fieldDef.IgnoreIsTrueFieldName == "" {
				// This case should ideally be caught by createFieldDefinition
				return nil, fmt.Errorf("internal error: field '%s' has conditional uniqueness but no ignore-is-true field name specified", fieldName)
			}

			referencedFieldName := fieldDef.IgnoreIsTrueFieldName
			referencedFieldDef, ok := table.Fields[referencedFieldName]
			if !ok {
				return nil, fmt.Errorf("field '%s' tagged with 'ignore-is-true:%s', but referenced field '%s' does not exist in struct %s", fieldName, referencedFieldName, referencedFieldName, t.Name())
			}

			if referencedFieldDef.Type != "bool" {
				return nil, fmt.Errorf("field '%s' tagged with 'ignore-is-true:%s', but referenced field '%s' must be of type 'bool', found '%s'", fieldName, referencedFieldName, referencedFieldName, referencedFieldDef.Type)
			}
		}
	}

	return &Repository[T]{definition: table, kvStore: kvStore, idGeneratorFactory: idGeneratorFactory}, nil
}

// GetTableDefinition returns the table definition for testing purposes
func (r *Repository[T]) GetTableDefinition() *TableDefinition {
	return r.definition
}

// extractFieldsRecursively extracts field definitions from a struct type.
// It handles embedded structs and ORM tags.
func extractFieldsRecursively(t reflect.Type, prefix string) ([]FieldDefinition, error) {
	var fields []FieldDefinition

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldType := field.Type

		// Handle special types like time.Time that should not be recursed into
		if fieldType.PkgPath() == "time" && fieldType.Name() == "Time" {
			def, err := createFieldDefinition(field, prefix)
			if err != nil {
				return nil, err
			}
			fields = append(fields, def)
			continue
		}

		switch fieldType.Kind() {
		case reflect.Struct:
			newPrefix := prefix
			if !field.Anonymous { // Named struct
				newPrefix += field.Name + "."
			}
			embeddedFields, err := extractFieldsRecursively(fieldType, newPrefix)
			if err != nil {
				return nil, err
			}
			fields = append(fields, embeddedFields...)
		default:
			def, err := createFieldDefinition(field, prefix)
			if err != nil {
				return nil, err
			}
			fields = append(fields, def)
		}
	}
	return fields, nil
}

// createFieldDefinition creates a FieldDefinition from a reflect.StructField and prefix.
func createFieldDefinition(field reflect.StructField, prefix string) (FieldDefinition, error) {
	tag := field.Tag.Get("orm")
	fullName := prefix + field.Name

	def := FieldDefinition{
		Name:                fullName,
		Type:                field.Type.Name(),
		UniqueCompoundIndex: -1, // Default: not part of compound uniqueness
		IsUniqueCompound:    false,
	}

	for _, rule := range strings.Split(tag, ",") {
		rule = strings.TrimSpace(rule)
		switch {
		case rule == "unique":
			def.Unique = true
		case strings.HasPrefix(rule, "unique-compound:"):
			parts := strings.SplitN(rule, ":", 2)
			if len(parts) != 2 {
				return FieldDefinition{}, fmt.Errorf("invalid unique-compound format for field '%s': %s", fullName, rule)
			}
			index, err := strconv.Atoi(parts[1])
			if err != nil {
				return FieldDefinition{}, fmt.Errorf("invalid unique-compound index for field '%s': %s", fullName, parts[1])
			}
			if index < 0 {
				return FieldDefinition{}, fmt.Errorf("unique-compound index must be >= 0 for field '%s': %d", fullName, index)
			}
			def.IsUniqueCompound = true
			def.UniqueCompoundIndex = index
		case strings.HasPrefix(rule, "ignore-is-true:"):
			parts := strings.SplitN(rule, ":", 2)
			if len(parts) == 2 && parts[1] != "" {
				def.Unique = true // Conditional uniqueness implies uniqueness
				def.HasConditionalUniqueness = true
				def.IgnoreIsTrueFieldName = parts[1]
			} else {
				return FieldDefinition{}, fmt.Errorf("invalid ignore-is-true format for field '%s': %s", fullName, rule)
			}
		case rule == "primary-key":
			if prefix != "" || field.Name != "ID" {
				return FieldDefinition{}, fmt.Errorf("primary key can only be defined on top-level 'ID' field, found on '%s'", fullName)
			}
			if field.Type.Kind() != reflect.String {
				return FieldDefinition{}, fmt.Errorf("field 'ID' must be of type string, found %s for '%s'", field.Type.Kind(), fullName)
			}
			def.Primary = true
		case rule == "ttl":
			if prefix != "" || field.Name != "TTL" {
				return FieldDefinition{}, fmt.Errorf("ttl can only be defined on top-level 'TTL' field, found on '%s'", fullName)
			}
			// TTL can be int, int32, int64, uint, uint32, uint64
			switch field.Type.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint32, reflect.Uint64:
				// valid
			default:
				return FieldDefinition{}, fmt.Errorf("field 'TTL' must be of integer type, found %s for '%s'", field.Type.Kind(), fullName)
			}
			def.TTL = true
		case strings.HasPrefix(rule, "max-length="):
			parts := strings.SplitN(rule, "=", 2)
			if len(parts) != 2 {
				return FieldDefinition{}, fmt.Errorf("invalid max-length format for field '%s': %s", fullName, rule)
			}
			max, err := strconv.Atoi(parts[1])
			if err != nil {
				return FieldDefinition{}, fmt.Errorf("invalid max-length value for field '%s': %s", fullName, parts[1])
			}
			if max <= 0 {
				return FieldDefinition{}, fmt.Errorf("max-length must be positive for field '%s': %d", fullName, max)
			}
			def.MaxLength = &max
		case rule == "":
			// ignore empty rule
		default:
			// Optional: Log or return error for unknown tags
			// fmt.Printf("Warning: Unknown ORM tag '%s' for field '%s'\n", rule, fullName)
		}
	}
	return def, nil
}

func NewRepositoryWithBatch[T ORMEntity](kvStore KVStore, ColumnFamily, columnFamilySector, schema string, idGeneratorFactory IDGeneratorFactory, batch *WriteBatch) (*Repository[T], error) {
	repo, err := NewRepository[T](kvStore, ColumnFamily, columnFamilySector, schema, idGeneratorFactory)
	if err != nil {
		return nil, err
	}
	// TODO: This is a temporary hack. We should not be modifying the kvStore after the repository is created.
	// This should be fixed by making the batch part of the KVStore interface or by passing it to the methods that need it.
	// For now, we will just replace the kvStore with a delegated one.
	repo.kvStore = &DelegatedKVStore{
		base:  kvStore,
		batch: batch,
	}
	return repo, nil
}
