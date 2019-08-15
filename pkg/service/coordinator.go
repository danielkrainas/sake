package service

import (
	"context"
	"fmt"

	"github.com/danielkrainas/gobag/util/token"
	"github.com/danielkrainas/sake/pkg/service/protobuf"
)

type APIServer interface {
	ListenAndServe() error
}

type CoordinatorService interface {
	Register(wf *Workflow) error
	UpdateExpired() error
}

type Coordinator struct {
	Hub     HubConnector
	Context context.Context
	Cache   CacheService
}

var _ CoordinatorService = &Coordinator{}

func NewCoordinator(ctx context.Context, hub HubConnector, cache CacheService, storage StorageService) *Coordinator {
	c := &Coordinator{
		Hub:     hub,
		Context: ctx,
		Cache:   cache,
	}

	workflows, err := storage.LoadAllWorkflows(ctx)
	if err != nil {
		//
	}

	for _, wf := range workflows {
		if err := c.Register(wf); err != nil {
			//
		}
	}

	activeTransactions, err := storage.LoadActiveTransactions(ctx)
	if err != nil {
		//
	}

	for _, trx := range activeTransactions {
		if err := c.load(trx); err != nil {
			//
		}
	}

	return c
}

func (c *Coordinator) Register(wf *Workflow) error {
	if wf, err := c.Cache.GetWorkflow(c.Context, wf.Name); err != nil {
		//
	} else if wf != nil {
		//
	}

	if err := c.Cache.PutWorkflow(c.Context, wf); err != nil {
		//
	}

	c.Hub.Sub(wf.TriggeredBy, c.createWorkflowTriggerHandler(wf))
	return nil
}

func (c *Coordinator) UpdateExpired() error {
	return c.Cache.TransactAll(c.Context, func(trx *Transaction) (*Transaction, error) {
		trx.Lock()
		defer trx.Unlock()

		if trx.IsExpired() && trx.State == IsExecuting {
			if err := trx.Commit(false); err != nil {
				//
			}

			if err := c.transition(trx); err != nil {
				//
			}
		}

		return trx, nil
	})
}

func (c *Coordinator) createWorkflowTriggerHandler(wf *Workflow) func([]byte) {
	return func(data []byte) {
		trx := NewTransaction(wf, nil)
		trx.Lock()
		defer trx.Unlock()
		if err := c.Cache.PutTransaction(c.Context, trx); err != nil {
			//
		}

		if err := c.transition(trx); err != nil {
			//
		}
	}
}

func (c *Coordinator) createTransactionSuccessHandler(trx *Transaction) func(*protocol.Reply) {
	return func(reply *protocol.Reply) {
		trx.Lock()
		defer trx.Unlock()
		if reply.NewData != nil {
			// log
			trx.Data = reply.NewData
		}

		if err := trx.Commit(true); err != nil {
			//
		}

		if err := c.transition(trx); err != nil {
			//
		}
	}
}

func (c *Coordinator) createTransactionFailureHandler(trx *Transaction) func(*protocol.Reply) {
	return func(reply *protocol.Reply) {
		trx.Lock()
		defer trx.Unlock()
		if err := trx.Commit(false); err != nil {
			//
		}

		if err := c.transition(trx); err != nil {
			//
		}
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
	if err := c.Cache.PutTransaction(c.Context, trx); err != nil {
		// log
		return err
	}

	trx.Step()
	if err := c.Hub.CancelGroup(trx); err != nil {
		//
		return err
	}

	fmt.Printf("#%s state=%s\n", trx.ID, trx.State)
	if !trx.IsCompleted() {
		fmt.Printf("#%s activity=%s\n", trx.ID, trx.StageTopic)
		successTopic := successReplyAddress(trx)
		failureTopic := failureReplyAddress(trx)
		c.Hub.SubReply(trx, successTopic, c.createTransactionSuccessHandler(trx))
		c.Hub.SubReply(trx, failureTopic, c.createTransactionFailureHandler(trx))

		req := &protocol.Request{
			ID:                token.Generate(),
			TransactionID:     trx.ID,
			SuccessReplyTopic: successTopic,
			FailureReplyTopic: failureTopic,
			Data:              trx.Data,
		}

		c.Hub.Pub(trx.StageTopic, req)
	} else {
		fmt.Printf("#%s completed with state=%s\n", trx.ID, trx.State)
		if err := c.unload(trx); err != nil {
			return err
		}
	}

	return nil
}

func successReplyAddress(trx *Transaction) string {
	return fmt.Sprintf("success/%s@%s", trx.ID, trx.StageTopic)
}

func failureReplyAddress(trx *Transaction) string {
	return fmt.Sprintf("failed/%s@%s", trx.ID, trx.StageTopic)
}
