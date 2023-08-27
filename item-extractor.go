package goautowp

import (
	"context"

	"github.com/autowp/goautowp/items"

	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/casbin/casbin"
)

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
	result := &APIItem{
		Id:               row.ID,
		Catname:          row.Catname,
		DescendantsCount: row.DescendantsCount,
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
