package storage

import (
	"strings"
)

// XXX(damnever): nothing to say...

func IsExist(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint")
}

func IsNotExist(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no rows")
}
