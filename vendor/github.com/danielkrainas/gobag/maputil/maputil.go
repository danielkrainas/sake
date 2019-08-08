package maputil

import (
	"github.com/danielkrainas/gobag/container"
)

func Filter(m map[string]interface{}, allowedKeys []string) map[string]interface{} {
	var ok bool
	var v interface{}
	r := map[string]interface{}{}
	for _, fieldName := range allowedKeys {
		v, ok = m[fieldName]
		if ok {
			r[fieldName] = v
		}
	}

	return r
}

func Exclude(m map[string]interface{}, filtered []string) map[string]interface{} {
	ss := bagcontainer.NewStringSet(filtered...)
	r := map[string]interface{}{}
	for fieldName, value := range m {
		if !ss.Contains(fieldName) {
			r[fieldName] = value
		}
	}

	return r
}
