package configuration

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/go-yaml/yaml"
	"github.com/sirupsen/logrus"
)

type Version string

func MajorMinorVersion(major uint, minor uint) Version {
	return Version(fmt.Sprintf("%d.%d", major, minor))
}

func (version Version) major() (uint, error) {
	majorPart := strings.Split(string(version), ".")[0]
	major, err := strconv.ParseUint(majorPart, 10, 0)
	return uint(major), err
}

func (version Version) Major() uint {
	major, _ := version.major()
	return major
}

func (version Version) minor() (uint, error) {
	minorPart := strings.Split(string(version), ".")[1]
	minor, err := strconv.ParseInt(minorPart, 10, 0)
	return uint(minor), err
}

func (version Version) Minor() uint {
	minor, _ := version.minor()
	return minor
}

type envVar struct {
	name  string
	value string
}

type envVars []envVar

func (a envVars) Len() int {
	return len(a)
}

func (a envVars) Swap(i int, j int) {
	x := a[i]
	a[i] = a[j]
	a[j] = x
}

func (a envVars) Less(i int, j int) bool {
	return a[i].name < a[j].name
}

type VersionedParseInfo struct {
	Version Version

	ParseAs reflect.Type

	ConversionFunc func(interface{}) (interface{}, error)
}

type Parser struct {
	prefix  string
	mapping map[Version]VersionedParseInfo
	env     envVars
}

func NewParser(prefix string, parseInfos []VersionedParseInfo) *Parser {
	p := Parser{
		prefix:  prefix,
		mapping: make(map[Version]VersionedParseInfo),
	}

	for _, parseInfo := range parseInfos {
		p.mapping[parseInfo.Version] = parseInfo
	}

	for _, env := range os.Environ() {
		envParts := strings.SplitN(env, "=", 2)
		p.env = append(p.env, envVar{envParts[0], envParts[1]})
	}

	sort.Sort(p.env)
	return &p
}

func (p *Parser) Parse(in []byte, v interface{}) error {
	var versionedData struct {
		Version Version
	}

	if err := yaml.Unmarshal(in, &versionedData); err != nil {
		return err
	}

	parseInfo, ok := p.mapping[versionedData.Version]
	if !ok {
		return fmt.Errorf("unsupported version: %q", versionedData.Version)
	}

	parseAs := reflect.New(parseInfo.ParseAs)
	if err := yaml.Unmarshal(in, parseAs.Interface()); err != nil {
		return err
	}

	for _, ev := range p.env {
		pathPart := ev.name
		if strings.HasPrefix(pathPart, strings.ToUpper(p.prefix)+"_") {
			path := strings.Split(pathPart, "_")
			if err := p.overwriteFields(parseAs, pathPart, path[1:], ev.value); err != nil {
				return err
			}
		}
	}

	c, err := parseInfo.ConversionFunc(parseAs.Interface())
	if err != nil {
		return err
	}

	reflect.ValueOf(v).Elem().Set(reflect.Indirect(reflect.ValueOf(c)))
	return nil
}

func (p *Parser) overwriteFields(v reflect.Value, fullpath string, path []string, payload string) error {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			panic("encountered nil pointer when handling environment variable " + fullpath)
		}

		v = reflect.Indirect(v)
	}

	switch v.Kind() {
	case reflect.Struct:
		return p.overwriteStruct(v, fullpath, path, payload)
	case reflect.Map:
		return p.overwriteMap(v, fullpath, path, payload)
	case reflect.Interface:
		if v.NumMethod() == 0 {
			if !v.IsNil() {
				return p.overwriteFields(v.Elem(), fullpath, path, payload)
			}

			var template map[string]interface{}
			wrapped := reflect.MakeMap(reflect.TypeOf(template))
			v.Set(wrapped)
			return p.overwriteMap(wrapped, fullpath, path, payload)
		}
	}

	return nil
}

func (p *Parser) overwriteStruct(v reflect.Value, fullpath string, path []string, payload string) error {
	uppercase := make(map[string]int)
	for i := 0; i < v.NumField(); i++ {
		vf := v.Type().Field(i)
		upper := strings.ToUpper(vf.Name)
		if _, present := uppercase[upper]; present {
			panic(fmt.Sprintf("field name collision in configuration object: %s", vf.Name))
		}

		uppercase[upper] = i
	}

	fi, present := uppercase[path[0]]
	if !present {
		if fullpath != "TINKERS_CONFIG_PATH" {
			logrus.Warnf("Ignoring unrecognized environment variable %s", fullpath)
		}

		return nil
	}

	f := v.Field(fi)
	vf := v.Type().Field(fi)
	if len(path) == 1 {
		fv := reflect.New(vf.Type)
		err := yaml.Unmarshal([]byte(payload), fv.Interface())
		if err != nil {
			return err
		}

		f.Set(reflect.Indirect(fv))
		return nil
	}

	switch vf.Type.Kind() {
	case reflect.Map:
		if f.IsNil() {
			f.Set(reflect.MakeMap(vf.Type))
		}

	case reflect.Ptr:
		if f.IsNil() {
			f.Set(reflect.New(vf.Type))
		}
	}

	err := p.overwriteFields(f, fullpath, path[1:], payload)
	if err != nil {
		return err
	}

	return nil
}

func (p *Parser) overwriteMap(m reflect.Value, fullpath string, path []string, payload string) error {
	if m.Type().Key().Kind() != reflect.String {
		logrus.Warnf("Ignoring environment variable %s involving map with non-string keys", fullpath)
		return nil
	}

	if len(path) > 1 {
		for _, k := range m.MapKeys() {
			if strings.ToUpper(k.String()) == path[0] {
				mv := m.MapIndex(k)
				if (mv.Kind() == reflect.Ptr ||
					mv.Kind() == reflect.Interface ||
					mv.Kind() == reflect.Map) &&
					mv.IsNil() {
					break
				}

				return p.overwriteFields(mv, fullpath, path[1:], payload)
			}
		}
	}

	var mv reflect.Value
	if m.Type().Elem().Kind() == reflect.Map {
		mv = reflect.MakeMap(m.Type().Elem())
	} else {
		mv = reflect.New(m.Type().Elem())
	}

	if len(path) > 1 {
		if err := p.overwriteFields(mv, fullpath, path[1:], payload); err != nil {
			return err
		}
	} else {
		if err := yaml.Unmarshal([]byte(payload), mv.Interface()); err != nil {
			return err
		}
	}

	m.SetMapIndex(reflect.ValueOf(strings.ToLower(path[0])), reflect.Indirect(mv))
	return nil
}
