package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/danielkrainas/gobag/errcode"
	"github.com/danielkrainas/sake/pkg/api/v1"
	"github.com/danielkrainas/sake/pkg/service"
	"github.com/danielkrainas/sake/pkg/util/log"
	"go.uber.org/zap"
)

type RequestContext struct {
	RequestID string
	StartedAt time.Time
	Errors    errcode.Errors

	Coordinator service.CoordinatorService
	Cache       service.CacheService
}

type ContextKey int

const RequestContextKey ContextKey = 1

func GetRequestContext(r *http.Request) *RequestContext {
	if v := r.Context().Value(RequestContextKey); v != nil {
		vv, ok := v.(*RequestContext)
		if ok {
			return vv
		}
	}

	return nil
}

func AppendRequestContext(rc *RequestContext, r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), RequestContextKey, rc))
}

type OptionsValidator interface {
	Validate() error
}

type OptionsRequestParser interface {
	ParseRequest(r *http.Request) error
}

func readRequestOptions(r *http.Request, options interface{}) error {
	if parser, ok := options.(OptionsRequestParser); ok {
		if err := parser.ParseRequest(r); err != nil {
			log.Error("request parse failed", zap.Error(err))
			return err
		}

		return nil
	}

	if r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodPut {
		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error("body read failed", zap.Error(err))
			return err
		}

		err = json.Unmarshal(buf, options)
		if err != nil {
			log.Error("body parse failed", zap.Error(err))
			return err
		}
	} else {
		log.Error("unsure how to parse request options")
	}

	return nil
}

func Parse(rc *RequestContext, r *http.Request, options interface{}) bool {
	err := readRequestOptions(r, options)
	if err != nil {
		SendError(rc, v1.ErrorCodeRequestInvalid.WithDetail(err))
		return false
	}

	return true
}

func Validate(rc *RequestContext, r *http.Request, validator interface{}) bool {
	if v, ok := validator.(OptionsValidator); ok {
		if err := v.Validate(); err != nil {
			SendError(rc, v1.ErrorCodeRequestInvalid.WithDetail(err))
			return false
		}
	}

	return true
}

func ParseAndValidate(rc *RequestContext, r *http.Request, options interface{}) bool {
	return Parse(rc, r, options) && Validate(rc, r, options)
}

func SendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error("result json encoding failed", zap.Error(err))
	}
}

func SendError(rc *RequestContext, err error) {
	rc.Errors = append(rc.Errors, err)
}
