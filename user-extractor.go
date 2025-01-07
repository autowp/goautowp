package goautowp

import (
	"context"
	"time"

	"github.com/autowp/goautowp/frontend"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/casbin/casbin"
	"github.com/drexedam/gravatar"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	avatarSize      = 70
	avatarLargeSize = 270
)

type UserExtractor struct {
	enforcer           *casbin.Enforcer
	imageStorage       *storage.Storage
	picturesRepository *pictures.Repository
}

func NewUserExtractor(
	enforcer *casbin.Enforcer, imageStorage *storage.Storage, picturesRepository *pictures.Repository,
) *UserExtractor {
	return &UserExtractor{
		enforcer:           enforcer,
		imageStorage:       imageStorage,
		picturesRepository: picturesRepository,
	}
}

func (s *UserExtractor) Extract(
	ctx context.Context, row *schema.UsersRow, fields *UserFields, currentUserID int64, currentUserRole string,
) (*APIUser, error) {
	longAway := true

	if row.LastOnline != nil {
		date := time.Now().AddDate(0, -6, 0)
		longAway = date.After(*row.LastOnline)
	}

	isGreen := row.Role != "" && s.enforcer.Enforce(row.Role, "status", "be-green")

	identity := ""
	if row.Identity != nil {
		identity = *row.Identity
	}

	user := APIUser{
		Id:            row.ID,
		Name:          row.Name,
		Deleted:       row.Deleted,
		LongAway:      longAway,
		Green:         isGreen,
		Route:         frontend.UserRoute(row.ID, row.Identity),
		Identity:      identity,
		SpecsWeight:   row.SpecsWeight,
		PicturesAdded: int32(row.PicturesAdded), //nolint:gosec
	}

	if fields.GetRegDate() && row.RegDate != nil {
		user.RegDate = timestamppb.New(*row.RegDate)
	}

	if row.LastOnline != nil {
		user.LastOnline = timestamppb.New(*row.LastOnline)
	}

	if row.EMail != nil {
		gr := gravatar.New(*row.EMail)
		user.Gravatar = gr.Size(avatarSize).Rating(gravatar.G).AvatarURL()

		if fields.GetGravatarLarge() {
			user.GravatarLarge = gr.Size(avatarLargeSize).Rating(gravatar.G).AvatarURL()
		}
	}

	if row.Img != nil {
		avatar, err := s.imageStorage.FormattedImage(ctx, *row.Img, "avatar")
		if err != nil {
			return nil, err
		}

		user.Avatar = APIImageToGRPC(avatar)

		if fields.GetPhoto() {
			photo, err := s.imageStorage.FormattedImage(ctx, *row.Img, "photo")
			if err != nil {
				return nil, err
			}

			user.Photo = APIImageToGRPC(photo)
		}
	}

	isMe := row.ID == currentUserID

	if fields.GetEmail() && row.EMail != nil &&
		(isMe || len(currentUserRole) > 0 && s.enforcer.Enforce(currentUserRole, "global", "moderate")) {
		user.Email = *row.EMail
	}

	if isMe {
		if fields.GetVotesLeft() {
			user.VotesLeft = row.VotesLeft
		}

		if fields.GetVotesPerDay() {
			user.VotesPerDay = row.VotesPerDay
		}

		if fields.GetLanguage() {
			user.Language = row.Language
		}

		if fields.GetTimezone() {
			user.Timezone = row.Timezone
		}

		if fields.GetImg() && row.Img != nil {
			img, err := s.imageStorage.Image(ctx, *row.Img)
			if err != nil {
				return nil, err
			}

			user.Img = APIImageToGRPC(img)
		}
	}

	if fields.GetPicturesAcceptedCount() {
		count, err := s.picturesRepository.Count(ctx, &query.PictureListOptions{
			Status:  schema.PictureStatusAccepted,
			OwnerID: row.ID,
		})
		if err != nil {
			return nil, err
		}

		user.PicturesAcceptedCount = int32(count) //nolint:gosec
	}

	if fields.GetLastIp() && len(currentUserRole) > 0 && s.enforcer.Enforce(currentUserRole, "user", "ip") {
		user.LastIp = row.LastIP
	}

	if fields.GetIsModer() {
		user.IsModer = s.enforcer.Enforce(row.Role, "global", "moderate")
	}

	if fields.GetLogin() && row.Login != nil && s.enforcer.Enforce(currentUserRole, "global", "moderate") {
		user.Login = *row.Login
	}

	return &user, nil
}
