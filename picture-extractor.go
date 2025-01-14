package goautowp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"

	"github.com/autowp/goautowp/attrs"
	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/itemofday"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/textstorage"
	"github.com/autowp/goautowp/util"
	"github.com/casbin/casbin"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var errItemNotFound = errors.New("item not found")

type PictureExtractor struct {
	imageStorage          *storage.Storage
	pictureNameFormatter  *pictures.PictureNameFormatter
	repository            *pictures.Repository
	i18n                  *i18nbundle.I18n
	commentsRepository    *comments.Repository
	itemsRepository       *items.Repository
	enforcer              *casbin.Enforcer
	textstorageRepository *textstorage.Repository
	itemOfDayRepository   *itemofday.Repository
	attrsRepository       *attrs.Repository
}

func NewPictureExtractor(
	repository *pictures.Repository, imageStorage *storage.Storage, i18n *i18nbundle.I18n,
	commentsRepository *comments.Repository, itemsRepository *items.Repository, enforcer *casbin.Enforcer,
	textstorageRepository *textstorage.Repository, itemOfDayRepository *itemofday.Repository,
	attrsRepository *attrs.Repository,
) *PictureExtractor {
	return &PictureExtractor{
		repository:            repository,
		imageStorage:          imageStorage,
		i18n:                  i18n,
		commentsRepository:    commentsRepository,
		itemsRepository:       itemsRepository,
		enforcer:              enforcer,
		textstorageRepository: textstorageRepository,
		itemOfDayRepository:   itemOfDayRepository,
		attrsRepository:       attrsRepository,
		pictureNameFormatter:  pictures.NewPictureNameFormatter(items.NewItemNameFormatter(i18n), i18n),
	}
}

func (s *PictureExtractor) Extract(
	ctx context.Context, row *schema.PictureRow, fields *PictureFields, lang string, isModer bool, userID int64,
	role string,
) (*Picture, error) {
	result, err := s.ExtractRows(ctx, []*schema.PictureRow{row}, fields, lang, isModer, userID, role)
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
	role string,
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
			AddDate:          timestamppb.New(row.AddDate),
			TakenDate: &date.Date{
				Year:  int32(util.NullInt16ToScalar(row.TakenYear)),
				Month: int32(util.NullByteToScalar(row.TakenMonth)),
				Day:   int32(util.NullByteToScalar(row.TakenDay)),
			},
			DpiX: util.NullInt32ToScalar(row.DPIX),
			DpiY: util.NullInt32ToScalar(row.DPIY),
		}

		if row.ChangeStatusUserID.Valid {
			resultRow.ChangeStatusUserId = row.ChangeStatusUserID.Int64
		}

		if isModer && fields.GetSpecialName() {
			resultRow.SpecialName = util.NullStringToString(row.Name)
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

			iExtractor := NewItemExtractor(s.enforcer, s.imageStorage, s.commentsRepository, s.repository,
				s.itemsRepository, s.itemOfDayRepository, s.attrsRepository, s.i18n)
			ipcExtractor := NewItemParentCacheExtractor(s.itemsRepository, iExtractor)
			extractor := NewPictureItemExtractor(s.itemsRepository, iExtractor, ipcExtractor)

			res, err := extractor.ExtractRows(ctx, piRows, pictureItemRequest.GetFields(), lang)
			if err != nil {
				return nil, err
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

			res, err := dfDistanceExtractor.ExtractRows(ctx, ddRows, dfDistanceRequest.GetFields(), lang, isModer, userID, role)
			if err != nil {
				return nil, err
			}

			resultRow.DfDistances = &DfDistances{
				Items: res,
			}
		}

		if fields.GetAcceptedCount() {
			acceptedCount, err := s.repository.Count(ctx, &query.PictureListOptions{
				Status: schema.PictureStatusAccepted,
				PictureItem: &query.PictureItemListOptions{
					PictureItemByItemID: &query.PictureItemListOptions{
						PictureID: row.ID,
					},
				},
			})
			if err != nil {
				return nil, err
			}

			resultRow.AcceptedCount = int32(acceptedCount) //nolint: gosec
		}

		if fields.GetCopyrights() {
			if row.CopyrightsTextID.Valid {
				copyrights, err := s.textstorageRepository.Text(ctx, row.CopyrightsTextID.Int32)
				if err != nil && !errors.Is(err, textstorage.ErrTextNotFound) {
					return nil, err
				}

				if err == nil {
					resultRow.Copyrights = copyrights
				}
			}
		}

		if fields.GetExif() && row.ImageID.Valid {
			exif, err := s.imageStorage.ImageEXIF(ctx, int(row.ImageID.Int64))
			if err != nil {
				return nil, err
			}

			exifStr := ""
			skipSections := []string{"FILE", "COMPUTED"}

			if len(exif) > 0 {
				for key, section := range exif {
					if util.Contains(skipSections, key) {
						continue
					}

					exifStr += "<p>[" + html.EscapeString(key) + "]"
					for name, val := range section {
						exifStr += "<br />" + html.EscapeString(name) + ": " + fmt.Sprintf("%v", val)
					}

					exifStr += "</p>"
				}
			}

			resultRow.Exif = exifStr
		}

		if fields.GetIsLast() {
			isLastPicture := false

			if row.Status == schema.PictureStatusAccepted {
				isLastPicture, err = s.repository.Exists(ctx, &query.PictureListOptions{
					ExcludeID: row.ID,
					Status:    schema.PictureStatusAccepted,
					PictureItem: &query.PictureItemListOptions{
						PictureItemByItemID: &query.PictureItemListOptions{
							PictureID: row.ID,
						},
					},
				})
				if err != nil {
					return nil, err
				}
			}

			resultRow.IsLast = isLastPicture
		}

		if fields.GetModerVoted() && userID != 0 {
			resultRow.ModerVoted, err = s.repository.HasModerVote(ctx, row.ID, userID)
			if err != nil {
				return nil, err
			}
		}

		pictureModerVoteRequest := fields.GetPictureModerVotes()
		if pictureModerVoteRequest != nil {
			pmvOptions := convertPictureModerVoteListOptions(pictureModerVoteRequest.GetOptions())
			if pmvOptions == nil {
				pmvOptions = &query.PictureModerVoteListOptions{}
			}

			pmvOptions.PictureID = row.ID

			pmvRows, err := s.repository.PictureModerVotes(ctx, pmvOptions)
			if err != nil {
				return nil, err
			}

			pmvExtractor := NewPictureModerVoteExtractor()

			res, err := pmvExtractor.ExtractRows(pmvRows)
			if err != nil {
				return nil, err
			}

			resultRow.PictureModerVotes = &PictureModerVotes{
				Items: res,
			}
		}

		replaceableRequest := fields.GetReplaceable()
		if replaceableRequest != nil && row.ReplacePictureID.Valid {
			pOptions, err := convertPictureListOptions(replaceableRequest.GetOptions())
			if err != nil {
				return nil, err
			}

			if pOptions == nil {
				pOptions = &query.PictureListOptions{}
			}

			pOptions.ID = row.ReplacePictureID.Int64

			pFields := convertPictureFields(replaceableRequest.GetFields())

			pRow, err := s.repository.Picture(ctx, pOptions, pFields, pictures.OrderByNone)
			if err != nil {
				return nil, err
			}

			res, err := s.Extract(ctx, pRow, replaceableRequest.GetFields(), lang, isModer, userID, role)
			if err != nil {
				return nil, err
			}

			resultRow.Replaceable = res
		}

		if fields.GetRights() {
			canAccept, err := s.repository.CanAccept(ctx, row)
			if err != nil {
				return nil, err
			}

			canDelete, err := s.repository.CanDelete(ctx, row)
			if err != nil {
				return nil, err
			}

			resultRow.Rights = &PictureRights{
				Move:      s.enforcer.Enforce(role, "picture", "move"),
				Unaccept:  (row.Status == schema.PictureStatusAccepted) && s.enforcer.Enforce(role, "picture", "unaccept"),
				Accept:    canAccept && s.enforcer.Enforce(role, "picture", "accept"),
				Restore:   (row.Status == schema.PictureStatusRemoving) && s.enforcer.Enforce(role, "picture", "restore"),
				Normalize: (row.Status == schema.PictureStatusInbox) && s.enforcer.Enforce(role, "picture", "normalize"),
				Flop:      (row.Status == schema.PictureStatusInbox) && s.enforcer.Enforce(role, "picture", "flop"),
				Crop:      s.enforcer.Enforce(role, "picture", "crop"),
				Delete:    canDelete,
			}
		}

		siblings := fields.GetSiblings()
		if siblings != nil {
			resultRow.Siblings = &PictureSiblings{
				Prev:    nil,
				Next:    nil,
				PrevNew: nil,
				NextNew: nil,
			}

			sFields := siblings.GetFields()
			scFields := convertPictureFields(sFields)

			prevPicture, err := s.repository.Picture(ctx, &query.PictureListOptions{
				IDLt: row.ID,
			}, scFields, pictures.OrderByIDDesc)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}

			if err == nil {
				resultRow.Siblings.Prev, err = s.Extract(ctx, prevPicture, sFields, lang, isModer, userID, role)
				if err != nil {
					return nil, err
				}
			}

			nextPicture, err := s.repository.Picture(ctx, &query.PictureListOptions{
				IDGt: row.ID,
			}, scFields, pictures.OrderByIDAsc)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}

			if err == nil {
				resultRow.Siblings.Next, err = s.Extract(ctx, nextPicture, sFields, lang, isModer, userID, role)
				if err != nil {
					return nil, err
				}
			}

			prevNewPicture, err := s.repository.Picture(ctx, &query.PictureListOptions{
				IDLt:   row.ID,
				Status: schema.PictureStatusInbox,
			}, scFields, pictures.OrderByIDDesc)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}

			if err == nil {
				resultRow.Siblings.PrevNew, err = s.Extract(ctx, prevNewPicture, sFields, lang, isModer, userID, role)
				if err != nil {
					return nil, err
				}
			}

			nextNewPicture, err := s.repository.Picture(ctx, &query.PictureListOptions{
				IDGt:   row.ID,
				Status: schema.PictureStatusInbox,
			}, scFields, pictures.OrderByIDAsc)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}

			if err == nil {
				resultRow.Siblings.NextNew, err = s.Extract(ctx, nextNewPicture, sFields, lang, isModer, userID, role)
				if err != nil {
					return nil, err
				}
			}
		}

		if s.enforcer.Enforce(role, "user", "ip") && row.IP != nil {
			resultRow.Ip = row.IP.ToIP().String()
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
	}, nil)
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
