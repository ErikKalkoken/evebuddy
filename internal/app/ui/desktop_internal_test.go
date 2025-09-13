package ui

import (
	"os"
	"strings"
)

func IsCI() bool {
	return strings.ToLower(os.Getenv("CI")) == "true"
}
