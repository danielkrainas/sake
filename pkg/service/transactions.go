package service

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/danielkrainas/gobag/util/token"
)

type Stage struct {
	Next            string        `json:"next"`
	Rollback        string        `json:"rollback"`
	RollbackTimeout time.Duration `json:"rollback_timeout,omitempty"`
	Timeout         time.Duration `json:"timeout,omitempty"`
	Terminate       bool          `json:"terminate,omitempty"`
}

type WorkflowStatus int32

const (
	StatusInactive WorkflowStatus = 0
	StatusActive                  = 1
	StatusDraining                = 2
)

type Workflow struct {
	ID                    string            `json:"id"`
	Name                  string            `json:"name"`
	TriggeredBy           string            `json:"trigger"`
	StartAt               string            `json:"start"`
	Stages                map[string]*Stage `json:"stages"`
	NumActiveTransactions int32             `json:"num_active_transactions"`
	StatusCode            int32             `json:"status"`
}

func (wf *Workflow) SetStatus(status WorkflowStatus) {
	atomic.StoreInt32(&wf.StatusCode, int32(status))
}

func (wf *Workflow) SetStatusCond(newStatus WorkflowStatus, currentStatus WorkflowStatus) bool {
	return atomic.CompareAndSwapInt32(&wf.StatusCode, int32(currentStatus), int32(newStatus))
}

func (wf *Workflow) Status() WorkflowStatus {
	return WorkflowStatus(atomic.LoadInt32(&wf.StatusCode))
}

type TransactionState string

const (
	IsReverting    TransactionState = "reverting"
	IsExecuting                     = "executing"
	IsInitializing                  = "initializing"
	IsSuccess                       = "success"
	IsFailed                        = "failed"
)

type Transaction struct {
	sync.Mutex
	ID           string
	State        TransactionState
	Data         []byte
	Stage        *Stage
	StageKey     string
	StageTopic   string
	StageStarted time.Time
	Started      time.Time
	Expires      *time.Time
	Workflow     *Workflow
	ExecutedPath []string
}

func NewTransaction(wf *Workflow, data []byte) *Transaction {
	trx := &Transaction{
		ID:           token.Generate(),
		State:        IsInitializing,
		Data:         data,
		Stage:        nil,
		StageKey:     "",
		StageTopic:   "",
		StageStarted: time.Unix(0, 0),
		Started:      time.Now(),
		Expires:      nil,
		Workflow:     wf,
	}

	return trx
}

func (trx *Transaction) Commit(success bool) error {
	if trx.IsCompleted() {
		return errors.New("transaction is completed")
	}

	if !success && trx.State == IsExecuting {
		trx.State = IsReverting
	}

	return nil
}

func (trx *Transaction) Step() {
	var stageKey string
	done := false
	if trx.State == IsInitializing {
		stageKey = trx.Workflow.StartAt
		trx.State = IsExecuting
		trx.ExecutedPath = []string{stageKey}
	} else if trx.State == IsExecuting {
		if trx.Stage.Terminate {
			done = true
		} else {
			stageKey = trx.Stage.Next
			trx.ExecutedPath = append(trx.ExecutedPath, stageKey)
		}
	} else if trx.State == IsReverting {
		if len(trx.ExecutedPath) < 1 {
			done = true
		} else {
			stageKey = trx.ExecutedPath[len(trx.ExecutedPath)-1]
			trx.ExecutedPath = trx.ExecutedPath[:len(trx.ExecutedPath)-1]
		}
	}

	if !done {
		stage := trx.Workflow.Stages[stageKey]
		trx.StageKey = stageKey
		trx.Stage = stage
		trx.StageTopic = stageKey
		trx.StageStarted = time.Now()
		if stage != nil {
			if trx.State == IsReverting && stage.Rollback == "" {
				trx.Step()
				return
			} else {
				if trx.State == IsReverting {
					trx.StageTopic = stage.Rollback
				}

				trx.SetTimeout(stage.Timeout)
			}
		}
	} else {
		if trx.State == IsExecuting {
			trx.State = IsSuccess
		} else {
			trx.State = IsFailed
		}
	}
}

func (trx *Transaction) SetTimeout(d time.Duration) {
	if d > 0 {
		expires := time.Now().Add(d)
		trx.Expires = &expires
	} else {
		trx.Expires = nil
	}
}

func (trx *Transaction) IsExpired() bool {
	if trx.Expires != nil {
		return trx.Expires.Before(time.Now())
	}

	return false
}

func (trx *Transaction) IsCompleted() bool {
	return trx.State == IsFailed || trx.State == IsSuccess
}
