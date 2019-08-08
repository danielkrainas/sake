package mapconv

import (
	"errors"
	"fmt"
	"time"

	"github.com/danielkrainas/gobag/iconv"
)

type Mapper interface {
	Map() map[string]interface{}
}

type Unmapper interface {
	Unmap(m map[string]interface{}) error
}

type MapFunc func() map[string]interface{}

type UnmapFunc func(map[string]interface{}) error

type ValueParser func(m map[string]interface{}) error

var errMissing = errors.New("missing required value")

func Parser(key string, required bool, f func(v interface{}) error) ValueParser {
	return func(m map[string]interface{}) error {
		var err error
		value, ok := m[key]
		if !ok && required {
			err = errMissing
		} else if ok {
			err = f(value)
		}

		if err != nil {
			return fmt.Errorf("%s: %v", key, err)
		}

		return nil
	}
}

func Int(key string, required bool, ref *int) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.Int(field)
		if err == nil {
			*ref = v
		}

		return err
	})
}

func Int16(key string, required bool, ref *int16) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.Int16(field)
		if err == nil {
			*ref = v
		}

		return err
	})
}

func Int32(key string, required bool, ref *int32) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.Int32(field)
		if err == nil {
			*ref = v
		}

		return err
	})
}

func Int64(key string, required bool, ref *int64) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.Int64(field)
		if err == nil {
			*ref = v
		}

		return err
	})
}

func Float64(key string, required bool, ref *float64) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.Float64(field)
		if err == nil {
			*ref = v
		}

		return err
	})
}

func Float32(key string, required bool, ref *float32) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.Float32(field)
		if err == nil {
			*ref = v
		}

		return err
	})
}

func Float32P(key string, required bool, ref **float32) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		if field != nil {
			v, err := iconv.Float32(field)
			if err == nil {
				*ref = &v
			}

			return err
		}

		*ref = nil
		return nil
	})
}

func String(key string, required bool, ref *string) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.String(field)
		if err == nil {
			*ref = v
		}

		return err
	})
}

func Interface(key string, required bool, ref *interface{}) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		*ref = field
		return nil
	})
}

func Bool(key string, required bool, ref *bool) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.Bool(field)
		if err == nil {
			*ref = v
		}

		return err
	})
}

func Stringish(key string, required bool, setter func(v string)) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.String(field)
		if err == nil {
			setter(v)
		}

		return err
	})
}

func Map(key string, required bool, ref *map[string]interface{}) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.Map(field)
		if err == nil {
			*ref = v
		}

		return err
	})
}

func MapArray(key string, required bool, ref *[]map[string]interface{}) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.MapArray(field)
		if err == nil {
			*ref = v
		}

		return err
	})
}

func StringArray(key string, required bool, ref *[]string) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.StringArray(field)
		if err == nil {
			*ref = v
		}

		return err
	})
}

func UnixTime(key string, required bool, ref *time.Time) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		v, err := iconv.Int64(field)
		if err == nil {
			*ref = time.Unix(v, 0)
		}

		return err
	})
}

func ParseUnmap(key string, required bool, init func(), unmap UnmapFunc) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		var err error
		if field == nil {
			err = unmap(nil)
		} else {
			v, err := iconv.Map(field)
			if err == nil {
				if v != nil {
					init()
				}

				err = unmap(v)
			}
		}

		return err
	})
}

func ParseUnmapArray(key string, required bool, init func(), unmap UnmapFunc) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		var err error
		if field == nil {
			err = unmap(nil)
		} else {
			v, err := iconv.MapArray(field)
			if err == nil {
				if v != nil {
					init()
				}

				for _, m := range v {
					if err = unmap(m); err != nil {
						break
					}
				}
			}
		}

		return err
	})
}

func ParseUnmapMap(key string, required bool, init func(), unmap func(key string, nm map[string]interface{}) error) ValueParser {
	return Parser(key, required, func(field interface{}) error {
		if field == nil {
			return nil
		}

		var err error
		if nm, ok := field.(map[string]map[string]interface{}); ok {
			init()
			for k, m := range nm {
				if err = unmap(k, m); err != nil {
					break
				}
			}
		} else if v, err := iconv.Map(field); err == nil {
			init()
			for k, m := range v {
				if sm, ok := m.(map[string]interface{}); ok {
					if err = unmap(k, sm); err != nil {
						break
					}
				}
			}
		}

		return err
	})
}

func Compose(m map[string]interface{}, parsers ...ValueParser) error {
	var err error
	for _, p := range parsers {
		if err = p(m); err != nil {
			break
		}
	}

	return err
}

func MapMapperArray(items []interface{}) []map[string]interface{} {
	ms := make([]map[string]interface{}, 0)
	for _, item := range items {
		if mapper, ok := item.(Mapper); ok {
			ms = append(ms, mapper.Map())
		} else if m, ok := item.(map[string]interface{}); ok {
			ms = append(ms, m)
		}
	}

	return ms
}
