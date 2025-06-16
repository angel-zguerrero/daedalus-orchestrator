package db

import (
	"reflect"
)

type UnitOfWork struct {
	batch   *WriteBatch
	repos   map[string]interface{}
	kvStore KVStore
}

func NewUnitOfWork(kvStore KVStore) *UnitOfWork {
	return &UnitOfWork{
		batch:   NewWriteBatch(),
		repos:   make(map[string]interface{}),
		kvStore: kvStore,
	}
}

func (u *UnitOfWork) Commit() error {
	return u.kvStore.Write(u.batch)
}

func GetRepository[T ORMEntity](uow *UnitOfWork, cf string, schema string, factory IDGeneratorFactory) (*Repository[T], error) {
	key := schema + "/" + cf + "/" + reflect.TypeOf((*T)(nil)).Elem().Name()
	if repo, ok := uow.repos[key]; ok {
		return repo.(*Repository[T]), nil
	}
	repo, err := NewRepositoryWithBatch[T](uow.kvStore, cf, schema, factory, uow.batch)
	if err != nil {
		return nil, err
	}
	uow.repos[key] = repo
	return repo, nil
}
