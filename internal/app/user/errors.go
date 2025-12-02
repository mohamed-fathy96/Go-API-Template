package user

import (
	domcommon "kabsa/internal/domain/common"
)

func IsNotFound(err error) bool {
	return domcommon.IsNotFound(err)
}

func NewUserNotFoundError() error {
	return domcommon.NewNotFound("user")
}
