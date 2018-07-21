package version

import (
	"fmt"
	"strings"
)

const (
	Major = "0"
	Minor = "1"
	Patch = "1"
)

var Build = "unknown" // For -ldflags '-X x=x'

func Full() string {
	return fmt.Sprintf("%s@%s", Info(), Build)
}

func Info() string {
	return fmt.Sprintf("%s.%s.%s", Major, Minor, Patch)
}

func IsCompatible(ver string) bool {
	parts := strings.SplitN(ver, ".", 3)
	if len(parts) != 3 {
		return false
	}
	major, minor := parts[0], parts[1]
	return major == Major && minor == Minor
}
