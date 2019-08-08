package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/danielkrainas/gobag/context"
	"github.com/danielkrainas/gobag/log"
	gmux "github.com/gorilla/mux"

	"github.com/danielkrainas/sake/pkg/api/v1"
)

type Handler func(req *RequestContext, resp *Responder)

type Mux struct {
	router *gmux.Router
}

func NewMux() (*Mux, error) {
	api := &Mux{
		router: v1.RouterWithPrefix(""),
	}

	api.registerHTTP(v1.RouteNameBase, http.HandlerFunc(baseHandler))
	mappings := map[string]func() Handler{}

	for routeName, dispatchFactory := range mappings {
		api.register(routeName, dispatchFactory())
	}

	return api, nil
}

func (api *Mux) register(routeName string, dispatch Handler) {
	api.router.GetRoute(routeName).Handler(api.dispatcher(dispatch))
}

func (api *Mux) registerHTTP(routeName string, dispatch http.Handler) {
	api.router.GetRoute(routeName).Handler(api.dispatcher(dispatch))
}

func (api *Mux) dispatcher(dispatch interface{}) http.Handler {
	if httpDispatch, ok := dispatch.(http.Handler); ok {
		// NOTE: a spot to wrap HTTP handler logic here
		return httpDispatch
	}

	if domainDispatch, ok := dispatch.(Handler); ok {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			req := &RequestContext{
				HTTP: r,
			}

			res := &Responder{
				Context: req.Context(),
				HTTP:    w,
			}

			domainDispatch(req, res)
		})
	}

	log.Fatal(context.Background(), "invalid dispatch handler type %T", dispatch)
	return nil
}

func (api *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// NOTE: a spot to add logic before and after the request is routed by the API
	defer r.Body.Close()

	w.Header().Add(v1.VersionHeader.Name, bagcontext.GetVersion(r.Context()))
	api.router.ServeHTTP(w, r)
}

type OptionsValidator interface {
	Validate() error
}

type RequestContext struct {
	HTTP *http.Request
}

func (r *RequestContext) readRequestOptions(options interface{}) error {
	err := json.NewDecoder(r.HTTP.Body).Decode(options)
	if err != nil {
		log.Error(r.Context(), "error reading body: %v", err)
		return err
	}

	return nil
}

func (r *RequestContext) Context() context.Context {
	if r.HTTP == nil {
		return nil
	}

	return r.HTTP.Context()
}

func (r *RequestContext) Parse(options interface{}) bool {
	err := r.readRequestOptions(options)
	if err != nil {
		bagcontext.TrackError(r.Context(), v1.ErrorCodeRequestInvalid.WithDetail(err))
		return false
	}

	return true
}

func (r *RequestContext) Validate(validator interface{}) bool {
	if v, ok := validator.(OptionsValidator); ok {
		if err := v.Validate(); err != nil {
			bagcontext.TrackError(r.Context(), v1.ErrorCodeRequestInvalid.WithDetail(err))
			return false
		}
	}

	return true
}

func (r *RequestContext) ParseAndValidate(options interface{}) bool {
	return r.Parse(options) && r.Validate(options)
}

type Responder struct {
	Context context.Context
	HTTP    http.ResponseWriter
}

func (responder *Responder) Send(data interface{}) {
	responder.HTTP.Header().Set("Content-Type", "application/json; charset=utf-8")
	responder.HTTP.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(responder.HTTP).Encode(data); err != nil {
		log.Error(responder.Context, "error encoding json result: %v", err)
	}
}

func (responder *Responder) Error(err error) {
	bagcontext.TrackError(responder.Context, err)
}

func MethodRouter(methodHandlers map[string]Handler) Handler {
	return Handler(func(req *RequestContext, res *Responder) {
		handler, ok := methodHandlers[req.HTTP.Method]
		if ok {
			handler(req, res)
		}
	})
}

func baseHandler(w http.ResponseWriter, r *http.Request) {
	const emptyJSON = "{}"

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprint(len(emptyJSON)))
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, emptyJSON)
}
