package goautowp

import (
	"crypto/md5"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/url"
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

func (s *UserExtractor) Extract(row *DBUser, fields map[string]bool) (*User, error) {
	longAway := true
	if row.LastOnline != nil {
		date := time.Now().AddDate(0, -6, 0)
		longAway = date.After(*row.LastOnline)
	}

	enforcer := s.container.GetEnforcer()

	isGreen := row.Role != "" && enforcer.Enforce(row.Role, "status", "be-green")

	route := []string{"/users", fmt.Sprintf("user%d", row.ID)}
	if row.Identity != nil {
		route = []string{"/users", *row.Identity}
	}

	identity := ""
	if row.Identity != nil {
		identity = *row.Identity
	}

	user := User{
		Id:       int32(row.ID),
		Name:     row.Name,
		Deleted:  row.Deleted,
		LongAway: longAway,
		Green:    isGreen,
		Route:    route,
		Identity: identity,
	}

	for field := range fields {
		switch field {
		case "avatar":
			// TODO
		case "gravatar":
			if row.EMail != nil {
				str := fmt.Sprintf(
					"https://www.gravatar.com/avatar/%x?s=70&d=%s&r=g",
					md5.Sum([]byte(*row.EMail)),
					url.PathEscape("https://www.autowp.ru/_.gif"),
				)
				user.Gravatar = str
			}
		case "last_online":
			if row.LastOnline != nil {
				user.LastOnline = timestamppb.New(*row.LastOnline)
			}
		}
	}

	return &user, nil
}
