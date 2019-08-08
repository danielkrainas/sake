package api

import (
	"context"
	"net/http"

	"github.com/danielkrainas/gobag/context"
	"github.com/danielkrainas/gobag/errcode"
	"github.com/danielkrainas/gobag/log"
	"github.com/urfave/negroni"
)

func aliveHandler(path string) negroni.Handler {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if r.URL.Path == path {
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	})
}

func contextHandler(parent context.Context) negroni.Handler {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		var iw http.ResponseWriter
		ctx := parent
		ctx = bagcontext.WithRequest(parent, r)
		ctx, iw = bagcontext.WithResponseWriter(ctx, w)
		ctx = bagcontext.WithVars(ctx, r)
		ctx = bagcontext.WithLogger(ctx, bagcontext.GetLogger(ctx))
		next(iw, r.WithContext(ctx))
	})
}

func loggingHandler(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	ctx := r.Context()
	bagcontext.GetRequestLogger(ctx).Info("request started")
	defer func() {
		status, ok := ctx.Value("http.response.status").(int)
		if ok && status >= 200 && status <= 399 {
			bagcontext.GetResponseLogger(ctx).Infof("response completed")
		}
	}()

	next(w, r)
}

func trackErrorsHandler(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	ctx := bagcontext.ErrorTracking(r.Context())
	next(w, r.WithContext(ctx))
	if errors := bagcontext.GetErrors(ctx); errors.Len() > 0 {
		if err := errcode.ServeJSON(w, errors); err != nil {
			log.Error(ctx, "error serving error json: %v (from %s)", err, errors)
		}

		logErrors(ctx, errors)
	}
}

func logErrors(ctx context.Context, errors errcode.Errors) {
	for _, err := range errors {
		var lctx context.Context

		switch err.(type) {
		case errcode.Error:
			e, _ := err.(errcode.Error)
			lctx = bagcontext.WithValue(ctx, "err.code", e.Code)
			lctx = bagcontext.WithValue(lctx, "err.message", e.Code.Message())
			lctx = bagcontext.WithValue(lctx, "err.detail", e.Detail)
		case errcode.ErrorCode:
			e, _ := err.(errcode.ErrorCode)
			lctx = bagcontext.WithValue(ctx, "err.code", e)
			lctx = bagcontext.WithValue(lctx, "err.message", e.Message())
		default:
			// normal "error"
			lctx = bagcontext.WithValue(ctx, "err.code", errcode.ErrorCodeUnknown)
			lctx = bagcontext.WithValue(lctx, "err.message", err.Error())
		}

		lctx = bagcontext.WithLogger(ctx, bagcontext.GetLogger(lctx,
			"err.code",
			"err.message",
			"err.detail"))

		bagcontext.GetResponseLogger(lctx).Errorf("response completed with error")
	}
}
