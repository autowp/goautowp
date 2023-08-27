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

	if fields.Name {
		result.Name = row.Name
	}

	if fields.NameText {
		nameText, err := s.nameFormatter.FormatText(items.ItemNameFormatterOptions{
			BeginModelYear:         row.BeginModelYear,
			EndModelYear:           row.EndModelYear,
			BeginModelYearFraction: row.BeginModelYearFraction,
			EndModelYearFraction:   row.EndModelYearFraction,
			Spec:                   row.SpecShortName,
			SpecFull:               row.SpecName,
			Body:                   row.Body,
			Name:                   row.Name,
			BeginYear:              row.BeginYear,
			EndYear:                row.EndYear,
			Today:                  row.Today,
			BeginMonth:             row.BeginMonth,
			EndMonth:               row.EndMonth,
		}, localizer)
		if err != nil {
			return nil, err
		}

		result.NameText = nameText
	}

	return result, nil
}
