package common

import (
	"errors"
	"fmt"
)

type NotFoundError struct {
	Entity string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", e.Entity)
}

func IsNotFound(err error) bool {
	var nf NotFoundError
	return errors.As(err, &nf)
}

func NewNotFound(entity string) error {
	return NotFoundError{Entity: entity}
}
