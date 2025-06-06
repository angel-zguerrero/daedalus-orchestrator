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

type IDGeneratorFactory interface {
	GenerateID() string
}
type ORMEntity interface {
	TableName() string
}

type FieldDefinition struct {
	Name      string
	Type      string
	Unique    bool
	Primary   bool
	MaxLength *int
}

type TableDefinition struct {
	ColumnFamily string
	Schema       string
	Name         string
	Fields       map[string]FieldDefinition
}

type Repository[T ORMEntity] struct {
	definition         *TableDefinition
	kvStore            KVStore
	idGeneratorFactory IDGeneratorFactory
}

type FindResult[T any] struct {
	Entities []T
	Cursor   string
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

type exprNode struct {
	op      string // "&" or "|" or "COND"
	left    *exprNode
	right   *exprNode
	condStr string
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

func (r *Repository[T]) FindByField(field string, value string) (*T, error) {
	searchKey := fmt.Sprintf("%s:%s:idx:%s:%s:*", r.definition.Schema, r.definition.Name, field, value)
	idBytes, _, err := r.kvStore.SearchByPatternPaginatedKV(r.definition.ColumnFamily, searchKey, "", 1)
	if err != nil || idBytes == nil || len(idBytes) == 0 {
		return nil, nil
	}

	dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, string(idBytes[0].Value))
	dataBytes, err := r.kvStore.Get(r.definition.ColumnFamily, dataKey)
	if err != nil {
		return nil, err
	}

	var result T
	err = json.Unmarshal(dataBytes, &result)
	return &result, err
}

func (r *Repository[T]) Create(entity *T) (string, error) {
	id := r.idGeneratorFactory.GenerateID()
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
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
			idxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, fieldValue, id)
			exists, err := r.kvStore.Exists(r.definition.ColumnFamily, idxKey)
			if err != nil {
				return "", err
			}
			if exists {
				return "", fmt.Errorf("duplicate unique field: %s = %s", fieldName, fieldValue)
			}
		}
	}

	for fieldName, _ := range r.definition.Fields {
		fieldValue := fmt.Sprintf("%v", val.FieldByName(fieldName).Interface())
		idxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, fieldValue, id)
		if err := r.kvStore.Put(r.definition.ColumnFamily, idxKey, []byte(id)); err != nil {
			return "", err
		}
	}

	dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
	dataBytes, err := json.Marshal(entity)
	if err != nil {
		return "", err
	}

	if err := r.kvStore.Put(r.definition.ColumnFamily, dataKey, dataBytes); err != nil {
		return "", err
	}

	return id, nil
}

func (r *Repository[T]) Update(id string, entity *T) (bool, error) {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Buscar campo primary
	var primaryFieldName string
	for name, def := range r.definition.Fields {
		if def.Primary {
			primaryFieldName = name
			break
		}
	}
	if primaryFieldName == "" {
		return false, fmt.Errorf("no primary key defined")
	}

	current, err := r.FindByField(primaryFieldName, id)
	if err != nil {
		return false, err
	}
	if current == nil {
		return false, nil
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
				idxKey := fmt.Sprintf("%s:%s:idx:%s:%s:*", r.definition.Schema, r.definition.Name, fieldName, newValue)
				existing, _, err := r.kvStore.SearchByPatternPaginatedKV(r.definition.ColumnFamily, idxKey, "", 1)
				if err != nil {
					return false, err
				}
				if len(existing) > 0 && string(existing[0].Value) != id {
					return false, fmt.Errorf("duplicate unique field: %s = %s", fieldName, newValue)
				}
			}

			oldIdxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, oldValue, id)
			if err := r.kvStore.Delete(r.definition.ColumnFamily, oldIdxKey); err != nil {
				return false, err
			}

			newIdxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, newValue, id)
			if err := r.kvStore.Put(r.definition.ColumnFamily, newIdxKey, []byte(id)); err != nil {
				return false, err
			}

			curField.Set(newField)
			changed = true
		}
	}

	if changed {
		dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
		dataBytes, err := json.Marshal(current)
		if err != nil {
			return false, err
		}

		if err := r.kvStore.Put(r.definition.ColumnFamily, dataKey, dataBytes); err != nil {
			return false, err
		}
	}

	return changed, nil
}

func (r *Repository[T]) Delete(id string) (bool, error) {
	var primaryFieldName string
	for name, def := range r.definition.Fields {
		if def.Primary {
			primaryFieldName = name
			break
		}
	}
	if primaryFieldName == "" {
		return false, fmt.Errorf("no primary key defined")
	}

	entity, err := r.FindByField(primaryFieldName, id)
	if err != nil {
		return false, err
	}
	if entity == nil {
		return false, nil
	}

	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	for fieldName := range r.definition.Fields {
		fieldValue := fmt.Sprintf("%v", val.FieldByName(fieldName).Interface())
		idxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, fieldValue, id)
		if err := r.kvStore.Delete(r.definition.ColumnFamily, idxKey); err != nil {
			return false, fmt.Errorf("error deleting index key %s: %w", idxKey, err)
		}
	}

	dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
	if err := r.kvStore.Delete(r.definition.ColumnFamily, dataKey); err != nil {
		return false, fmt.Errorf("error deleting data key %s: %w", dataKey, err)
	}

	return true, nil
}

type DefaultIDGeneratorFactory struct{}

func (idG *DefaultIDGeneratorFactory) GenerateID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

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
