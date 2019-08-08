package configuration

import (
	"fmt"
	"os"
)

type Resolver struct {
	PathEnvKey  string
	DefaultPath string
}

func (r *Resolver) Resolve(args []string) (*os.File, error) {
	var configPath string

	if os.Getenv(r.PathEnvKey) != "" {
		configPath = os.Getenv(r.PathEnvKey)
	}

	if configPath == "" {
		return nil, fmt.Errorf("configuration path not specified")
	}

	return os.Open(configPath)
}
