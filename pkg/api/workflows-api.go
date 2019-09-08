package api

import (
	"net/http"

	"github.com/danielkrainas/sake/pkg/service"
)

func WorkflowsAPI() HttpHandler {
	return MethodRouter(map[string]HttpHandler{
		http.MethodGet:  GetAllWorkflows,
		http.MethodPost: CreateWorkflow,
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
