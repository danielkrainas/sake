package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/danielkrainas/gobag/util/token"
	"github.com/danielkrainas/sake/pkg/service/protobuf"
)

type APIServer interface {
	ListenAndServe() error
}

type CoordinatorService interface {
	Register(wf *Workflow)
	UpdateExpired() error
}

type Coordinator struct {
	Hub          HubConnector
	Context      context.Context
	transactions map[string]*Transaction
	trxMutex     sync.Mutex
	wfMutex      sync.Mutex
	workflows    map[string]*Workflow
}

func NewCoordinator(ctx context.Context, hub HubConnector) *Coordinator {
	return &Coordinator{
		Hub:          hub,
		Context:      ctx,
		transactions: make(map[string]*Transaction),
		workflows:    make(map[string]*Workflow),
	}
}

func (c *Coordinator) Register(wf *Workflow) {
	c.wfMutex.Lock()
	defer c.wfMutex.Unlock()
	c.workflows[wf.Name] = wf
	c.Hub.Sub(wf.TriggeredBy, c.createWorkflowTriggerHandler(wf))
}

func (c *Coordinator) UpdateExpired() error {
	c.trxMutex.Lock()
	defer c.trxMutex.Unlock()
	for _, trx := range c.transactions {
		func(trx *Transaction) {
			trx.Lock()
			defer trx.Unlock()

			if trx.IsExpired() && trx.State == IsExecuting {
				trx.Commit(false)
				c.transition(trx)
			}
		}(trx)
	}

	return nil
}

func (c *Coordinator) createWorkflowTriggerHandler(wf *Workflow) func([]byte) {
	return func(data []byte) {
		trx := NewTransaction(wf, nil)
		c.trxMutex.Lock()
		defer c.trxMutex.Unlock()
		c.transactions[trx.ID] = trx
		c.transition(trx)
	}
}

func (c *Coordinator) createTransactionSuccessHandler(trx *Transaction) func(*protocol.Reply) {
	return func(reply *protocol.Reply) {
		trx.Lock()
		defer trx.Unlock()
		trx.Data = reply.NewData
		trx.Commit(true)
		if err := c.transition(trx); err != nil {
			//
		}
	}
}

func (c *Coordinator) createTransactionFailureHandler(trx *Transaction) func(*protocol.Reply) {
	return func(reply *protocol.Reply) {
		trx.Lock()
		defer trx.Unlock()
		trx.Commit(false)
		if err := c.transition(trx); err != nil {
			//
		}
	}
}

func (c *Coordinator) unload(trx *Transaction) {
	c.trxMutex.Lock()
	defer c.trxMutex.Unlock()
	delete(c.transactions, trx.ID)
}

func (c *Coordinator) recordState(trx *Transaction) {
	// noop
}

func (c *Coordinator) findTransaction(id string) (*Transaction, error) {
	c.trxMutex.Lock()
	defer c.trxMutex.Unlock()
	trx, ok := c.transactions[id]
	if !ok {
		return nil, errors.New("transaction not found")
	}

	return trx, nil
}

func (c *Coordinator) transition(trx *Transaction) error {
	c.recordState(trx)
	trx.Step()
	c.Hub.CancelGroup(trx)
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
		c.unload(trx)
	}

	return nil
}

func successReplyAddress(trx *Transaction) string {
	return fmt.Sprintf("success/%s@%s", trx.ID, trx.StageTopic)
}

func failureReplyAddress(trx *Transaction) string {
	return fmt.Sprintf("failed/%s@%s", trx.ID, trx.StageTopic)
}
