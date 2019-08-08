package v1

import "github.com/gorilla/mux"

const (
	RouteNameBase = "base"
)

func Router() *mux.Router {
	return RouterWithPrefix("")
}

func RouterWithPrefix(prefix string) *mux.Router {
	rootRouter := mux.NewRouter()
	router := rootRouter
	if prefix != "" {
		router = router.PathPrefix(prefix).Subrouter()
	}

	router.StrictSlash(true)
	for _, descriptor := range routeDescriptors {
		router.Path(descriptor.Path).Name(descriptor.Name)
	}

	return rootRouter
}
