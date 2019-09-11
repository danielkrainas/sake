package v1

import (
	"net/http"

	"github.com/danielkrainas/gobag/errcode"
)

const ErrorGroup = "sake.api.v1"

var (
	ErrorCodeRequestInvalid = errcode.Register(ErrorGroup, errcode.ErrorDescriptor{
		Value:          "REQUEST_INVALID",
		Message:        "request validation failed",
		Description:    "",
		HTTPStatusCode: http.StatusBadRequest,
	})

	ErrorCodeRecipeMultiModify = errcode.Register(ErrorGroup, errcode.ErrorDescriptor{
		Value:          "WORKFLOW_MULTI_MODIFY",
		Message:        "two or more requests attempted to modify the %q recipe. please retry the request",
		Description:    "",
		HTTPStatusCode: http.StatusBadRequest,
	})
)
