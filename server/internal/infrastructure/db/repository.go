package db

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

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
	definition *TableDefinition
	kvStore    KVStore
}

func (r *Repository[T]) Find(filter string) ([]T, error) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return nil, fmt.Errorf("filter string is empty")
	}

	orConditions := strings.Split(filter, "|")
	orResults := map[string]bool{}
	var matchedIDs []string

	for _, orCond := range orConditions {
		orCond = strings.TrimSpace(orCond)
		andConditions := strings.Split(orCond, "&")

		var andMatchedIDs map[string]bool
		for i, andCond := range andConditions {
			andCond = strings.TrimSpace(andCond)
			parts := strings.SplitN(andCond, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid condition: %s", andCond)
			}
			field := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "'")

			searchKey := fmt.Sprintf("%s:%s:idx:%s:%s:*", r.definition.Schema, r.definition.Name, field, value)
			idxBytes, _, err := r.kvStore.SearchByPatternPaginatedKV(r.definition.ColumnFamily, searchKey, "", 1000)
			if err != nil {
				return nil, err
			}

			ids := make(map[string]bool)
			for _, item := range idxBytes {
				ids[string(item.Value)] = true
			}

			if i == 0 {
				andMatchedIDs = ids
			} else {
				for id := range andMatchedIDs {
					if !ids[id] {
						delete(andMatchedIDs, id)
					}
				}
			}
		}

		for id := range andMatchedIDs {
			orResults[id] = true
		}
	}

	for id := range orResults {
		matchedIDs = append(matchedIDs, id)
	}

	var results []T
	for _, id := range matchedIDs {
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

	return results, nil
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

func (r *Repository[T]) Create(entity T) error {
	val := reflect.ValueOf(entity)
	t := reflect.TypeOf(entity)

	var id string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		def, exists := r.definition.Fields[field.Name]
		if !exists {
			continue
		}
		if def.Primary {
			id = fmt.Sprintf("%v", val.Field(i).Interface())
			break
		}
	}

	if id == "" {
		return fmt.Errorf("missing primary key value for %s", r.definition.Name)
	}

	for fieldName, def := range r.definition.Fields {
		if def.Unique {
			fieldValue := fmt.Sprintf("%v", val.FieldByName(fieldName).Interface())
			idxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, fieldValue, id)
			exists, err := r.kvStore.Exists(r.definition.ColumnFamily, idxKey)
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("duplicate unique field: %s = %s", fieldName, fieldValue)
			}
		}
	}

	for fieldName, _ := range r.definition.Fields {
		fieldValue := fmt.Sprintf("%v", val.FieldByName(fieldName).Interface())
		idxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, fieldValue, id)
		if err := r.kvStore.Put(r.definition.ColumnFamily, idxKey, []byte(id)); err != nil {
			return err
		}
	}

	dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
	dataBytes, err := json.Marshal(entity)
	if err != nil {
		return err
	}

	if err := r.kvStore.Put(r.definition.ColumnFamily, dataKey, dataBytes); err != nil {
		return err
	}

	return nil
}

func NewRepository[T ORMEntity](kvStore KVStore, ColumnFamily string, schema string) (*Repository[T], error) {
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
				def.Primary = true
				hasPrimaryKey = true
			case strings.HasPrefix(rule, "maxLength="):
				var max int
				fmt.Sscanf(rule, "maxLength=%d", &max)
				def.MaxLength = &max
			}
		}

		table.Fields[field.Name] = def
	}

	if !hasPrimaryKey {
		return nil, fmt.Errorf("no primaryKey field defined in model %s", table.Name)
	}

	return &Repository[T]{definition: table, kvStore: kvStore}, nil
}
