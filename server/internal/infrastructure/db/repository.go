package db

import (
	"encoding/json"
	"fmt"
	"reflect"
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

	fmt.Println("aqui se hace el primer search")
	fmt.Println("aqui se hace el primer GET")
	current, err := r.FindByField(primaryFieldName, id)
	if err != nil {
		return false, err
	}
	if current == nil {
		return false, nil // no se encontró, no hay cambios
	}

	changed := false
	currentVal := reflect.ValueOf(current).Elem()
	newVal := reflect.ValueOf(entity).Elem()

	for fieldName, def := range r.definition.Fields {
		if def.Primary {
			continue // no se permite cambiar el campo primario
		}

		curField := currentVal.FieldByName(fieldName)
		newField := newVal.FieldByName(fieldName)

		if !curField.IsValid() || !newField.IsValid() {
			continue
		}

		oldValue := fmt.Sprintf("%v", curField.Interface())
		newValue := fmt.Sprintf("%v", newField.Interface())

		if oldValue != newValue {
			// Si es unique, validar que el nuevo valor no exista ya
			if def.Unique {
				idxKey := fmt.Sprintf("%s:%s:idx:%s:%s:*", r.definition.Schema, r.definition.Name, fieldName, newValue)
				fmt.Println("aqui se hace el segundo search")
				existing, _, err := r.kvStore.SearchByPatternPaginatedKV(r.definition.ColumnFamily, idxKey, "", 1)
				if err != nil {
					return false, err
				}
				if len(existing) > 0 && string(existing[0].Value) != id {
					return false, fmt.Errorf("duplicate unique field: %s = %s", fieldName, newValue)
				}
			}

			// Eliminar índice viejo
			oldIdxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, oldValue, id)
			fmt.Println("El delete del viejo aqui ", oldValue)
			if err := r.kvStore.Delete(r.definition.ColumnFamily, oldIdxKey); err != nil {
				return false, err
			}

			// Crear nuevo índice
			newIdxKey := fmt.Sprintf("%s:%s:idx:%s:%s:%s", r.definition.Schema, r.definition.Name, fieldName, newValue, id)
			fmt.Println("El Put del nuevo aqui ", newValue)
			if err := r.kvStore.Put(r.definition.ColumnFamily, newIdxKey, []byte(id)); err != nil {
				return false, err
			}

			// Aplicar el cambio al struct actual
			curField.Set(newField)
			changed = true
		}
	}

	if changed {
		// Guardar objeto actualizado
		dataKey := fmt.Sprintf("%s:%s:data:%s", r.definition.Schema, r.definition.Name, id)
		dataBytes, err := json.Marshal(current)
		if err != nil {
			return false, err
		}
		fmt.Println("Guardando aqui el nuevo objeto ", current)
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
