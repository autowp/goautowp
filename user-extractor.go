package goautowp

import (
	"fmt"
	"time"
)

type UserExtractor struct {
	container *Container
}

func NewUserExtractor(container *Container) *UserExtractor {
	return &UserExtractor{
		container: container,
	}
}

func (s *UserExtractor) Extract(row *DBUser) (*APIUser, error) {
	longAway := true
	if row.LastOnline != nil {
		date := time.Now().AddDate(0, -6, 0)
		longAway = date.After(*row.LastOnline)
	}

	enforcer, err := s.container.GetEnforcer()
	if err != nil {
		return nil, err
	}

	isGreen := row.Role != "" && enforcer.Enforce(row.Role, "status", "be-green")

	route := []string{"/users", fmt.Sprintf("user%d", row.ID)}
	if row.Identity != nil {
		route = []string{"/users", *row.Identity}
	}

	return &APIUser{
		ID:       row.ID,
		Name:     row.Name,
		Deleted:  row.Deleted,
		LongAway: longAway,
		Green:    isGreen,
		Route:    route,
		Identity: row.Identity,
	}, nil
}
