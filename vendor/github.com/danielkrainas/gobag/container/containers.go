package bagcontainer

type StringSet map[string]struct{}

func NewStringSet(keys ...string) StringSet {
	ss := make(StringSet, len(keys))
	ss.Add(keys...)
	return ss
}

func (ss StringSet) Add(keys ...string) {
	for _, key := range keys {
		ss[key] = struct{}{}
	}
}

func (ss StringSet) Contains(key string) bool {
	_, ok := ss[key]
	return ok
}

func (ss StringSet) Keys() []string {
	keys := make([]string, 0, len(ss))
	for key := range ss {
		keys = append(keys, key)
	}

	return keys
}
