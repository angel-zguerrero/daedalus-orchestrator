package db

import (
	"reflect"
)

type UnitOfWork struct {
	batch   *WriteBatch
	repos   map[string]interface{}
	KVStore KVStore
}

func NewUnitOfWork(kvStore KVStore, batch *WriteBatch) *UnitOfWork {
	if batch == nil {
		batch = NewWriteBatch()
	}
	return &UnitOfWork{
		batch:   batch,
		repos:   make(map[string]interface{}),
		KVStore: kvStore,
	}
}

func (u *UnitOfWork) Commit() error {
	defer func() {
		u.batch.Data = []DataStruct{}
	}()
	return u.KVStore.Write(u.batch)
}

func GetRepository[T ORMEntity](uow *UnitOfWork, cf string, schema string, factory IDGeneratorFactory) (*Repository[T], error) {
	key := schema + "/" + cf + "/" + reflect.TypeOf((*T)(nil)).Elem().Name()
	if repo, ok := uow.repos[key]; ok {
		return repo.(*Repository[T]), nil
	}
	repo, err := NewRepositoryWithBatch[T](uow.KVStore, cf, schema, factory, uow.batch)
	if err != nil {
		return nil, err
	}
	uow.repos[key] = repo
	return repo, nil
}
