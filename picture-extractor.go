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
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/casbin/casbin"
	"google.golang.org/genproto/googleapis/type/latlng"
)

var errItemNotFound = errors.New("item not found")

type PictureExtractor struct {
	imageStorage         *storage.Storage
	pictureNameFormatter *pictures.PictureNameFormatter
	repository           *pictures.Repository
	i18n                 *i18nbundle.I18n
	commentsRepository   *comments.Repository
	itemsRepository      *items.Repository
	enforcer             *casbin.Enforcer
}

func NewPictureExtractor(
	repository *pictures.Repository, imageStorage *storage.Storage, i18n *i18nbundle.I18n,
	commentsRepository *comments.Repository, itemsRepository *items.Repository, enforcer *casbin.Enforcer,
) *PictureExtractor {
	return &PictureExtractor{
		repository:           repository,
		imageStorage:         imageStorage,
		i18n:                 i18n,
		commentsRepository:   commentsRepository,
		itemsRepository:      itemsRepository,
		enforcer:             enforcer,
		pictureNameFormatter: pictures.NewPictureNameFormatter(items.NewItemNameFormatter(i18n), i18n),
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

func (s *PictureExtractor) ExtractRows( //nolint: maintidx
	ctx context.Context, rows []*schema.PictureRow, fields *PictureFields, lang string, isModer bool, userID int64,
) ([]*Picture, error) {
	var (
		namesData map[int64]pictures.PictureNameFormatterOptions
		err       error
		result    = make([]*Picture, 0, len(rows))
		images    = make(map[int]*storage.Image)
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
			Status:           extractPicturesStatus(row.Status),
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
					resultRow.NameText, err = s.pictureNameFormatter.FormatText(nameData, lang)
					if err != nil {
						return nil, err
					}
				}

				if fields.GetNameHtml() {
					resultRow.NameHtml, err = s.pictureNameFormatter.FormatHTML(nameData, lang)
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

		if fields.GetThumb() && row.ImageID.Valid {
			image, err := s.imageStorage.FormattedImage(ctx, int(row.ImageID.Int64), "picture-thumb")
			if err != nil {
				return nil, err
			}

			resultRow.Thumb = APIImageToGRPC(image)
		}

		if fields.GetThumbMedium() && row.ImageID.Valid {
			image, err := s.imageStorage.FormattedImage(ctx, int(row.ImageID.Int64), "picture-thumb-medium")
			if err != nil {
				return nil, err
			}

			resultRow.ThumbMedium = APIImageToGRPC(image)
		}

		if fields.GetImageGalleryFull() && row.ImageID.Valid {
			image, err := s.imageStorage.FormattedImage(ctx, int(row.ImageID.Int64), "picture-gallery-full")
			if err != nil {
				return nil, err
			}

			resultRow.ImageGalleryFull = APIImageToGRPC(image)
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

		path := fields.GetPath()
		if path != nil {
			resultRow.Path, err = s.path(ctx, row.ID, path.GetParentId())
			if err != nil {
				return nil, err
			}
		}

		pictureItemRequest := fields.GetPictureItem()
		if pictureItemRequest != nil {
			piOptions, err := convertPictureItemOptions(pictureItemRequest.GetOptions())
			if err != nil {
				return nil, err
			}

			if piOptions == nil {
				piOptions = &query.PictureItemListOptions{}
			}

			piOptions.PictureID = row.ID

			piRows, err := s.repository.PictureItems(ctx, piOptions)
			if err != nil {
				return nil, err
			}

			extractor := NewPictureItemExtractor(s.enforcer)

			res := make([]*PictureItem, 0, len(piRows))
			for _, piRow := range piRows {
				res = append(res, extractor.Extract(piRow))
			}

			resultRow.PictureItems = &PictureItems{
				Items: res,
			}
		}

		dfDistanceRequest := fields.GetDfDistance()
		if dfDistanceRequest != nil {
			ddOptions, err := convertDfDistanceListOptions(dfDistanceRequest.GetOptions())
			if err != nil {
				return nil, err
			}

			if ddOptions == nil {
				ddOptions = &query.DfDistanceListOptions{}
			}

			ddOptions.SrcPictureID = row.ID

			ddRows, err := s.repository.DfDistances(ctx, ddOptions, dfDistanceRequest.GetLimit())
			if err != nil {
				return nil, err
			}

			dfDistanceExtractor := NewDfDistanceExtractor(s.repository, s)

			res, err := dfDistanceExtractor.ExtractRows(ctx, ddRows, dfDistanceRequest.GetFields(), lang, isModer, userID)
			if err != nil {
				return nil, err
			}

			resultRow.DfDistances = &DfDistances{
				Items: res,
			}
		}

		result = append(result, resultRow)
	}

	return result, nil
}

func (s *PictureExtractor) path(
	ctx context.Context, pictureID int64, targetItemID int64,
) ([]*PathTreePictureItem, error) {
	piRows, err := s.repository.PictureItems(ctx, &query.PictureItemListOptions{
		PictureID: pictureID,
		TypeID:    schema.PictureItemContent,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*PathTreePictureItem, 0)

	for _, piRow := range piRows {
		item, err := s.itemRoute(ctx, piRow.ItemID, targetItemID)
		if err != nil {
			return nil, err
		}

		if item != nil {
			result = append(result, &PathTreePictureItem{
				PerspectiveId: int32(util.NullInt64ToScalar(piRow.PerspectiveID)), //nolint: gosec
				Item:          item,
			})
		}
	}

	return result, nil
}

func (s *PictureExtractor) itemRoute(ctx context.Context, itemID int64, targetItemID int64) (*PathTreeItem, error) {
	row, err := s.itemsRepository.Item(ctx, &query.ItemListOptions{
		ItemID: itemID,
	}, items.ListFields{})
	if err != nil {
		if errors.Is(err, items.ErrItemNotFound) {
			return nil, nil //nolint: nilnil
		}

		return nil, err
	}

	parentItemTypes := []schema.ItemTableItemTypeID{
		schema.ItemTableItemTypeIDCategory,
		schema.ItemTableItemTypeIDEngine,
		schema.ItemTableItemTypeIDVehicle,
	}

	parents := make([]*PathTreeItemParent, 0)
	if util.Contains(parentItemTypes, row.ItemTypeID) {
		parents, err = s.itemParentRoute(ctx, row.ID, targetItemID)
		if err != nil {
			return nil, err
		}
	}

	if len(parents) == 0 && targetItemID != 0 && itemID != targetItemID {
		return nil, nil //nolint: nilnil
	}

	return &PathTreeItem{
		ItemTypeId: extractItemTypeID(row.ItemTypeID),
		Catname:    util.NullStringToString(row.Catname),
		Parents:    parents,
	}, nil
}

func (s *PictureExtractor) itemParentRoute(
	ctx context.Context, itemID int64, targetItemID int64,
) ([]*PathTreeItemParent, error) {
	result := make([]*PathTreeItemParent, 0)

	if itemID > 0 {
		rows, _, err := s.itemsRepository.ItemParents(ctx, &query.ItemParentListOptions{
			ItemID: itemID,
		}, items.ItemParentFields{}, items.ItemParentOrderByNone)
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			item, err := s.itemRoute(ctx, row.ParentID, targetItemID)
			if err != nil {
				return nil, err
			}

			if item != nil {
				result = append(result, &PathTreeItemParent{
					Catname: row.Catname,
					Item:    item,
				})
			}
		}
	}

	return result, nil
}
