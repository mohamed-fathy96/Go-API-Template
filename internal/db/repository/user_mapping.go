package repository

import (
	"kabsa/ent"
	dom "kabsa/internal/domain/user"
)

func toDomainUser(e *ent.User) *dom.User {
	if e == nil {
		return nil
	}
	return &dom.User{
		ID:        e.ID,
		Email:     e.Email,
		Name:      e.Name,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func toDomainUsers(list []*ent.User) []dom.User {
	res := make([]dom.User, 0, len(list))
	for _, e := range list {
		if e == nil {
			continue
		}
		res = append(res, *toDomainUser(e))
	}
	return res
}
