package hana

import (
	"errors"
)

// ErrFileNotFound error
var ErrFileNotFound = errors.New("File not found")

// ErrOpNotAllowed error
var ErrOpNotAllowed = errors.New("Operation not allowed")
