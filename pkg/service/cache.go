package service

import (
	"context"

	"github.com/danielkrainas/sake/pkg/util/log"
	memdb "github.com/hashicorp/go-memdb"
)

type CacheService interface {
	PutRecipe(ctx context.Context, recipe *Recipe) error
	GetAllRecipes(ctx context.Context) ([]*Recipe, error)
	RemoveRecipe(ctx context.Context, recipe *Recipe) error
	PutTransaction(ctx context.Context, trx *Transaction) error
	GetTransaction(ctx context.Context, id string) (*Transaction, error)
	RemoveTransaction(ctx context.Context, trx *Transaction) error
	TransactAll(ctx context.Context, action func(trx *Transaction) error) error
	FilterRecipes(ctx context.Context, predicate func(recipe *Recipe) (bool, error)) ([]*Recipe, error)
}

type WriteThruCache struct {
	CacheService
	Storage StorageService
}

var _ CacheService = &WriteThruCache{}

func (thru *WriteThruCache) RemoveRecipe(ctx context.Context, recipe *Recipe) error {
	go func(recipe *Recipe) {
		if err := thru.Storage.RemoveRecipe(ctx, recipe); err != nil {
			//
		}
	}(recipe)

	return thru.CacheService.RemoveRecipe(ctx, recipe)
}

func (thru *WriteThruCache) PutTransaction(ctx context.Context, trx *Transaction) error {
	go func(trx *Transaction) {
		if err := thru.Storage.SaveTransaction(ctx, trx); err != nil {
			//
		}
	}(trx)

	return thru.CacheService.PutTransaction(ctx, trx)
}

func (thru *WriteThruCache) PutRecipe(ctx context.Context, recipe *Recipe) error {
	go func(recipe *Recipe) {
		if err := thru.Storage.SaveRecipe(ctx, recipe); err != nil {
			//
		}
	}(recipe)

	return thru.CacheService.PutRecipe(ctx, recipe)
}

type InMemoryCache struct {
	db *memdb.MemDB
}

var _ CacheService = &InMemoryCache{}

func NewInMemoryCache() (*InMemoryCache, error) {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"recipe": &memdb.TableSchema{
				Name: "recipe",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
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

func (cache *InMemoryCache) PutRecipe(ctx context.Context, recipe *Recipe) error {
	transact := cache.db.Txn(true)
	if err := transact.Insert("recipe", recipe); err != nil {
		transact.Abort()
		return err
	}

	transact.Commit()
	return nil
}

func (cache *InMemoryCache) GetAllRecipes(ctx context.Context) ([]*Recipe, error) {
	result := make([]*Recipe, 0)
	transact := cache.db.Txn(false)
	defer transact.Abort()
	it, err := transact.Get("recipe", "id")
	if err != nil {
		return nil, err
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		recipe, ok := obj.(*Recipe)
		if ok {
			result = append(result, recipe)
		}
	}

	return result, nil
}

func (cache *InMemoryCache) RemoveRecipe(ctx context.Context, recipe *Recipe) error {
	transact := cache.db.Txn(true)
	irecipe, err := transact.First("recipe", "id", recipe.ID)
	if err == nil && irecipe != nil {
		err = transact.Delete("recipe", irecipe)
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

func (cache *InMemoryCache) FilterRecipes(ctx context.Context, predicate func(recipe *Recipe) (bool, error)) ([]*Recipe, error) {
	result := make([]*Recipe, 0)
	transact := cache.db.Txn(false)
	defer transact.Abort()
	it, err := transact.Get("recipe", "id")
	if err != nil {
		return nil, err
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		recipe, ok := obj.(*Recipe)
		if ok {
			match, err := predicate(recipe)
			if err != nil {
				return nil, err
			} else if match {
				result = append(result, recipe)
			}
		}
	}

	return result, nil
}
