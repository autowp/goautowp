package goautowp

import (
	"context"

	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/itemofday"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
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
		Catname:                    util.NullStringToString(row.Catname),
		EngineItemId:               util.NullInt64ToScalar(row.EngineItemID),
		DescendantsCount:           row.DescendantsCount,
		ItemTypeId:                 convertItemTypeID(row.ItemTypeID),
		IsConcept:                  row.IsConcept,
		IsConceptInherit:           row.IsConceptInherit,
		SpecId:                     int64(util.NullInt32ToScalar(row.SpecID)),
		Description:                row.Description,
		FullText:                   row.FullText,
		DescendantPicturesCount:    row.DescendantPicturesCount,
		ChildsCount:                row.ChildsCount,
		DescendantTwinsGroupsCount: row.DescendantTwinsGroupsCount,
		InboxPicturesCount:         row.InboxPicturesCount,
		FullName:                   row.FullName,
		MostsActive:                row.MostsActive,
		CommentsAttentionsCount:    row.CommentsAttentionsCount,
	}

	if fields.GetLogo120() && row.LogoID.Valid {
		logo120, err := s.imageStorage.FormattedImage(ctx, int(row.LogoID.Int64), "logo")
		if err != nil {
			return nil, err
		}

		result.Logo120 = APIImageToGRPC(logo120)
	}

	if fields.GetBrandicon() && row.LogoID.Valid {
		brandicon2, err := s.imageStorage.FormattedImage(ctx, int(row.LogoID.Int64), "brandicon2")
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
			BeginModelYear:         util.NullInt32ToScalar(row.BeginModelYear),
			EndModelYear:           util.NullInt32ToScalar(row.EndModelYear),
			BeginModelYearFraction: util.NullStringToString(row.BeginModelYearFraction),
			EndModelYearFraction:   util.NullStringToString(row.EndModelYearFraction),
			Spec:                   row.SpecShortName,
			SpecFull:               row.SpecName,
			Body:                   row.Body,
			Name:                   row.NameOnly,
			BeginYear:              util.NullInt32ToScalar(row.BeginYear),
			EndYear:                util.NullInt32ToScalar(row.EndYear),
			Today:                  util.NullBoolToBoolPtr(row.Today),
			BeginMonth:             util.NullInt16ToScalar(row.BeginMonth),
			EndMonth:               util.NullInt16ToScalar(row.EndMonth),
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
