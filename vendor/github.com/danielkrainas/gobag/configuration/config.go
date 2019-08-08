package configuration

import (
	"fmt"
	"strings"
)

func (version *Version) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var versionString string
	err := unmarshal(&versionString)
	if err != nil {
		return err
	}

	newVersion := Version(versionString)
	if _, err := newVersion.major(); err != nil {
		return err
	}

	if _, err := newVersion.minor(); err != nil {
		return err
	}

	*version = newVersion
	return nil
}

type Parameters map[string]interface{}

type Driver map[string]Parameters

func (driver Driver) Type() string {
	var driverType []string

	for k := range driver {
		driverType = append(driverType, k)
	}

	if len(driverType) > 1 {
		panic("multiple drivers specified in the configuration or environment: %s" + strings.Join(driverType, ", "))
	}

	if len(driverType) == 1 {
		return driverType[0]
	}

	return ""
}

func (driver Driver) Parameters() Parameters {
	return driver[driver.Type()]
}

func (driver Driver) setParameter(key string, value interface{}) {
	driver[driver.Type()][key] = value
}

func (driver *Driver) UnmarshalText(text []byte) error {
	driverType := string(text)
	*driver = Driver{
		driverType: Parameters{},
	}

	return nil
}

func (driver *Driver) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var driverMap map[string]Parameters
	err := unmarshal(&driverMap)
	if err == nil && len(driverMap) > 0 {
		types := make([]string, 0, len(driverMap))
		for k := range driverMap {
			types = append(types, k)
		}

		if len(types) > 1 {
			return fmt.Errorf("Must provide exactly one driver type. provided: %v", types)
		}

		*driver = driverMap
		return nil
	}

	var driverType string
	if err = unmarshal(&driverType); err != nil {
		return err
	}

	*driver = Driver{
		driverType: Parameters{},
	}

	return nil
}

func (driver Driver) MarshalYAML() (interface{}, error) {
	if driver.Parameters() == nil {
		return driver.Type(), nil
	}

	return map[string]Parameters(driver), nil
}

type LogLevel string

func (logLevel *LogLevel) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strLogLevel string
	err := unmarshal(&strLogLevel)
	if err != nil {
		return err
	}

	strLogLevel = strings.ToLower(strLogLevel)
	switch strLogLevel {
	case "error", "warn", "info", "debug":
	default:
		return fmt.Errorf("Invalid log level %s. Must be one of [error, warn, info, debug]", strLogLevel)
	}

	*logLevel = LogLevel(strLogLevel)
	return nil
}
