package api

import (
	"fmt"
	"net/http"

	"github.com/danielkrainas/gobag/context"
	"github.com/danielkrainas/sake/pkg/util/log"
	gmux "github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/danielkrainas/sake/pkg/api/v1"
)

type HttpHandler func(rc *RequestContext, w http.ResponseWriter, r *http.Request)

type Mux struct {
	router *gmux.Router
}

func NewMux() (*Mux, error) {
	api := &Mux{
		router: v1.RouterWithPrefix(""),
	}

	api.register(v1.RouteNameBase, http.HandlerFunc(baseHandler))
	mappings := map[string]func() HttpHandler{
		v1.RouteNameRecipes: RecipesAPI,
		v1.RouteNameRecipe:  RecipeAPI,
	}

	for routeName, dispatchFactory := range mappings {
		api.register(routeName, dispatchFactory())
	}

	return api, nil
}

func (api *Mux) register(routeName string, dispatch interface{}) {
	api.router.GetRoute(routeName).Handler(api.dispatcher(dispatch))
}

func (api *Mux) dispatcher(dispatch interface{}) http.Handler {
	if httpDispatch, ok := dispatch.(http.Handler); ok {
		// NOTE: a spot to wrap HTTP handler logic here
		return httpDispatch
	}

	// NOTE: a spot to add logic and decorate the route's http handler
	if handlerDispatch, ok := dispatch.(HttpHandler); ok {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerDispatch(GetRequestContext(r), w, r)
		})
	}

	log.Fatal("invalid dispatch handler type", zap.String("dispatch", fmt.Sprintf("%T", dispatch)))
	return nil
}

func (api *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Header().Add(v1.VersionHeader.Name, bagcontext.GetVersion(r.Context()))
	api.router.ServeHTTP(w, r)
}

func MethodRouter(methodHandlers map[string]HttpHandler) HttpHandler {
	return HttpHandler(func(rc *RequestContext, w http.ResponseWriter, r *http.Request) {
		if handler, ok := methodHandlers[r.Method]; ok {
			handler(rc, w, r)
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
