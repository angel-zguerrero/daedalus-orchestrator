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
	"strings"

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
}

// TableDefinition describes the schema of a table in the key-value store.
type TableDefinition struct {
	// ColumnFamily is the name of the column family where the table data is stored.
	ColumnFamily string
	// Schema is the namespace or schema name for the table.
	Schema string
	// Name is the name of the table.
	Name string
	// Fields is a map of field names to their definitions.
	Fields map[string]FieldDefinition
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
func (r *Repository[T]) evalCondition(condStr string, limit int) (map[string]bool, error) {
	conditionRegex := regexp.MustCompile(`(?i)^([\w.]+)\s*(=|!=|<=|>=|<|>|LIKE|BETWEEN)\s*(.+)$`)
	parts := conditionRegex.FindStringSubmatch(strings.TrimSpace(condStr))
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid condition: %s", condStr)
	}

	field := strings.TrimSpace(parts[1])
	operator := strings.ToUpper(strings.TrimSpace(parts[2]))
	value := strings.TrimSpace(strings.Trim(parts[3], "'"))

	allIDs := make(map[string]bool)
	prefix := fmt.Sprintf("%s:%s:idx:%s:", r.definition.Schema, r.definition.Name, field)
	cursorInner := ""

	switch operator {
	case "=":
		pattern := prefix + value + ":*"
		for {
			items, next, err := r.kvStore.SearchByPatternPaginatedKV(r.definition.ColumnFamily, pattern, cursorInner, limit)
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
			items, next, err := r.kvStore.SearchByPatternPaginatedKV(r.definition.ColumnFamily, prefix+"*", cursorInner, limit)
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
			items, next, err := r.kvStore.SearchByPatternPaginatedKV(r.definition.ColumnFamily, prefix+"*", cursorInner, limit)
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
func (r *Repository[T]) evalExpr(node *exprNode, limit int) (map[string]bool, error) {
	if node.op == "COND" {
		return r.evalCondition(node.condStr, limit)
	}
	left, err := r.evalExpr(node.left, limit)
	if err != nil {
		return nil, err
	}
	right, err := r.evalExpr(node.right, limit)
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
func (r *Repository[T]) Find(filter string, limit int, cursor string) (*FindResult[T], error) {
	tokens, err := tokenize(filter)
	if err != nil {
		return nil, err
	}
	tree, err := parse(tokens)
	if err != nil {
		return nil, err
	}
	idMap, err := r.evalExpr(tree, limit)
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
	var results []T
	for _, id := range selectedIDs {
		dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
		dataBytes, err := r.kvStore.Get(r.definition.ColumnFamily, dataKey)
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
func (r *Repository[T]) FindByField(field string, value string) (*T, error) {
	def := r.definition.Fields[field]
	var dataKey string
	if def.Unique {
		searchKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, field, value)
		idBytes, err := r.kvStore.Get(r.definition.ColumnFamily, searchKey)
		if err != nil || idBytes == nil || len(idBytes) == 0 {
			return nil, nil
		}

		dataKey = fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, string(idBytes))
	} else if def.Primary {
		dataKey = fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, value)
	} else {
		searchKey := fmt.Sprintf("%s:%s:idx:%s:%s:*", r.definition.Schema, r.definition.Name, field, value)
		idBytes, _, err := r.kvStore.SearchByPatternPaginatedKV(r.definition.ColumnFamily, searchKey, "", 1)
		if err != nil || idBytes == nil || len(idBytes) == 0 {
			return nil, nil
		}

		dataKey = fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, string(idBytes[0].Value))
	}
	dataBytes, err := r.kvStore.Get(r.definition.ColumnFamily, dataKey)
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
func (r *Repository[T]) BulkCreate(entities []*T) ([]string, error) {
	var ids []string
	batch := NewWriteBatch()
	type uniqueCheck struct {
		Key       string
		FieldName string
		Value     string
	}
	var uniqueChecks []uniqueCheck

	// Map para detectar duplicados en el batch, clave = campo+valor
	uniqueInBatch := make(map[string]struct{})

	for _, entity := range entities {
		id := r.idGeneratorFactory.GenerateID()
		ids = append(ids, id)

		val := reflect.ValueOf(entity)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// Set primary key field
		for fieldName, def := range r.definition.Fields {
			if def.Primary {
				field := val.FieldByName(fieldName)
				if field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
					field.SetString(id)
				}
				break
			}
		}

		for fieldName, def := range r.definition.Fields {
			if def.Unique {
				fieldValue := fmt.Sprintf("%v", val.FieldByName(fieldName).Interface())
				uniqueIdxKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, fieldName, fieldValue)
				uniqueChecks = append(uniqueChecks, uniqueCheck{
					Key:       uniqueIdxKey,
					FieldName: fieldName,
					Value:     fieldValue,
				})

				// Validación de duplicados en el mismo batch
				batchKey := fieldName + ":" + fieldValue
				if _, exists := uniqueInBatch[batchKey]; exists {
					return nil, fmt.Errorf("duplicate unique field in input batch: %s = %s", fieldName, fieldValue)
				}
				uniqueInBatch[batchKey] = struct{}{}
			}
		}
	}

	// Validar duplicados en la base
	for _, check := range uniqueChecks {
		exists, err := r.kvStore.Exists(r.definition.ColumnFamily, check.Key)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("duplicate unique field: %s = %s", check.FieldName, check.Value)
		}
	}

	// Insertar datos y sus índices
	for i, entity := range entities {
		val := reflect.ValueOf(entity)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		id := ids[i]

		for fieldName, def := range r.definition.Fields {
			fieldValue := fmt.Sprintf("%v", val.FieldByName(fieldName).Interface())
			idxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, fieldValue, id)
			batch.Put(r.definition.ColumnFamily, idxKey, []byte(id))

			if def.Unique {
				uniqueIdxKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, fieldName, fieldValue)
				batch.Put(r.definition.ColumnFamily, uniqueIdxKey, []byte(id))
			}
		}

		dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
		dataBytes, err := json.Marshal(entity)
		if err != nil {
			return nil, err
		}
		batch.Put(r.definition.ColumnFamily, dataKey, dataBytes)
	}

	if err := r.kvStore.Write(batch); err != nil {
		return nil, err
	}

	return ids, nil
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
func (r *Repository[T]) Create(entity *T) (string, error) {
	ids, err := r.BulkCreate([]*T{entity})
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
func (r *Repository[T]) BulkUpdate(entities []*T) ([]bool, error) {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

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

	results := make([]bool, len(entities))
	batch := NewWriteBatch()

	uniqueValuesInBatch := make(map[string]map[string]string)

	for i, entity := range entities {
		if entity == nil {
			results[i] = false
			continue
		}
		entityVal := reflect.ValueOf(entity).Elem()
		id := fmt.Sprintf("%v", entityVal.FieldByName(primaryFieldName).Interface())

		for fieldName, def := range r.definition.Fields {
			if def.Primary || !def.Unique {
				continue
			}

			field := entityVal.FieldByName(fieldName)
			if !field.IsValid() {
				continue
			}

			value := fmt.Sprintf("%v", field.Interface())

			if _, ok := uniqueValuesInBatch[fieldName]; !ok {
				uniqueValuesInBatch[fieldName] = make(map[string]string)
			}

			if existingID, exists := uniqueValuesInBatch[fieldName][value]; exists && existingID != id {
				return nil, fmt.Errorf("duplicate unique field in batch: %s = %s", fieldName, value)
			}
			uniqueValuesInBatch[fieldName][value] = id
		}
	}

	for i, entity := range entities {
		if entity == nil {
			results[i] = false
			continue
		}
		entityVal := reflect.ValueOf(entity).Elem()
		id := fmt.Sprintf("%v", entityVal.FieldByName(primaryFieldName).Interface())

		current, err := r.FindByField(primaryFieldName, id)
		if err != nil {
			return nil, err
		}
		if current == nil {
			results[i] = false
			continue
		}

		changed := false
		currentVal := reflect.ValueOf(current).Elem()
		newVal := reflect.ValueOf(entity).Elem()

		for fieldName, def := range r.definition.Fields {
			if def.Primary {
				continue
			}

			curField := currentVal.FieldByName(fieldName)
			newField := newVal.FieldByName(fieldName)

			if !curField.IsValid() || !newField.IsValid() {
				continue
			}

			oldValue := fmt.Sprintf("%v", curField.Interface())
			newValue := fmt.Sprintf("%v", newField.Interface())

			if oldValue != newValue {
				if def.Unique {
					idxKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, fieldName, newValue)
					existing, err := r.kvStore.Get(r.definition.ColumnFamily, idxKey)
					if err != nil {
						return nil, err
					}
					if len(existing) > 0 && string(existing) != id {
						return nil, fmt.Errorf("duplicate unique field: %s = %s", fieldName, newValue)
					}

					oldUIdxKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, fieldName, oldValue)
					batch.Delete(r.definition.ColumnFamily, oldUIdxKey)

					newUIdxKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, fieldName, newValue)
					batch.Put(r.definition.ColumnFamily, newUIdxKey, []byte(id))
				}

				oldIdxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, oldValue, id)
				batch.Delete(r.definition.ColumnFamily, oldIdxKey)

				newIdxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, newValue, id)
				batch.Put(r.definition.ColumnFamily, newIdxKey, []byte(id))

				curField.Set(newField)
				changed = true
			}
		}

		if changed {
			dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
			dataBytes, err := json.Marshal(current)
			if err != nil {
				return nil, err
			}
			batch.Put(r.definition.ColumnFamily, dataKey, dataBytes)
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

// Update updates a single entity in the repository.
// It's a convenience wrapper around BulkUpdate.
//
// Parameters:
//   - entity: A pointer to the entity to update. The primary key field must be populated.
//
// Returns:
//   - A boolean indicating whether the entity was updated.
//   - An error if the operation fails.
func (r *Repository[T]) Update(entity *T) (bool, error) {
	results, err := r.BulkUpdate([]*T{entity})
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
func (r *Repository[T]) BulkDelete(ids []string) ([]bool, error) {
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
		entity, err := r.FindByField(primaryFieldName, id)
		if err != nil {
			return nil, fmt.Errorf("error finding entity with id %s: %w", id, err)
		}
		if entity == nil {
			results[i] = false
			continue
		}
		results[i] = true

		val := reflect.ValueOf(entity)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		for fieldName, def := range r.definition.Fields {
			fieldValue := fmt.Sprintf("%v", val.FieldByName(fieldName).Interface())

			idxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, fieldValue, id)
			batch.Delete(r.definition.ColumnFamily, idxKey)

			if def.Unique {
				idxUKey := fmt.Sprintf("%s:%s:idx-u:%s:%s", r.definition.Schema, r.definition.Name, fieldName, fieldValue)
				batch.Delete(r.definition.ColumnFamily, idxUKey)
			}
		}

		dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
		batch.Delete(r.definition.ColumnFamily, dataKey)
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
func (r *Repository[T]) Delete(id string) (bool, error) {
	results, err := r.BulkDelete([]string{id})
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

// NewRepository creates a new instance of the Repository.
// It inspects the type T to determine the table schema, including field definitions,
// primary key, and unique constraints based on struct tags.
//
// The struct T must have a field named 'ID' of type string with the tag `orm:"primaryKey"`.
// Other fields can have tags like `orm:"unique"` or `orm:"maxLength=N"`.
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
func NewRepository[T ORMEntity](kvStore KVStore, ColumnFamily string, schema string, idGeneratorFactory IDGeneratorFactory) (*Repository[T], error) {
	t := reflect.TypeOf(new(T)).Elem()

	var tableName string
	var zero T
	if tn, ok := any(zero).(interface{ TableName() string }); ok {
		tableName = tn.TableName()
	} else {
		tableName = t.Name()
	}

	table := &TableDefinition{
		ColumnFamily: ColumnFamily,
		Schema:       schema,
		Name:         tableName,
		Fields:       map[string]FieldDefinition{},
	}

	hasPrimaryKey := false

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("orm")

		def := FieldDefinition{
			Name: field.Name,
			Type: field.Type.Name(),
		}

		for _, rule := range strings.Split(tag, ",") {
			rule = strings.TrimSpace(rule)
			switch {
			case rule == "unique":
				def.Unique = true
			case rule == "primaryKey":
				if field.Type.Kind() != reflect.String {
					return nil, fmt.Errorf("field 'ID' must be of type string")
				}
				def.Primary = true
				hasPrimaryKey = true
				if field.Name != "ID" {
					hasPrimaryKey = false
				}
			case strings.HasPrefix(rule, "maxLength="):
				var max int
				fmt.Sscanf(rule, "maxLength=%d", &max)
				def.MaxLength = &max
			}
		}

		table.Fields[field.Name] = def
	}

	if !hasPrimaryKey {
		return nil, fmt.Errorf("struct %s must have a string field named 'ID' with `orm:primaryKey`", t.Name())
	}

	return &Repository[T]{definition: table, kvStore: kvStore, idGeneratorFactory: idGeneratorFactory}, nil
}
