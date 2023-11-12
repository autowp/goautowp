package goautowp

import (
	"context"

	"github.com/autowp/goautowp/items"
	"github.com/casbin/casbin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func convertItemTypeID(itemTypeID items.ItemType) ItemType {
	switch itemTypeID {
	case items.VEHICLE:
		return ItemType_ITEM_TYPE_VEHICLE
	case items.ENGINE:
		return ItemType_ITEM_TYPE_ENGINE
	case items.CATEGORY:
		return ItemType_ITEM_TYPE_CATEGORY
	case items.TWINS:
		return ItemType_ITEM_TYPE_TWINS
	case items.BRAND:
		return ItemType_ITEM_TYPE_BRAND
	case items.FACTORY:
		return ItemType_ITEM_TYPE_FACTORY
	case items.MUSEUM:
		return ItemType_ITEM_TYPE_MUSEUM
	case items.PERSON:
		return ItemType_ITEM_TYPE_PERSON
	case items.COPYRIGHT:
		return ItemType_ITEM_TYPE_COPYRIGHT
	}

	return ItemType_ITEM_TYPE_UNKNOWN
}

func reverseConvertItemTypeID(itemTypeID ItemType) items.ItemType {
	switch itemTypeID {
	case ItemType_ITEM_TYPE_UNKNOWN:
		return 0
	case ItemType_ITEM_TYPE_VEHICLE:
		return items.VEHICLE
	case ItemType_ITEM_TYPE_ENGINE:
		return items.ENGINE
	case ItemType_ITEM_TYPE_CATEGORY:
		return items.CATEGORY
	case ItemType_ITEM_TYPE_TWINS:
		return items.TWINS
	case ItemType_ITEM_TYPE_BRAND:
		return items.BRAND
	case ItemType_ITEM_TYPE_FACTORY:
		return items.FACTORY
	case ItemType_ITEM_TYPE_MUSEUM:
		return items.MUSEUM
	case ItemType_ITEM_TYPE_PERSON:
		return items.PERSON
	case ItemType_ITEM_TYPE_COPYRIGHT:
		return items.COPYRIGHT
	}

	return 0
}

type ItemExtractor struct {
	enforcer      *casbin.Enforcer
	nameFormatter *items.ItemNameFormatter
}

func NewItemExtractor(enforcer *casbin.Enforcer) *ItemExtractor {
	return &ItemExtractor{
		enforcer:      enforcer,
		nameFormatter: &items.ItemNameFormatter{},
	}
}

func (s *ItemExtractor) Extract(
	_ context.Context, row items.Item, fields *ItemFields, localizer *i18n.Localizer,
) (*APIItem, error) {
	if fields == nil {
		fields = &ItemFields{}
	}

	result := &APIItem{
		Id:                   row.ID,
		Catname:              row.Catname,
		EngineItemId:         row.EngineItemID,
		DescendantsCount:     row.DescendantsCount,
		ItemTypeId:           convertItemTypeID(row.ItemTypeID),
		IsConcept:            row.IsConcept,
		IsConceptInherit:     row.IsConceptInherit,
		SpecId:               row.SpecID,
		Description:          row.Description,
		FullText:             row.FullText,
		CurrentPicturesCount: row.CurrentPicturesCount,
	}

	if fields.NameOnly {
		result.NameOnly = row.NameOnly
	}

	if fields.NameText || fields.NameHtml {
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

		if fields.NameText {
			nameText, err := s.nameFormatter.FormatText(formatterOptions, localizer)
			if err != nil {
				return nil, err
			}

			result.NameText = nameText
		}

		if fields.NameHtml {
			nameHTML, err := s.nameFormatter.FormatHTML(formatterOptions, localizer)
			if err != nil {
				return nil, err
			}

			result.NameHtml = nameHTML
		}
	}

	return result, nil
}
