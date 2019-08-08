package v1

import (
	"github.com/danielkrainas/gobag/api/describe"
)

var (
	VersionHeader = describe.Parameter{
		Name:        "Api-Version",
		Type:        "string",
		Description: "The build version of the server.",
		Format:      "<version>",
		Examples:    []string{"0.0.0-dev"},
	}
)

var (
	errorsBody = `{
	"errors:" [
	    {
            "code": <error code>,
            "message": <error message>,
            "detail": ...
        },
        ...
    ]
}`
)

type Route struct {
	Path string
	Name string
}

var API = struct {
	Routes []Route `json:"routes"`
}{
	Routes: routeDescriptors,
}

var routeDescriptors = []Route{
	{"/v1", RouteNameBase},
}

var APIDescriptor map[string]Route

func init() {
	APIDescriptor = make(map[string]Route, len(routeDescriptors))
	for _, descriptor := range routeDescriptors {
		APIDescriptor[descriptor.Name] = descriptor
	}
}
