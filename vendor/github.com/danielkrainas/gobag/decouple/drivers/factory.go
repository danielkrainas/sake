package drivers

import (
	"fmt"
	"sync"
)

type Factory interface {
	Create(parameters map[string]interface{}) (DriverBase, error)
}

type Registry struct {
	m         sync.Mutex
	Factories map[string]Factory
	AssetType string
}

func (r *Registry) Register(name string, factory Factory) {
	if factory == nil {
		panic(fmt.Sprintf("%s Factory cannot be nil", r.AssetType))
	}

	r.m.Lock()
	defer r.m.Unlock()

	if r.Factories == nil {
		r.Factories = make(map[string]Factory, 0)
	}

	if _, registered := r.Factories[name]; registered {
		panic(fmt.Sprintf("%s Factory named %s already registered", r.AssetType, name))
	}

	r.Factories[name] = factory
}

func (r *Registry) Create(name string, parameters map[string]interface{}) (DriverBase, error) {
	r.m.Lock()
	defer r.m.Unlock()

	if factory, ok := r.Factories[name]; ok {
		return factory.Create(parameters)
	}

	return nil, InvalidDriverError{r.AssetType, name}
}

type InvalidDriverError struct {
	AssetType string
	Name      string
}

func (err InvalidDriverError) Error() string {
	return fmt.Sprintf("%s driver not registered: %s", err.AssetType, err.Name)
}
