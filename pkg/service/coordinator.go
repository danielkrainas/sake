package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/danielkrainas/gobag/util/token"
	"github.com/danielkrainas/sake/pkg/service/protobuf"
	"github.com/danielkrainas/sake/pkg/util/log"
	"go.uber.org/zap"
)

type APIServer interface {
	ListenAndServe() error
}

type CoordinatorService interface {
	Component
	Register(wf *Workflow) error
	UpdateExpired() error
}

type Coordinator struct {
	Hub               HubConnector
	Context           context.Context
	Cache             CacheService
	readyWaitGroup    sync.WaitGroup
	shutdownWaitGroup sync.WaitGroup
}

var _ CoordinatorService = &Coordinator{}

func NewCoordinator(ctx context.Context, hub HubConnector, cache CacheService, storage StorageService) (*Coordinator, error) {
	c := &Coordinator{
		Hub:     hub,
		Context: ctx,
		Cache:   cache,
	}

	c.readyWaitGroup.Add(1)
	log.Info("loading stored workflows")
	workflows, err := storage.LoadAllWorkflows(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't load workflows: %v", err)
	}

	log.Info("registering workflows", zap.Int("count", len(workflows)))
	for _, wf := range workflows {
		if err := c.Register(wf); err != nil {
			return nil, fmt.Errorf("registering workflow %q failed: %v", wf.Name, err)
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

func (c *Coordinator) Register(wf *Workflow) error {
	if wf, err := c.Cache.GetWorkflow(c.Context, wf.Name); err != nil {
		return err
	} else if wf != nil {
		return fmt.Errorf("workflow with name %q already registered", wf.Name)
	}

	if err := c.Cache.PutWorkflow(c.Context, wf); err != nil {
		return err
	}

	log.Info("registered workflow", zap.String("workflow", wf.Name))
	c.Hub.Sub(wf.TriggeredBy, c.createWorkflowTriggerHandler(wf))
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

func (c *Coordinator) createWorkflowTriggerHandler(wf *Workflow) func([]byte) error {
	return func(data []byte) error {
		trx := NewTransaction(wf, nil)
		log.Info("start transaction", log.Combine(WorkflowField(wf), TransactionFields(trx)...)...)
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
