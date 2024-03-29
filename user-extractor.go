package goautowp

import (
	"context"
	"fmt"
	"time"

	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/users"
	"github.com/casbin/casbin"
	"github.com/drexedam/gravatar"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const avatarSize = 70

type UserExtractor struct {
	enforcer     *casbin.Enforcer
	imageStorage *storage.Storage
}

func NewUserExtractor(enforcer *casbin.Enforcer, imageStorage *storage.Storage) *UserExtractor {
	return &UserExtractor{
		enforcer:     enforcer,
		imageStorage: imageStorage,
	}
}

func (s *UserExtractor) Extract(ctx context.Context, row *users.DBUser) (*APIUser, error) {
	longAway := true

	if row.LastOnline != nil {
		date := time.Now().AddDate(0, -6, 0)
		longAway = date.After(*row.LastOnline)
	}

	isGreen := row.Role != "" && s.enforcer.Enforce(row.Role, "status", "be-green")

	route := []string{"/users", fmt.Sprintf("user%d", row.ID)}
	if row.Identity != nil {
		route = []string{"/users", *row.Identity}
	}

	identity := ""
	if row.Identity != nil {
		identity = *row.Identity
	}

	user := APIUser{
		Id:          row.ID,
		Name:        row.Name,
		Deleted:     row.Deleted,
		LongAway:    longAway,
		Green:       isGreen,
		Route:       route,
		Identity:    identity,
		SpecsWeight: row.SpecsWeight,
	}

	if row.LastOnline != nil {
		user.LastOnline = timestamppb.New(*row.LastOnline)
	}

	if row.EMail != nil {
		user.Gravatar = gravatar.New(*row.EMail).Size(avatarSize).Rating(gravatar.G).AvatarURL()
	}

	if row.Img != nil {
		avatar, err := s.imageStorage.FormattedImage(ctx, *row.Img, "avatar")
		if err != nil {
			return nil, err
		}

		user.Avatar = APIImageToGRPC(avatar)
	}

	return &user, nil
}
