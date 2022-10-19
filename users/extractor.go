package users

import (
	"context"
	"crypto/md5" //nolint: gosec
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/autowp/goautowp/image/storage"
	"github.com/casbin/casbin"
)

type UserExtractor struct {
	enforcer     *casbin.Enforcer
	imageStorage *storage.Storage
}

type APIImage struct {
	ID       int32
	Src      string
	Width    int32
	Height   int32
	Filesize int32
}

func NewUserExtractor(enforcer *casbin.Enforcer, imageStorage *storage.Storage) *UserExtractor {
	return &UserExtractor{
		enforcer:     enforcer,
		imageStorage: imageStorage,
	}
}

func ImageToAPIImage(i *storage.Image) *APIImage {
	if i == nil {
		return nil
	}

	return &APIImage{
		ID:       int32(i.ID()),
		Width:    int32(i.Width()),
		Height:   int32(i.Height()),
		Filesize: int32(i.FileSize()),
		Src:      i.Src(),
	}
}

func (s *UserExtractor) Extract(ctx context.Context, row *DBUser, fields map[string]bool) (*APIUser, error) {
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
		ID:          row.ID,
		Name:        row.Name,
		Deleted:     row.Deleted,
		LongAway:    longAway,
		Green:       isGreen,
		Route:       route,
		Identity:    identity,
		SpecsWeight: row.SpecsWeight,
	}

	for field := range fields {
		switch field {
		case "avatar":
			if row.Img != nil {
				avatar, err := s.imageStorage.FormattedImage(ctx, *row.Img, "avatar")
				if err != nil {
					return nil, err
				}

				user.Avatar = ImageToAPIImage(avatar)
			}

		case "gravatar":
			if row.EMail != nil {
				hash := md5.Sum([]byte(*row.EMail)) //nolint: gosec
				str := fmt.Sprintf(
					"https://www.gravatar.com/avatar/%x?s=70&d=%s&r=g",
					hex.EncodeToString(hash[:]),
					url.PathEscape("https://www.autowp.ru/_.gif"),
				)
				user.Gravatar = str
			}
		case "last_online":
			user.LastOnline = row.LastOnline
		}
	}

	return &user, nil
}
