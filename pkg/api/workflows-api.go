package api

import (
	"net/http"

	"github.com/danielkrainas/sake/pkg/api/v1"
	"github.com/danielkrainas/sake/pkg/service"
	"github.com/gorilla/mux"
)

func WorkflowsAPI() HttpHandler {
	return MethodRouter(map[string]HttpHandler{
		http.MethodGet:  GetAllWorkflows,
		http.MethodPost: CreateWorkflow,
	})
}

func WorkflowAPI() HttpHandler {
	return MethodRouter(map[string]HttpHandler{
		http.MethodDelete: RemoveWorkflow,
	})
}

func GetAllWorkflows(ctx *RequestContext, w http.ResponseWriter, r *http.Request) {
	wfs, err := ctx.Cache.GetAllWorkflows(r.Context())
	if err != nil {
		SendError(ctx, err)
	} else {
		SendJSON(w, wfs)
	}
}

func CreateWorkflow(ctx *RequestContext, w http.ResponseWriter, r *http.Request) {
	wf := &service.Workflow{}
	if !ParseAndValidate(ctx, r, wf) {
		return
	}

	if err := ctx.Coordinator.Register(wf); err != nil {
		SendError(ctx, err)
	} else {
		SendJSON(w, wf)
	}
}

func RemoveWorkflow(ctx *RequestContext, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name, ok := vars["name"]
	if !ok || name == "" {
		SendError(ctx, v1.ErrorCodeRequestInvalid.WithDetail("url name parameter missing or invalid"))
	} else {
		if _, err := ctx.Coordinator.UnloadWorkflow(name); err != nil {
			SendError(ctx, err)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
