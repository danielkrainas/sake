package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/danielkrainas/gobag/util/token"
	"github.com/danielkrainas/gobag/util/uid"
	"github.com/danielkrainas/sake/pkg/api/v1"
	"github.com/danielkrainas/sake/pkg/service/protobuf"
	"github.com/danielkrainas/sake/pkg/util/log"
	"go.uber.org/zap"
)

type CoordinatorService interface {
	Component
	Register(recipe *Recipe) error
	UpdateExpired() error
	ClearInactive() error
	UnloadRecipe(name string) (bool, error)
}

type Coordinator struct {
	Hub            HubConnector
	Context        context.Context
	Cache          CacheService
	readyWaitGroup sync.WaitGroup
}

var _ CoordinatorService = &Coordinator{}

func NewCoordinator(ctx context.Context, hub HubConnector, cache CacheService, storage StorageService) (*Coordinator, error) {
	c := &Coordinator{
		Hub:     hub,
		Context: ctx,
		Cache:   cache,
	}

	c.readyWaitGroup.Add(1)
	log.Info("loading stored recipes")
	recipes, err := storage.LoadAllRecipes(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't load recipes: %v", err)
	}

	log.Info("registering recipes", zap.Int("count", len(recipes)))
	for _, recipe := range recipes {
		if err := c.Register(recipe); err != nil {
			return nil, fmt.Errorf("registering recipe %q failed: %v", recipe.Name, err)
		}
	}

	activeTransactions, err := storage.LoadActiveTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't load stored transactions: %v", err)
	}

	log.Info("active transactions restored", zap.Int("count", len(activeTransactions)))
	for _, trx := range activeTransactions {
		if err := c.load(trx); err != nil {
			return nil, fmt.Errorf("restoring transaction %s failed: %v", trx.ID, err)
		}
	}

	return c, nil
}

func (c *Coordinator) ComponentName() string {
	return "coordinator"
}

func (c *Coordinator) Run(ctx ComponentRunContext) error {
	defer func() {
		c.shutdown()
	}()

	c.readyWaitGroup.Done()
	<-ctx.QuitCh
	return nil
}

func (c *Coordinator) shutdown() error {
	if err := c.Hub.CancelAll(); err != nil {
		log.Error("failed to cancel all hub subscriptions", zap.Error(err))
	}

	// shutdown hub
	// clear cache
	return nil
}

func (c *Coordinator) UnloadRecipe(name string) (bool, error) {
	found := false
	recipes, err := c.Cache.FilterRecipes(c.Context, func(recipe *Recipe) (bool, error) {
		return recipe.Name == name && recipe.Status() == StatusActive, nil
	})

	if err != nil {
		return found, err
	} else if len(recipes) < 1 {
		return found, nil
	}

	recipe := recipes[0]
	found = true
	log.Info("draining recipe", zap.String("id", recipe.ID), RecipeField(recipe))
	if ok := recipe.SetStatusCond(StatusDraining, StatusActive); !ok {
		return found, v1.ErrorCodeRecipeMultiModify.WithArgs(recipe.Name)
	}

	if err := c.Hub.CancelGroup(recipe.ID); err != nil {
		return found, err
	}

	return found, c.Cache.PutRecipe(c.Context, recipe)
}

func (c *Coordinator) Register(recipe *Recipe) error {
	recipe.NumActiveTransactions = 0
	if recipe.ID == "" {
		recipe.ID = uid.Generate()
		recipe.SetStatus(StatusActive)
	}

	upgraded, err := c.UnloadRecipe(recipe.Name)
	if err != nil {
		return err
	}

	if err := c.Cache.PutRecipe(c.Context, recipe); err != nil {
		return err
	}

	if recipe.Status() == StatusActive {
		err = c.Hub.SubGroup(recipe.ID, RawGroup{
			recipe.TriggeredBy: c.createRecipeTriggerHandler(recipe),
		})

		if err == nil {
			if upgraded {
				log.Info("recipe upgraded", RecipeField(recipe))
			} else {
				log.Info("recipe registered", RecipeField(recipe))
			}
		}
	}

	return err
}

func (c *Coordinator) ClearInactive() error {
	wfs, err := c.Cache.FilterRecipes(c.Context, func(recipe *Recipe) (bool, error) {
		return recipe.Status() != StatusActive, nil
	})

	if err != nil {
		return err
	}

	for _, recipe := range wfs {
		if recipe.Status() == StatusDraining && atomic.LoadInt32(&recipe.NumActiveTransactions) < 1 {
			recipe.SetStatus(StatusInactive)
			log.Debug("recipe drained", RecipeField(recipe))
			if err := c.Cache.PutRecipe(c.Context, recipe); err != nil {
				return err
			}
		}

		log.Debug("unloading inactive recipe", RecipeField(recipe))
		if err := c.Cache.RemoveRecipe(c.Context, recipe); err != nil {
			return err
		}
	}

	return nil
}

func (c *Coordinator) UpdateExpired() error {
	n := 0
	err := c.Cache.TransactAll(c.Context, func(trx *Transaction) error {
		log.Debug("waiting on transaction lock", TransactionFields(trx)...)
		trx.Lock()
		defer func() {
			trx.Unlock()
			log.Debug("unlocked transaction", TransactionFields(trx)...)
		}()

		if trx.IsExpired() && trx.State == IsExecuting {
			log.Debug("transaction expired", TransactionFields(trx)...)
			if err := trx.Commit(false); err != nil {
				log.Error("couldn't commit expired transaction", zap.Error(err))
				return err
			}

			if err := c.transition(trx); err != nil {
				log.Error("couldn't transition expired transaction", zap.Error(err))
				return err
			}

			n++
		}

		return nil
	})

	log.Info("expired transactions updated", zap.Int("expired", n))
	return err
}

func (c *Coordinator) createRecipeTriggerHandler(recipe *Recipe) func([]byte) error {
	return func(data []byte) error {
		if recipe.Status() != StatusActive {
			log.Error("inactive recipe trigger handler still subscribed", RecipeField(recipe))
			return fmt.Errorf("recipe %q (id=%s) is inactive", recipe.Name, recipe.ID)
		}

		atomic.AddInt32(&recipe.NumActiveTransactions, 1)
		trx := NewTransaction(recipe, nil)
		log.Info("start transaction", log.Combine(RecipeField(recipe), TransactionFields(trx)...)...)
		trx.Lock()
		defer trx.Unlock()
		if err := c.transition(trx); err != nil {
			log.Error("transition failed", log.Combine(zap.Error(err), TransactionFields(trx)...)...)
			return fmt.Errorf("failed to transition transaction: %v", err)
		}

		return nil
	}
}

func (c *Coordinator) createTransactionSuccessHandler(trx *Transaction) func(*protocol.Reply) error {
	return func(reply *protocol.Reply) error {
		log.Info("stage success", TransactionFields(trx)...)
		trx.Lock()
		defer trx.Unlock()
		if reply.NewData != nil {
			log.Info("updating transaction data", TransactionFields(trx)...)
			trx.Data = reply.NewData
		}

		if err := trx.Commit(true); err != nil {
			log.Error("commit failed", log.Combine(zap.Error(err), TransactionFields(trx)...)...)
			return fmt.Errorf("failed to commit reply: %v", err)
		}

		if err := c.transition(trx); err != nil {
			log.Error("transition failed", log.Combine(zap.Error(err), TransactionFields(trx)...)...)
			return fmt.Errorf("failed to transition transaction: %v", err)
		}

		return nil
	}
}

func (c *Coordinator) createTransactionFailureHandler(trx *Transaction) func(*protocol.Reply) error {
	return func(reply *protocol.Reply) error {
		log.Info("stage failed", TransactionFields(trx)...)
		trx.Lock()
		defer trx.Unlock()
		if err := trx.Commit(false); err != nil {
			log.Error("commit failed", log.Combine(zap.Error(err), TransactionFields(trx)...)...)
			return fmt.Errorf("failed to commit reply: %v", err)
		}

		if err := c.transition(trx); err != nil {
			log.Error("transition failed", log.Combine(zap.Error(err), TransactionFields(trx)...)...)
			return fmt.Errorf("failed to transition transaction: %v", err)
		}

		return nil
	}
}

func (c *Coordinator) load(trx *Transaction) error {
	trx.Lock()
	defer trx.Unlock()
	return c.transition(trx)
}

func (c *Coordinator) unload(trx *Transaction) error {
	return c.Cache.RemoveTransaction(c.Context, trx)
}

func (c *Coordinator) transition(trx *Transaction) error {
	c.readyWaitGroup.Wait()
	if err := c.Cache.PutTransaction(c.Context, trx); err != nil {
		return fmt.Errorf("record transaction state failed: %v", err)
	}

	log.Info("record transaction", TransactionFields(trx)...)
	previousStep := trx.State
	trx.Step()
	log.Debug("step transaction", TransactionFields(trx, zap.String("prev_state", string(previousStep)))...)
	if !trx.IsCompleted() {
		successTopic := successReplyAddress(trx)
		failureTopic := failureReplyAddress(trx)
		req := &protocol.Request{
			ID:                token.Generate(),
			TransactionID:     trx.ID,
			SuccessReplyTopic: successTopic,
			FailureReplyTopic: failureTopic,
			Data:              trx.Data,
		}

		log.Debug("dispatch request", log.CombineAll([]zap.Field{zap.String("req", req.ID), zap.String("topic", trx.StageTopic)}, TransactionFields(trx))...)
		finalizer := c.createReplyFinalizer(trx, trx.StageTopic, req.ID)
		err := c.Hub.SubReply(req.ID, finalizer, ReplyGroup{
			successTopic: c.createTransactionSuccessHandler(trx),
			failureTopic: c.createTransactionFailureHandler(trx),
		})

		if err != nil {
			log.Error("failed to attach reply subscribers", zap.Error(err))
		}

		c.Hub.Pub(trx.StageTopic, req)
	} else {
		atomic.AddInt32(&trx.Recipe.NumActiveTransactions, -1)
		log.Info("completed transaction", TransactionFields(trx)...)
		if err := c.unload(trx); err != nil {
			return err
		}
	}

	return nil
}

func (c *Coordinator) createReplyFinalizer(trx *Transaction, stageTopic string, reqID string) func() {
	oncer := sync.Once{}
	return func() {
		oncer.Do(func() {
			for {
				if err := c.Hub.CancelGroup(reqID); err != nil {
					log.Error("failed to unsubscribe group", log.Combine(zap.Error(err), TransactionFields(trx, zap.String("topic", stageTopic))...)...)
					continue
				}

				break
			}
		})
	}
}

func successReplyAddress(trx *Transaction) string {
	return fmt.Sprintf("sake.reply.ok.%s@%s", trx.ID, trx.StageTopic)
}

func failureReplyAddress(trx *Transaction) string {
	return fmt.Sprintf("sake.reply.fail.%s@%s", trx.ID, trx.StageTopic)
}
