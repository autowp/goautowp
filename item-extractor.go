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
		Id:               row.ID,
		Catname:          row.Catname,
		EngineItemId:     row.EngineItemID,
		DescendantsCount: row.DescendantsCount,
		ItemTypeId:       convertItemTypeID(row.ItemTypeID),
		IsConcept:        row.IsConcept,
		IsConceptInherit: row.IsConceptInherit,
		SpecId:           row.SpecID,
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
