package describe

import (
	"regexp"

	"github.com/danielkrainas/gobag/errcode"
)

type Route struct {
	Name        string
	Path        string
	Entity      string
	Description string
	Methods     []Method
}

type Method struct {
	Method      string
	Description string
	Requests    []Request
}

type Request struct {
	Name            string
	Description     string
	Headers         []Parameter
	PathParameters  []Parameter
	QueryParameters []Parameter
	Body            Body
	Successes       []Response
	Failures        []Response
}

type Response struct {
	Name        string
	Description string
	StatusCode  int
	Headers     []Parameter
	Fields      []Parameter
	ErrorCodes  []errcode.ErrorCode
	Body        Body
}

type Body struct {
	ContentType string
	Format      string
}

type Parameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Format      string
	Regexp      *regexp.Regexp
	Examples    []string
}
