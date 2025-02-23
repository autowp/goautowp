package mosts

import (
	"context"
	"errors"
	"fmt"
	"html"

	"github.com/autowp/goautowp/attrs"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

var (
	errAttributeIsGroup  = errors.New("attribute is group")
	errAttributeNotFound = errors.New("attribute not found")
)

type Attr struct {
	Attribute int64
	OrderAsc  bool
}

func (s Attr) Items(
	ctx context.Context, db *goqu.Database, attrsRepository *attrs.Repository, listOptions *query.ItemListOptions,
	lang string,
) (*MostData, error) {
	attribute, err := attrsRepository.Attribute(ctx, s.Attribute)
	if err != nil {
		return nil, err
	}

	if attribute == nil {
		return nil, fmt.Errorf("%w: '%d'", errAttributeNotFound, s.Attribute)
	}

	if !attribute.TypeID.Valid {
		return nil, fmt.Errorf("%w: '%d'", errAttributeIsGroup, s.Attribute)
	}

	valueTable, err := attrs.ValueTableByType(attribute.TypeID.AttributeTypeID)
	if err != nil {
		return nil, err
	}

	orderExp := valueTable.ValueCol.Desc()
	if s.OrderAsc {
		orderExp = valueTable.ValueCol.Asc()
	}

	iAliasTable := goqu.T(query.ItemAlias)
	itemIDCol := iAliasTable.Col(schema.ItemTableIDColName)

	sqSelect, err := listOptions.Select(db, query.ItemAlias)
	if err != nil {
		return nil, err
	}

	var itemIDs []int64

	err = sqSelect.
		Select(itemIDCol).
		Join(valueTable.Table, goqu.On(itemIDCol.Eq(valueTable.ItemIDCol))).
		Where(
			valueTable.AttributeIDCol.Eq(attribute.ID),
			valueTable.ValueCol.IsNotNull(),
		).
		Order(orderExp).
		ScanValsContext(ctx, &itemIDs)
	if err != nil {
		return nil, err
	}

	result := make([]MostDataCar, 0, len(itemIDs))

	for _, itemID := range itemIDs {
		_, valueText, err := attrsRepository.ActualValueText(ctx, attribute.ID, itemID, lang)
		if err != nil {
			return nil, err
		}

		result = append(result, MostDataCar{
			ItemID:    itemID,
			ValueHTML: html.EscapeString(valueText),
		})
	}

	var unit *schema.AttrsUnitRow
	if attribute.UnitID.Valid {
		unit, err = attrsRepository.Unit(ctx, attribute.UnitID.Int64)
		if err != nil {
			return nil, err
		}
	}

	return &MostData{
		Unit: unit,
		Cars: result,
	}, nil
}
