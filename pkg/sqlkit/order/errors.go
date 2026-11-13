package order

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyField      = errors.New("empty field")
	ErrDuplicateField  = errors.New("duplicate field")
	ErrTooManyFields   = errors.New("too many fields")
	ErrFieldNotAllowed = errors.New("field not allowed")
)

type Error struct {
	Field string
	Err   error
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("order: %s", e.Err.Error())
	}

	return fmt.Sprintf("order: %q %s", e.Field, e.Err.Error())
}
