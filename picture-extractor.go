package goautowp

import (
	"context"
	"errors"
	"fmt"

	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"google.golang.org/genproto/googleapis/type/latlng"
)

var errItemNotFound = errors.New("item not found")

type PictureExtractor struct {
	imageStorage         *storage.Storage
	pictureNameFormatter pictures.PictureNameFormatter
	repository           *pictures.Repository
	i18n                 *i18nbundle.I18n
	commentsRepository   *comments.Repository
}

func NewPictureExtractor(
	repository *pictures.Repository, imageStorage *storage.Storage, i18n *i18nbundle.I18n,
	commentsRepository *comments.Repository,
) *PictureExtractor {
	return &PictureExtractor{
		repository:         repository,
		imageStorage:       imageStorage,
		i18n:               i18n,
		commentsRepository: commentsRepository,
		pictureNameFormatter: pictures.PictureNameFormatter{
			ItemNameFormatter: items.ItemNameFormatter{},
		},
	}
}

func (s *PictureExtractor) Extract(
	ctx context.Context, row *schema.PictureRow, fields *PictureFields, lang string, isModer bool, userID int64,
) (*Picture, error) {
	result, err := s.ExtractRows(ctx, []*schema.PictureRow{row}, fields, lang, isModer, userID)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, errItemNotFound
	}

	return result[0], nil
}

func (s *PictureExtractor) ExtractRows(
	ctx context.Context, rows []*schema.PictureRow, fields *PictureFields, lang string, isModer bool, userID int64,
) ([]*Picture, error) {
	var (
		namesData map[int64]pictures.PictureNameFormatterOptions
		err       error
		result    = make([]*Picture, 0, len(rows))
		localizer = s.i18n.Localizer(lang)
		images    = make(map[int]*storage.Image, 0)
	)

	if fields.GetNameText() || fields.GetNameHtml() {
		namesData, err = s.repository.NameData(ctx, rows, pictures.NameDataOptions{
			Language: lang,
		})
		if err != nil {
			return nil, err
		}
	}

	if fields.GetImage() || isModer {
		ids := make([]int, 0, len(rows))

		for _, row := range rows {
			if row.ImageID.Valid {
				ids = append(ids, int(row.ImageID.Int64))
			}
		}

		imageRows, err := s.imageStorage.Images(ctx, ids)
		if err != nil {
			return nil, err
		}

		for _, imageRow := range imageRows {
			images[imageRow.ID()] = imageRow
		}
	}

	for _, row := range rows {
		resultRow := &Picture{
			Id:               row.ID,
			Identity:         row.Identity,
			Width:            uint32(row.Width),
			Height:           uint32(row.Height),
			CopyrightsTextId: util.NullInt32ToScalar(row.CopyrightsTextID),
			OwnerId:          util.NullInt64ToScalar(row.OwnerID),
			Status:           reverseConvertPicturesStatus(row.Status),
			Resolution:       fmt.Sprintf("%d×%d", row.Width, row.Height),
		}

		if isModer && row.ImageID.Valid {
			if img, ok := images[int(row.ImageID.Int64)]; ok {
				resultRow.Cropped = img.CropHeight() > 0 && img.CropWidth() > 0
				if resultRow.GetCropped() {
					resultRow.CropResolution = fmt.Sprintf("%d×%d", img.CropWidth(), img.CropHeight())
				}
			}
		}

		if fields.GetNameText() || fields.GetNameHtml() {
			nameData, ok := namesData[row.ID]
			if ok {
				if fields.GetNameText() {
					resultRow.NameText, err = s.pictureNameFormatter.FormatText(nameData, localizer)
					if err != nil {
						return nil, err
					}
				}

				if fields.GetNameHtml() {
					resultRow.NameHtml, err = s.pictureNameFormatter.FormatHTML(nameData, localizer)
					if err != nil {
						return nil, err
					}
				}
			}
		}

		if row.Point.Valid {
			resultRow.Point = &latlng.LatLng{
				Latitude:  row.Point.Point.Lat(),
				Longitude: row.Point.Point.Lng(),
			}
		}

		if fields.GetImage() && row.ImageID.Valid {
			if image, ok := images[int(row.ImageID.Int64)]; ok {
				resultRow.Image = APIImageToGRPC(image)
			}
		}

		if fields.GetThumbMedium() {
			if row.ImageID.Valid {
				image, err := s.imageStorage.FormattedImage(ctx, int(row.ImageID.Int64), "picture-thumb-medium")
				if err != nil {
					return nil, err
				}

				resultRow.ThumbMedium = APIImageToGRPC(image)
			}
		}

		if fields.GetViews() {
			resultRow.Views, err = s.repository.PictureViews(ctx, row.ID)
			if err != nil {
				return nil, err
			}
		}

		if fields.GetVotes() {
			vote, err := s.repository.GetVote(ctx, row.ID, userID)
			if err != nil {
				return nil, err
			}

			resultRow.Votes = &PicturesVoteSummary{
				Value:    vote.Value,
				Positive: vote.Positive,
				Negative: vote.Negative,
			}
		}

		if fields.GetCommentsCount() {
			var count, newCount int32
			if userID > 0 {
				count, newCount, err = s.commentsRepository.TopicStatForUser(
					ctx, schema.CommentMessageTypeIDPictures, row.ID, userID,
				)
				if err != nil {
					return nil, err
				}
			} else {
				count, err = s.commentsRepository.TopicStat(ctx, schema.CommentMessageTypeIDPictures, row.ID)
				if err != nil {
					return nil, err
				}
			}

			resultRow.CommentsCountTotal = count
			resultRow.CommentsCountNew = newCount
		}

		if fields.GetModerVote() {
			count, sum, err := s.repository.ModerVoteCount(ctx, row.ID)
			if err != nil {
				return nil, err
			}

			resultRow.ModerVoteCount = count
			resultRow.ModerVoteVote = sum
		}

		result = append(result, resultRow)
	}

	return result, nil
}
