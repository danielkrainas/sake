package cqrs

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrNoHandler = errors.New("no command handler")

type Command interface{}

type CommandHandler interface {
	Handle(ctx context.Context, cmd Command) error
}

type commandHandler struct {
	f func(context.Context, Command) error
}

func (h *commandHandler) Handle(ctx context.Context, cmd Command) error {
	return h.f(ctx, cmd)
}

func CommandFunc(f func(context.Context, Command) error) CommandHandler {
	return &commandHandler{f}
}

type CommandDispatcher struct {
	Handlers []CommandHandler
}

func (d *CommandDispatcher) Dispatch(ctx context.Context, cmd Command) error {
	for _, h := range d.Handlers {
		if err := h.Handle(ctx, cmd); err != nil && err != ErrNoHandler {
			return err
		} else if err == nil {
			return nil
		}
	}

	return ErrNoHandler
}

type CommandRouter map[string]CommandHandler

func getCommandKey(c Command) string {
	return fmt.Sprintf("%T", c)
}

func (r CommandRouter) Register(c Command, handler CommandHandler) {
	r[getCommandKey(c)] = handler
}

func (r CommandRouter) Handle(ctx context.Context, c Command) error {
	h, ok := r[getCommandKey(c)]
	if !ok {
		return ErrNoHandler
	}

	return h.Handle(ctx, c)
}

func WithCommandDispatch(ctx context.Context, d *CommandDispatcher) context.Context {
	return context.WithValue(ctx, "cmd.dispatcher", d)
}

func DispatchCommand(ctx context.Context, c Command) error {
	d, ok := ctx.Value("cmd.dispatcher").(*CommandDispatcher)
	if !ok || d == nil {
		return fmt.Errorf("no valid command dispatchers found in context")
	}

	return d.Dispatch(ctx, c)
}

type RetryHandler struct {
	Retries int
	Delay   time.Duration
	Inner   CommandHandler
}

func (h *RetryHandler) Handle(ctx context.Context, cmd Command) error {
	var err error
	for i := 0; i < h.Retries; i++ {
		if err = h.Inner.Handle(ctx, cmd); err == nil {
			break
		}

		time.Sleep(h.Delay)
	}

	return err
}
