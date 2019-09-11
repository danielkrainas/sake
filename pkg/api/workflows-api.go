package api

import (
	"net/http"

	"github.com/danielkrainas/sake/pkg/api/v1"
	"github.com/danielkrainas/sake/pkg/service"
	"github.com/gorilla/mux"
)

func RecipesAPI() HttpHandler {
	return MethodRouter(map[string]HttpHandler{
		http.MethodGet:  GetAllRecipes,
		http.MethodPost: CreateRecipe,
	})
}

func RecipeAPI() HttpHandler {
	return MethodRouter(map[string]HttpHandler{
		http.MethodDelete: RemoveRecipe,
	})
}

func GetAllRecipes(ctx *RequestContext, w http.ResponseWriter, r *http.Request) {
	wfs, err := ctx.Cache.GetAllRecipes(r.Context())
	if err != nil {
		SendError(ctx, err)
	} else {
		SendJSON(w, wfs)
	}
}

func CreateRecipe(ctx *RequestContext, w http.ResponseWriter, r *http.Request) {
	wf := &service.Recipe{}
	if !ParseAndValidate(ctx, r, wf) {
		return
	}

	if err := ctx.Coordinator.Register(wf); err != nil {
		SendError(ctx, err)
	} else {
		SendJSON(w, wf)
	}
}

func RemoveRecipe(ctx *RequestContext, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name, ok := vars["name"]
	if !ok || name == "" {
		SendError(ctx, v1.ErrorCodeRequestInvalid.WithDetail("url name parameter missing or invalid"))
	} else {
		if _, err := ctx.Coordinator.UnloadRecipe(name); err != nil {
			SendError(ctx, err)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
