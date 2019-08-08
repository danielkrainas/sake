package slugify

import (
	slug "github.com/metal3d/go-slugify"
)

func Marshal(s string) string {
	return slug.Marshal(s)
}
