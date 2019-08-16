package service

import "go.uber.org/zap"

func WorkflowField(wf *Workflow) zap.Field {
	return zap.String("workflow", wf.Name)
}

func TransactionFields(trx *Transaction, others ...zap.Field) []zap.Field {
	return append([]zap.Field{zap.Namespace("transaction"), zap.String("id", trx.ID), zap.String("state", string(trx.State))}, others...)
}
