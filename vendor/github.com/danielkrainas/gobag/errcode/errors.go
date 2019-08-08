package errcode

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ErrorCoder interface {
	ErrorCode() ErrorCode
}

type ErrorCode int

func (ec ErrorCode) ErrorCode() ErrorCode {
	return ec
}

func (ec ErrorCode) Error() string {
	return strings.ToLower(strings.Replace(ec.String(), "_", " ", -1))
}

func (ec ErrorCode) Descriptor() ErrorDescriptor {
	if d, ok := errorCodeToDescriptors[ec]; ok {
		return d
	}

	return ErrorCodeUnknown.Descriptor()
}

func (ec ErrorCode) String() string {
	return ec.Descriptor().Value
}

func (ec ErrorCode) Message() string {
	return ec.Descriptor().Message
}

func (ec ErrorCode) MarshalText() ([]byte, error) {
	return []byte(ec.String()), nil
}

func (ec *ErrorCode) UnmarshalText(text []byte) error {
	desc, ok := idToDescriptors[string(text)]
	if !ok {
		desc = ErrorCodeUnknown.Descriptor()
	}

	*ec = desc.Code
	return nil
}

func (ec ErrorCode) WithMessage(message string) Error {
	return Error{
		Code:    ec,
		Message: message,
	}
}

func (ec ErrorCode) WithDetail(detail interface{}) Error {
	e := Error{
		Code:    ec,
		Message: ec.Message(),
	}

	return e.WithDetail(detail)
}

func (ec ErrorCode) WithArgs(args ...interface{}) Error {
	e := Error{
		Code:    ec,
		Message: ec.Message(),
	}

	return e.WithArgs(args...)
}

type Error struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Detail  interface{} `json:"detail,omitempty"`
}

var _ error = Error{}

func (e Error) ErrorCode() ErrorCode {
	return e.Code
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code.Error(), e.Message)
}

func (e Error) WithDetail(detail interface{}) Error {
	return Error{
		Code:    e.Code,
		Message: e.Message,
		Detail:  detail,
	}
}

func (e Error) WithArgs(args ...interface{}) Error {
	return Error{
		Code:    e.Code,
		Message: fmt.Sprintf(e.Code.Message(), args...),
		Detail:  e.Detail,
	}
}

type ErrorDescriptor struct {
	Code           ErrorCode
	Value          string
	Message        string
	Description    string
	HTTPStatusCode int
}

type errorsStruct struct {
	Errors []Error `json:"errors,omitempty"`
}

type Errors []error

var _ error = Error{}

func (errs Errors) Error() string {
	switch len(errs) {
	case 0:
		return "<nil>"

	case 1:
		return errs[0].Error()

	default:
		msg := "errors:\n"
		for _, err := range errs {
			msg += err.Error() + "\n"
		}

		return msg
	}
}

func (errs Errors) Len() int {
	return len(errs)
}

func (errs Errors) MarshalJSON() ([]byte, error) {
	tmp := errorsStruct{}

	for _, err := range errs {
		var rerr Error

		switch err.(type) {
		case ErrorCode:
			rerr = err.(ErrorCode).WithDetail(nil)

		case Error:
			rerr = err.(Error)

		default:
			rerr = ErrorCodeUnknown.WithDetail(err)
		}

		msg := rerr.Message
		if msg == "" {
			msg = rerr.Code.Message()
		}

		tmp.Errors = append(tmp.Errors, Error{
			Code:    rerr.Code,
			Message: msg,
			Detail:  rerr.Detail,
		})
	}

	return json.Marshal(tmp)
}

func (errs *Errors) UnmarshalJSON(data []byte) error {
	tmp := errorsStruct{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	var verrs Errors
	for _, err := range tmp.Errors {
		if err.Detail == nil && (err.Message == "" || err.Message == err.Code.Message()) {
			verrs = append(verrs, err.Code)
		} else {
			verrs = append(verrs, Error{
				Code:    err.Code,
				Message: err.Message,
				Detail:  err.Detail,
			})
		}
	}

	*errs = verrs
	return nil
}
