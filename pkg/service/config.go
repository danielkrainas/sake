package service

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/caarlos0/env"
	"github.com/go-yaml/yaml"
)

type Config struct {
	Log struct {
		Level     string                 `yaml:"level" toml:"level" env:"SAKE_LOG_LEVEL"`
		Formatter string                 `yaml:"formatter" toml:"formatter" env:"SAKE_LOG_FORMAT"`
		Fields    map[string]interface{} `yaml:"fields" toml:"fields"`
	} `yaml:"log" toml:"log"`

	HTTP struct {
		Addr string `yaml:"addr" toml:"addr" env:"SAKE_HTTP_ADDR"`
	} `yaml:"http" toml:"http"`

	StorageDriver string `yaml:"storage" toml:"storage" env:"SAKE_STORAGE"`
}

func DefaultConfig() *Config {
	config := &Config{}
	config.HTTP.Addr = ":8889"
	config.Log.Level = "debug"
	config.Log.Formatter = "text"
	config.StorageDriver = ""

	return config
}

// Resolve determines the application's config location and loads it.
func ResolveConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = os.Getenv("SAKE_CONFIG_PATH")
	}

	if configPath == "" {
		return DefaultConfig(), nil
	}

	fp, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("configuration: %v", err)
	}

	defer fp.Close()
	config, err := ParseConfig(fp, path.Ext(configPath)[1:])
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", configPath, err)
	}

	fmt.Fprintf(os.Stderr, "%+v\n====\n", config)
	return config, nil
}

// Validate determines if the configuration is prepared correctly and valid to use.
func ValidateConfig(config *Config) (*Config, error) {
	return config, nil
}

// Parse loads and parses the configuration from a reader.
func ParseConfig(rd io.Reader, parser string) (*Config, error) {
	in, err := ioutil.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	config := new(Config)
	switch parser {
	case "yml":
		fallthrough
	case "yaml":
		if err := yaml.Unmarshal(in, config); err != nil {
			return nil, err
		}

	default:
	case "toml":
		if _, err := toml.Decode(string(in), config); err != nil {
			return nil, err
		}
	}

	if err := env.Parse(config); err != nil {
		return nil, err
	}

	return config, nil
}
