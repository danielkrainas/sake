package uid

import "github.com/segmentio/ksuid"

func Generate() string {
	uid, _ := ksuid.NewRandom()
	return uid.String()
}
