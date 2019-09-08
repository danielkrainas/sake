package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/danielkrainas/gobag/context"
	"github.com/danielkrainas/gobag/errcode"
	"github.com/danielkrainas/gobag/http"
	"github.com/danielkrainas/gobag/util/uid"
	"github.com/urfave/negroni"
	"go.uber.org/zap"

	"github.com/danielkrainas/sake/pkg/service"
	"github.com/danielkrainas/sake/pkg/util/log"
)

type ServerConfig struct {
	Addr string
}

func NewServer(ctx context.Context, mux http.Handler, cache service.CacheService, config ServerConfig) (*Server, error) {
	n := negroni.New()
	srv := &Server{
		config: config,
		server: &http.Server{
			Addr:    config.Addr,
			Handler: n,
		},
	}

	n.Use(&negroni.Recovery{
		Logger:     negroni.ALogger(bagcontext.GetLogger(ctx)),
		PrintStack: true,
		StackAll:   true,
	})

	n.Use(aliveHandler("/"))
	n.Use(contextHandler(cache))
	n.Use(loggingHandler())
	n.UseHandler(mux)

	return srv, nil
}

type Server struct {
	config ServerConfig
	server *http.Server
}

func (srv *Server) ComponentName() string {
	return "server"
}

func (srv *Server) Run(ctx service.ComponentRunContext) error {
	errch := make(chan error, 1)
	go func() {
		defer close(errch)
		if err := srv.ListenAndServe(); err != nil {
			errch <- err
		} else {
			errch <- nil
		}
	}()

	select {
	case err := <-errch:
		return err
	case <-ctx.QuitCh:
		tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.server.Shutdown(tctx); err != nil {
			return err
		}
	}

	return nil
}

func (srv *Server) ListenAndServe() error {
	config := srv.config
	ln, err := net.Listen("tcp", config.Addr)
	if err != nil {
		return err
	}

	log.Info("http server listening", zap.Stringer("address", ln.Addr()))
	return srv.server.Serve(ln)
}

func loggingHandler() negroni.Handler {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		rc := GetRequestContext(r)
		if rc != nil {
			w, iw := baghttp.NewInstrumentedResponseWriter(w)
			log.Info("request start", requestLogFields(rc, r)...)
			defer func() {
				info := iw.Info()
				if info.Status >= 200 && info.Status <= 399 {
					log.Info("response complete", responseLogFields(rc, info)...)
				} else if rc.Errors.Len() > 0 {
					if err := errcode.ServeJSON(w, rc.Errors); err != nil {
						log.Error("serve error json failed", zap.Error(err))
					}

					log.Info("response error", responseErrorLogFields(rc, info)...)
				}
			}()
		}

		next(w, r)
	})
}

func contextHandler(cache service.CacheService) negroni.Handler {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		rc := &RequestContext{
			Errors:    make(errcode.Errors, 0),
			Cache:     cache,
			RequestID: uid.Generate(),
			StartedAt: time.Now(),
		}

		next(w, AppendRequestContext(rc, r))
	})
}

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

func requestLogFields(rc *RequestContext, r *http.Request) []zap.Field {
	fields := []zap.Field{
		zap.Namespace("http.req"),
		zap.String("id", rc.RequestID),
		zap.String("method", r.Method),
		zap.String("host", r.Host),
		zap.String("uri", r.RequestURI),
		zap.String("useragent", r.UserAgent()),
		zap.String("remoteaddr", baghttp.RemoteAddr(r)),
	}

	if r.Referer() != "" {
		fields = append(fields, zap.String("referer", r.Referer()))
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "" {
		fields = append(fields, zap.String("contenttype", contentType))
	}

	return fields
}

func responseLogFields(rc *RequestContext, resp baghttp.ResponseInstrumentationInfo) []zap.Field {
	duration := time.Since(rc.StartedAt)
	fields := []zap.Field{
		zap.Namespace("http.req"),
		zap.String("id", rc.RequestID),
		zap.Namespace("http.res"),
		zap.Duration("duration", duration),
		zap.Int32("status", resp.Status),
	}

	if resp.ContentType != "" {
		fields = append(fields, zap.String("contenttype", resp.ContentType))
	}

	return fields
}

func responseErrorLogFields(rc *RequestContext, resp baghttp.ResponseInstrumentationInfo) []zap.Field {
	fields := []zap.Field{}
	for i, err := range rc.Errors {
		ns := "err"
		if rc.Errors.Len() > 1 {
			ns = fmt.Sprintf("err_%d", i)
		}

		fields = append(fields, zap.Namespace(ns))
		var detail interface{}
		code := fmt.Stringer(errcode.ErrorCodeUnknown)
		message := err.Error()

		switch e := err.(type) {
		case errcode.Error:
			code = e.Code
			message = e.Code.Message()
			detail = e.Detail

		case errcode.ErrorCode:
			code = e
			message = e.Message()
		}

		fields = append(fields, []zap.Field{
			zap.Stringer("code", code),
			zap.String("message", message),
		}...)

		if detail != nil {
			fields = append(fields, zap.Any("detail", detail))
		}
	}

	return append(fields, responseLogFields(rc, resp)...)
}
