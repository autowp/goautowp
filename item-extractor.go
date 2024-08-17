package goautowp

import (
	"context"

	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/itemofday"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/schema"
	"github.com/casbin/casbin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func convertItemTypeID(itemTypeID schema.ItemTableItemTypeID) ItemType {
	switch itemTypeID {
	case schema.ItemTableItemTypeIDVehicle:
		return ItemType_ITEM_TYPE_VEHICLE
	case schema.ItemTableItemTypeIDEngine:
		return ItemType_ITEM_TYPE_ENGINE
	case schema.ItemTableItemTypeIDCategory:
		return ItemType_ITEM_TYPE_CATEGORY
	case schema.ItemTableItemTypeIDTwins:
		return ItemType_ITEM_TYPE_TWINS
	case schema.ItemTableItemTypeIDBrand:
		return ItemType_ITEM_TYPE_BRAND
	case schema.ItemTableItemTypeIDFactory:
		return ItemType_ITEM_TYPE_FACTORY
	case schema.ItemTableItemTypeIDMuseum:
		return ItemType_ITEM_TYPE_MUSEUM
	case schema.ItemTableItemTypeIDPerson:
		return ItemType_ITEM_TYPE_PERSON
	case schema.ItemTableItemTypeIDCopyright:
		return ItemType_ITEM_TYPE_COPYRIGHT
	}

	return ItemType_ITEM_TYPE_UNKNOWN
}

func reverseConvertItemTypeID(itemTypeID ItemType) schema.ItemTableItemTypeID {
	switch itemTypeID {
	case ItemType_ITEM_TYPE_UNKNOWN:
		return 0
	case ItemType_ITEM_TYPE_VEHICLE:
		return schema.ItemTableItemTypeIDVehicle
	case ItemType_ITEM_TYPE_ENGINE:
		return schema.ItemTableItemTypeIDEngine
	case ItemType_ITEM_TYPE_CATEGORY:
		return schema.ItemTableItemTypeIDCategory
	case ItemType_ITEM_TYPE_TWINS:
		return schema.ItemTableItemTypeIDTwins
	case ItemType_ITEM_TYPE_BRAND:
		return schema.ItemTableItemTypeIDBrand
	case ItemType_ITEM_TYPE_FACTORY:
		return schema.ItemTableItemTypeIDFactory
	case ItemType_ITEM_TYPE_MUSEUM:
		return schema.ItemTableItemTypeIDMuseum
	case ItemType_ITEM_TYPE_PERSON:
		return schema.ItemTableItemTypeIDPerson
	case ItemType_ITEM_TYPE_COPYRIGHT:
		return schema.ItemTableItemTypeIDCopyright
	}

	return 0
}

type ItemExtractor struct {
	enforcer            *casbin.Enforcer
	nameFormatter       *items.ItemNameFormatter
	imageStorage        *storage.Storage
	commentsRepository  *comments.Repository
	picturesRepository  *pictures.Repository
	itemOfDayRepository *itemofday.Repository
}

func NewItemExtractor(
	enforcer *casbin.Enforcer,
	imageStorage *storage.Storage,
	commentsRepository *comments.Repository,
	picturesRepository *pictures.Repository,
	itemOfDayRepository *itemofday.Repository,
) *ItemExtractor {
	return &ItemExtractor{
		enforcer:            enforcer,
		nameFormatter:       &items.ItemNameFormatter{},
		imageStorage:        imageStorage,
		commentsRepository:  commentsRepository,
		picturesRepository:  picturesRepository,
		itemOfDayRepository: itemOfDayRepository,
	}
}

func (s *ItemExtractor) Extract(
	ctx context.Context, row items.Item, fields *ItemFields, localizer *i18n.Localizer,
) (*APIItem, error) {
	if fields == nil {
		fields = &ItemFields{}
	}

	result := &APIItem{
		Id:                         row.ID,
		Catname:                    row.Catname,
		EngineItemId:               row.EngineItemID,
		DescendantsCount:           row.DescendantsCount,
		ItemTypeId:                 convertItemTypeID(row.ItemTypeID),
		IsConcept:                  row.IsConcept,
		IsConceptInherit:           row.IsConceptInherit,
		SpecId:                     row.SpecID,
		Description:                row.Description,
		FullText:                   row.FullText,
		CurrentPicturesCount:       row.CurrentPicturesCount,
		ChildsCount:                row.ChildsCount,
		DescendantTwinsGroupsCount: row.DescendantTwinsGroupsCount,
		InboxPicturesCount:         row.InboxPicturesCount,
		FullName:                   row.FullName,
		MostsActive:                row.MostsActive,
	}

	if fields.GetLogo120() && row.LogoID != 0 {
		logo120, err := s.imageStorage.FormattedImage(ctx, int(row.LogoID), "logo")
		if err != nil {
			return nil, err
		}

		result.Logo120 = APIImageToGRPC(logo120)
	}

	if fields.GetBrandicon() && row.LogoID != 0 {
		brandicon2, err := s.imageStorage.FormattedImage(ctx, int(row.LogoID), "brandicon2")
		if err != nil {
			return nil, err
		}

		result.Brandicon = APIImageToGRPC(brandicon2)
	}

	if fields.GetNameOnly() {
		result.NameOnly = row.NameOnly
	}

	if fields.GetNameText() || fields.GetNameHtml() {
		formatterOptions := items.ItemNameFormatterOptions{
			BeginModelYear:         row.BeginModelYear,
			EndModelYear:           row.EndModelYear,
			BeginModelYearFraction: row.BeginModelYearFraction,
			EndModelYearFraction:   row.EndModelYearFraction,
			Spec:                   row.SpecShortName,
			SpecFull:               row.SpecName,
			Body:                   row.Body,
			Name:                   row.NameOnly,
			BeginYear:              row.BeginYear,
			EndYear:                row.EndYear,
			Today:                  row.Today,
			BeginMonth:             row.BeginMonth,
			EndMonth:               row.EndMonth,
		}

		if fields.GetNameText() {
			nameText, err := s.nameFormatter.FormatText(formatterOptions, localizer)
			if err != nil {
				return nil, err
			}

			result.NameText = nameText
		}

		if fields.GetNameHtml() {
			nameHTML, err := s.nameFormatter.FormatHTML(formatterOptions, localizer)
			if err != nil {
				return nil, err
			}

			result.NameHtml = nameHTML
		}

		if fields.GetCommentsAttentionsCount() {
			cnt, err := s.commentsRepository.Count(ctx, comments.ModeratorAttentionRequired, comments.TypeIDPictures, row.ID)
			if err != nil {
				return nil, err
			}

			result.CommentsAttentionsCount = cnt
		}

		if fields.GetInboxPicturesCount() {
			cnt, err := s.picturesRepository.Count(ctx, pictures.ListOptions{
				Status:         pictures.StatusInbox,
				AncestorItemID: row.ID,
			})
			if err != nil {
				return nil, err
			}

			result.InboxPicturesCount = int32(cnt)
		}

		if fields.GetIsCompilesItemOfDay() {
			IsCompiles, err := s.itemOfDayRepository.IsComplies(ctx, row.ID)
			if err != nil {
				return nil, err
			}

			result.IsCompilesItemOfDay = IsCompiles
		}
	}

	return result, nil
}
