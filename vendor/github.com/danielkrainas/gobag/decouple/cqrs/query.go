package cqrs

import (
	"context"
	"errors"
	"fmt"
)

var ErrNoExecutor = errors.New("query was unhandled or invalid")

type Query interface{}

type QueryExecutor interface {
	Execute(ctx context.Context, q Query) (interface{}, error)
}

type QueryDispatcher struct {
	Executors []QueryExecutor
}

func (d *QueryDispatcher) Dispatch(ctx context.Context, q Query) (interface{}, error) {
	for _, e := range d.Executors {
		if result, err := e.Execute(ctx, q); err != nil && err != ErrNoExecutor {
			return nil, err
		} else if err == nil {
			return result, nil
		}
	}

	return nil, ErrNoExecutor
}

type QueryRouter map[string]QueryExecutor

func getQueryKey(q Query) string {
	return fmt.Sprintf("%T", q)
}

func (r QueryRouter) Register(q Query, exec QueryExecutor) {
	r[getQueryKey(q)] = exec
}

func (r QueryRouter) Execute(ctx context.Context, q Query) (interface{}, error) {
	exec, ok := r[getQueryKey(q)]
	if !ok {
		return nil, ErrNoExecutor
	}

	return exec.Execute(ctx, q)
}

func WithQueryDispatch(ctx context.Context, d *QueryDispatcher) context.Context {
	return context.WithValue(ctx, "query.dispatcher", d)
}

func DispatchQuery(ctx context.Context, q Query) (interface{}, error) {
	d, ok := ctx.Value("query.dispatcher").(*QueryDispatcher)
	if !ok || d == nil {
		return nil, fmt.Errorf("no valid query dispatchers found in context")
	}

	return d.Dispatch(ctx, q)
}
