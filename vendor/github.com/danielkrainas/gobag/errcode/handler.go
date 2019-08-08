package errcode

import (
	"encoding/json"
	"net/http"
)

func ServeJSON(w http.ResponseWriter, err error) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	var status int

	switch errs := err.(type) {
	case Errors:
		if len(errs) < 1 {
			break
		}

		if err, ok := errs[0].(ErrorCoder); ok {
			status = err.ErrorCode().Descriptor().HTTPStatusCode
		}

	case ErrorCoder:
		status = errs.ErrorCode().Descriptor().HTTPStatusCode
		err = Errors{err}

	default:
		err = Errors{err}
	}

	if status == 0 {
		status = http.StatusInternalServerError
	}

	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(err)
}
