package api

import (
	"context"
	"net"
	"net/http"

	"github.com/danielkrainas/gobag/context"
	"github.com/danielkrainas/gobag/log"
	"github.com/urfave/negroni"

	"github.com/danielkrainas/sake/pkg/api/v1"
)

type ServerConfig struct {
	Addr string
}

func NewServer(ctx context.Context, mux http.Handler, config ServerConfig) (srv *Server, err error) {
	n := negroni.New()
	n.Use(contextHandler(ctx))
	n.UseFunc(loggingHandler)
	n.Use(&negroni.Recovery{
		Logger:     negroni.ALogger(bagcontext.GetLogger(ctx)),
		PrintStack: true,
		StackAll:   true,
	})

	n.Use(aliveHandler("/"))
	n.UseFunc(trackErrorsHandler)
	n.UseHandler(mux)

	srv = &Server{
		Context: ctx,
		config:  config,
		handler: n,
	}

	srv.server = &http.Server{
		Addr:    config.Addr,
		Handler: srv,
	}

	return srv, nil
}

type Server struct {
	context.Context

	config  ServerConfig
	server  *http.Server
	handler http.Handler
}

func (srv *Server) ListenAndServe() error {
	config := srv.config
	ln, err := net.Listen("tcp", config.Addr)
	if err != nil {
		return err
	}

	log.Info(srv, "listening on %v", ln.Addr())
	return srv.server.Serve(ln)
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// NOTE: a spot to add logic before and after the request is routed by the API
	defer r.Body.Close()

	w.Header().Add(v1.VersionHeader.Name, bagcontext.GetVersion(r.Context()))
	srv.handler.ServeHTTP(w, r)
}
