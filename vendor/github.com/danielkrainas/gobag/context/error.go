package bagcontext

import (
	"context"

	"github.com/danielkrainas/gobag/errcode"
)

type errorsContext struct {
	context.Context
	errors errcode.Errors
}

func (ctx *errorsContext) Value(key interface{}) interface{} {
	if key == "errors" {
		return ctx.errors
	} else if key == "errors.ctx" {
		return ctx
	}

	return ctx.Context.Value(key)
}

func ErrorTracking(ctx context.Context) context.Context {
	return &errorsContext{
		Context: ctx,
		errors:  make(errcode.Errors, 0),
	}
}

func TrackError(ctx context.Context, err error) {
	ectx, ok := ctx.Value("errors.ctx").(*errorsContext)
	if ok {
		ectx.errors = append(ectx.errors, err)
	}
}

func WithErrors(ctx context.Context, errors errcode.Errors) context.Context {
	return WithValue(ctx, "errors", errors)
}

func AppendError(ctx context.Context, err error) context.Context {
	errors := GetErrors(ctx)
	errors = append(errors, err)
	return WithErrors(ctx, errors)
}

func GetErrors(ctx context.Context) errcode.Errors {
	if errors, ok := ctx.Value("errors").(errcode.Errors); errors != nil && ok {
		return errors
	}

	return make(errcode.Errors, 0)
}
