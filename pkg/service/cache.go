package service

import (
	"context"

	"github.com/danielkrainas/sake/pkg/util/log"
	memdb "github.com/hashicorp/go-memdb"
)

type CacheService interface {
	PutWorkflow(ctx context.Context, wf *Workflow) error
	GetWorkflow(ctx context.Context, name string) (*Workflow, error)
	RemoveWorkflow(ctx context.Context, wf *Workflow) error
	PutTransaction(ctx context.Context, trx *Transaction) error
	GetTransaction(ctx context.Context, id string) (*Transaction, error)
	RemoveTransaction(ctx context.Context, trx *Transaction) error
	TransactAll(ctx context.Context, action func(trx *Transaction) error) error
}

type WriteThruCache struct {
	CacheService
	Storage StorageService
}

var _ CacheService = &WriteThruCache{}

func (thru *WriteThruCache) RemoveWorkflow(ctx context.Context, wf *Workflow) error {
	go func(wf *Workflow) {
		if err := thru.Storage.RemoveWorkflow(ctx, wf); err != nil {
			//
		}
	}(wf)

	return thru.CacheService.RemoveWorkflow(ctx, wf)
}

func (thru *WriteThruCache) PutTransaction(ctx context.Context, trx *Transaction) error {
	go func(trx *Transaction) {
		if err := thru.Storage.SaveTransaction(ctx, trx); err != nil {
			//
		}
	}(trx)

	return thru.CacheService.PutTransaction(ctx, trx)
}

func (thru *WriteThruCache) PutWorkflow(ctx context.Context, wf *Workflow) error {
	go func(wf *Workflow) {
		if err := thru.Storage.SaveWorkflow(ctx, wf); err != nil {
			//
		}
	}(wf)

	return thru.CacheService.PutWorkflow(ctx, wf)
}

type InMemoryCache struct {
	db *memdb.MemDB
}

var _ CacheService = &InMemoryCache{}

func NewInMemoryCache() (*InMemoryCache, error) {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"workflow": &memdb.TableSchema{
				Name: "workflow",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},
			"transaction": &memdb.TableSchema{
				Name: "transaction",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
				},
			},
		},
	}

	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}

	log.Info("in-memory cache ready")
	return &InMemoryCache{
		db: db,
	}, nil
}

func (cache *InMemoryCache) PutWorkflow(ctx context.Context, wf *Workflow) error {
	transact := cache.db.Txn(true)
	if err := transact.Insert("workflow", wf); err != nil {
		transact.Abort()
		return err
	}

	transact.Commit()
	return nil
}

func (cache *InMemoryCache) GetWorkflow(ctx context.Context, name string) (*Workflow, error) {
	transact := cache.db.Txn(false)
	defer transact.Abort()

	iwf, err := transact.First("workflow", "id", name)
	if err != nil {
		return nil, err
	} else if iwf == nil {
		return nil, nil
	}

	return iwf.(*Workflow), nil
}

func (cache *InMemoryCache) RemoveWorkflow(ctx context.Context, wf *Workflow) error {
	transact := cache.db.Txn(true)
	iwf, err := transact.First("workflow", "id", wf.Name)
	if err == nil && iwf != nil {
		err = transact.Delete("workflow", iwf)
	}

	if err != nil {
		transact.Abort()
		return err
	}

	transact.Commit()
	return nil
}

func (cache *InMemoryCache) PutTransaction(ctx context.Context, trx *Transaction) error {
	transact := cache.db.Txn(true)
	if err := transact.Insert("transaction", trx); err != nil {
		transact.Abort()
		return err
	}

	transact.Commit()
	return nil
}

func (cache *InMemoryCache) GetTransaction(ctx context.Context, id string) (*Transaction, error) {
	transact := cache.db.Txn(false)
	defer transact.Abort()

	itrx, err := transact.First("transaction", "id", id)
	if err != nil {
		return nil, err
	} else if itrx == nil {
		return nil, nil
	}

	return itrx.(*Transaction), nil
}

func (cache *InMemoryCache) RemoveTransaction(ctx context.Context, trx *Transaction) error {
	transact := cache.db.Txn(true)
	itrx, err := transact.First("transaction", "id", trx.ID)
	if err == nil && itrx != nil {
		err = transact.Delete("transaction", itrx)
	}

	if err != nil {
		transact.Abort()
		return err
	}

	transact.Commit()
	return nil
}

func (cache *InMemoryCache) TransactAll(ctx context.Context, action func(trx *Transaction) error) error {
	transact := cache.db.Txn(false)
	defer transact.Abort()
	it, err := transact.Get("transaction", "id")
	if err != nil {
		return err
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		trx := obj.(*Transaction)
		err = action(trx)
		/*if updated != nil && err == nil {
			err = transact.Insert("transaction", updated)
		}*/

		if err != nil {
			return err
		}
	}

	return nil
}
