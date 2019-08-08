package uuid

import (
	"github.com/satori/go.uuid"
)

func Generate() string {
	return uuid.NewV4().String()
}
