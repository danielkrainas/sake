package service

import (
	"context"

	"github.com/danielkrainas/sake/pkg/util/log"
	memdb "github.com/hashicorp/go-memdb"
)

type StorageService interface {
	SaveTransaction(ctx context.Context, trx *Transaction) error
	SaveRecipe(ctx context.Context, recipe *Recipe) error
	RemoveRecipe(ctx context.Context, recipe *Recipe) error
	LoadAllRecipes(ctx context.Context) ([]*Recipe, error)
	LoadActiveTransactions(ctx context.Context) ([]*Transaction, error)
}

type DebugStorage struct {
	db *memdb.MemDB
}

var _ StorageService = &DebugStorage{}

func NewDebugStorage(recipes []*Recipe, transactions []*Transaction) (*DebugStorage, error) {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"recipe": &memdb.TableSchema{
				Name: "recipe",
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

	storage := &DebugStorage{
		db: db,
	}

	txn := db.Txn(true)
	for _, wf := range recipes {
		log.Info("pre-inserting recipe", RecipeField(wf))
		if err := txn.Insert("recipe", wf); err != nil {
			return nil, err
		}
	}

	for _, trx := range transactions {
		log.Info("pre-inserting transaction", TransactionFields(trx)...)
		if err := txn.Insert("transaction", trx); err != nil {
			return nil, err
		}
	}

	txn.Commit()
	log.Info("in-memory storage ready")
	return storage, nil
}

func (storage *DebugStorage) SaveTransaction(ctx context.Context, trx *Transaction) error {
	transact := storage.db.Txn(true)
	if err := transact.Insert("transaction", trx); err != nil {
		transact.Abort()
		return err
	}

	transact.Commit()
	return nil
}

func (storage *DebugStorage) SaveRecipe(ctx context.Context, recipe *Recipe) error {
	transact := storage.db.Txn(true)
	if err := transact.Insert("recipe", recipe); err != nil {
		transact.Abort()
		return err
	}

	transact.Commit()
	return nil
}

func (storage *DebugStorage) LoadAllRecipes(ctx context.Context) ([]*Recipe, error) {
	result := make([]*Recipe, 0)
	transact := storage.db.Txn(false)
	it, err := transact.Get("recipe", "id")
	if err != nil {
		transact.Abort()
		return nil, err
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		wf := obj.(*Recipe)
		result = append(result, wf)
	}

	return result, nil
}

func (storage *DebugStorage) LoadActiveTransactions(ctx context.Context) ([]*Transaction, error) {
	transact := storage.db.Txn(false)
	it, err := transact.Get("transaction", "id")
	if err != nil {
		transact.Abort()
		return nil, err
	}

	result := make([]*Transaction, 0)
	for obj := it.Next(); obj != nil; obj = it.Next() {
		trx := obj.(*Transaction)
		if !trx.IsCompleted() {
			result = append(result, trx)
		}
	}

	return result, nil
}

func (storage *DebugStorage) RemoveRecipe(ctx context.Context, recipe *Recipe) error {
	transact := storage.db.Txn(true)
	iwf, err := transact.First("recipe", "id", recipe.Name)
	if err == nil && iwf != nil {
		err = transact.Delete("recipe", iwf)
	}

	if err != nil {
		transact.Abort()
		return err
	}

	transact.Commit()
	return nil
}
