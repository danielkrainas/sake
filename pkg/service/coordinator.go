package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
	c.Hub.Subscribe(wf.TriggeredBy, c.createWorkflowTriggerHandler(wf))
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

func (c *Coordinator) createWorkflowTriggerHandler(wf *Workflow) func(e *Envelope) {
	return func(e *Envelope) {
		trx := NewTransaction(wf, []byte("unmodified"))
		c.trxMutex.Lock()
		defer c.trxMutex.Unlock()
		c.transactions[trx.ID] = trx
		c.transition(trx)
	}
}

func (c *Coordinator) handleTransactionSuccessReply(e *Envelope) {
	trx, err := c.findTransaction(e.TransactionID)
	trx.Lock()
	defer trx.Unlock()
	trx.Data = e.Data
	trx.Commit(true)
	err = c.transition(trx)
	if err != nil {
	}
}

func (c *Coordinator) handleTransactionFailureReply(e *Envelope) {
	trx, err := c.findTransaction(e.TransactionID)
	trx.Lock()
	defer trx.Unlock()
	trx.Commit(false)
	err = c.transition(trx)
	if err != nil {

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
		fmt.Printf("#%s activity=%s\n", trx.ID, trx.StageAddress)
		successAddress := successReplyAddress(trx)
		failureAddress := failureReplyAddress(trx)
		c.Hub.GroupSubscribe(trx, successAddress, c.handleTransactionSuccessReply)
		c.Hub.GroupSubscribe(trx, failureAddress, c.handleTransactionFailureReply)

		e := &Envelope{
			TransactionID:       trx.ID,
			SuccessReplyAddress: successAddress,
			FailureReplyAddress: failureAddress,
			Data:                trx.Data,
		}

		c.Hub.Publish(trx.StageAddress, e)
	} else {
		fmt.Printf("#%s completed with state=%s\n", trx.ID, trx.State)
		c.unload(trx)
	}

	return nil
}

func successReplyAddress(trx *Transaction) string {
	return fmt.Sprintf("success/%s@%s", trx.ID, trx.StageAddress)
}

func failureReplyAddress(trx *Transaction) string {
	return fmt.Sprintf("failed/%s@%s", trx.ID, trx.StageAddress)
}
