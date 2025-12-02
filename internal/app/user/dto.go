package user

import (
	dom "kabsa/internal/domain/user"
	"time"
)

type UserDto struct {
	Id        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateUserInput struct {
	Email string
	Name  string
}

type UpdateUserInput struct {
	ID   int64
	Name *string
}

type ListUsersInput struct {
	Limit  int
	Offset int
}

func toDTO(u *dom.User) *UserDto {
	if u == nil {
		return nil
	}
	return &UserDto{
		Id:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func toDTOs(list []dom.User) []UserDto {
	res := make([]UserDto, 0, len(list))
	for _, u := range list {
		item := u // copy
		res = append(res, *toDTO(&item))
	}
	return res
}
