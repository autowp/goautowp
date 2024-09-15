package attrs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

var errAttributeNotFound = errors.New("attribute not found")

type TopUserBrand struct {
	ID      int64  `db:"id"`
	Name    string `db:"name"`
	Catname string `db:"catname"`
	Volume  int64  `db:"volume"`
}

type Value struct {
	Valid       bool
	IntValue    int32
	FloatValue  float64
	StringValue string
	BoolValue   bool
	ListValue   []int64
	Type        schema.AttrsAttributeTypeID
	IsEmpty     bool
}

// Repository Main Object.
type Repository struct {
	db                *goqu.Database
	i18n              *i18nbundle.I18n
	listOptions       map[int64]map[int64]string
	listOptionsChilds map[int64]map[int64][]int64
}

// NewRepository constructor.
func NewRepository(
	db *goqu.Database,
	i18n *i18nbundle.I18n,
) *Repository {
	return &Repository{
		db:                db,
		i18n:              i18n,
		listOptions:       make(map[int64]map[int64]string),
		listOptionsChilds: make(map[int64]map[int64][]int64),
	}
}

func (s *Repository) Attribute(ctx context.Context, id int64) (bool, schema.AttrsAttributeRow, error) {
	sqSelect := s.db.Select(
		schema.AttrsAttributesTableIDCol, schema.AttrsAttributesTableNameCol, schema.AttrsAttributesTableDescriptionCol,
		schema.AttrsAttributesTableTypeIDCol, schema.AttrsAttributesTableUnitIDCol, schema.AttrsAttributesTableMultipleCol,
		schema.AttrsAttributesTablePrecisionCol, schema.AttrsAttributesTableParentIDCol,
	).
		From(schema.AttrsAttributesTable).
		Order(schema.AttrsAttributesTablePositionCol.Asc()).
		Where(schema.AttrsAttributesTableIDCol.Eq(id))

	r := schema.AttrsAttributeRow{}
	success, err := sqSelect.ScanStructContext(ctx, &r)

	return success, r, err
}

func (s *Repository) Attributes(ctx context.Context, zoneID int64, parentID int64) ([]schema.AttrsAttributeRow, error) {
	sqSelect := s.db.Select(
		schema.AttrsAttributesTableIDCol, schema.AttrsAttributesTableNameCol, schema.AttrsAttributesTableDescriptionCol,
		schema.AttrsAttributesTableTypeIDCol, schema.AttrsAttributesTableUnitIDCol, schema.AttrsAttributesTableMultipleCol,
		schema.AttrsAttributesTablePrecisionCol, schema.AttrsAttributesTableParentIDCol,
	).
		From(schema.AttrsAttributesTable)

	if zoneID > 0 {
		sqSelect = sqSelect.Join(
			schema.AttrsZoneAttributesTable,
			goqu.On(schema.AttrsAttributesTableIDCol.Eq(schema.AttrsZoneAttributesTableAttributeIDCol)),
		).
			Where(schema.AttrsZoneAttributesTableZoneIDCol.Eq(zoneID)).
			Order(schema.AttrsZoneAttributesTablePositionCol.Asc())
	} else {
		sqSelect = sqSelect.Order(schema.AttrsAttributesTablePositionCol.Asc())
	}

	if parentID > 0 {
		sqSelect = sqSelect.Where(schema.AttrsAttributesTableParentIDCol.Eq(parentID))
	}

	r := make([]schema.AttrsAttributeRow, 0)
	err := sqSelect.ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) AttributeTypes(ctx context.Context) ([]schema.AttrsAttributeTypeRow, error) {
	r := make([]schema.AttrsAttributeTypeRow, 0)
	err := s.db.Select(schema.AttrsTypesTableIDCol, schema.AttrsTypesTableNameCol).
		From(schema.AttrsTypesTable).
		ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) ListOptions(ctx context.Context, attributeID int64) ([]schema.AttrsListOptionRow, error) {
	sqSelect := s.db.Select(schema.AttrsListOptionsTableIDCol, schema.AttrsListOptionsTableNameCol,
		schema.AttrsListOptionsTableAttributeIDCol, schema.AttrsListOptionsTableParentIDCol).
		From(schema.AttrsListOptionsTable).
		Order(schema.AttrsListOptionsTablePositionCol.Asc())

	if attributeID > 0 {
		sqSelect = sqSelect.Where(schema.AttrsListOptionsTableAttributeIDCol.Eq(attributeID))
	}

	r := make([]schema.AttrsListOptionRow, 0)
	err := sqSelect.ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) Units(ctx context.Context) ([]schema.AttrsUnitRow, error) {
	r := make([]schema.AttrsUnitRow, 0)
	err := s.db.Select(schema.AttrsUnitsTableIDCol, schema.AttrsUnitsTableNameCol, schema.AttrsUnitsTableAbbrCol).
		From(schema.AttrsUnitsTable).
		ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) ZoneAttributes(ctx context.Context, zoneID int64) ([]schema.AttrsZoneAttributeRow, error) {
	attrs := make([]schema.AttrsZoneAttributeRow, 0)
	err := s.db.Select(schema.AttrsZoneAttributesTableZoneIDCol, schema.AttrsZoneAttributesTableAttributeIDCol).
		From(schema.AttrsZoneAttributesTable).
		Where(schema.AttrsZoneAttributesTableZoneIDCol.Eq(zoneID)).
		ScanStructsContext(ctx, &attrs)

	return attrs, err
}

func (s *Repository) Zones(ctx context.Context) ([]schema.AttrsZoneRow, error) {
	r := make([]schema.AttrsZoneRow, 0)
	err := s.db.Select(schema.AttrsZonesTableIDCol, schema.AttrsZonesTableNameCol).
		From(schema.AttrsZonesTable).
		ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) TotalValues(ctx context.Context) (int32, error) {
	sqSelect := s.db.From(schema.AttrsValuesTable)

	result, err := sqSelect.CountContext(ctx)
	if err != nil {
		return 0, err
	}

	return int32(result), nil //nolint: gosec
}

func (s *Repository) TotalZoneAttrs(ctx context.Context, zoneID int64) (int32, error) {
	sqSelect := s.db.From(schema.AttrsAttributesTable).
		Join(
			schema.AttrsZoneAttributesTable,
			goqu.On(schema.AttrsAttributesTableIDCol.Eq(schema.AttrsZoneAttributesTableAttributeIDCol)),
		).
		Where(schema.AttrsZoneAttributesTableZoneIDCol.Eq(zoneID))

	result, err := sqSelect.CountContext(ctx)
	if err != nil {
		return 0, err
	}

	return int32(result), nil //nolint: gosec
}

func (s *Repository) TopUserBrands(
	ctx context.Context, userID int64, limit uint,
) ([]TopUserBrand, error) {
	rows := make([]TopUserBrand, 0)

	const volumeAlias = "volume"
	err := s.db.Select(
		schema.ItemTableIDCol, schema.ItemTableNameCol, schema.ItemTableCatnameCol,
		goqu.COUNT(goqu.Star()).As(volumeAlias),
	).
		From(schema.ItemTable).
		Join(schema.ItemParentCacheTable, goqu.On(schema.ItemTableIDCol.Eq(schema.ItemParentCacheTableParentIDCol))).
		Join(
			schema.AttrsUserValuesTable,
			goqu.On(schema.ItemParentCacheTableItemIDCol.Eq(schema.AttrsUserValuesTableItemIDCol)),
		).
		Where(
			schema.ItemTableItemTypeIDCol.Eq(schema.ItemTableItemTypeIDBrand),
			schema.AttrsUserValuesTableUserIDCol.Eq(userID),
		).
		GroupBy(schema.ItemTableIDCol).
		Order(goqu.C(volumeAlias).Desc()).
		Limit(limit).
		ScanStructsContext(ctx, &rows)

	return rows, err
}

func (s *Repository) Values(ctx context.Context, options query.AttrsValuesListOptions) ([]schema.AttrsValueRow, error) {
	res := make([]schema.AttrsValueRow, 0)

	err := options.Select(s.db).Select(
		goqu.T(query.AttrsValuesAlias).Col(schema.AttrsValuesTableAttributeIDColName),
		goqu.T(query.AttrsValuesAlias).Col(schema.AttrsValuesTableItemIDColName),
	).
		ScanStructsContext(ctx, &res)

	return res, err
}

func (s *Repository) ActualValue(ctx context.Context, attributeID int64, itemID int64) (Value, error) {
	success, attribute, err := s.Attribute(ctx, attributeID)
	if err != nil {
		return Value{}, err
	}

	if !success {
		return Value{}, fmt.Errorf("%w: `%d`", errAttributeNotFound, attributeID)
	}

	if !attribute.TypeID.Valid {
		return Value{}, nil
	}

	switch attribute.TypeID.AttributeTypeID {
	case schema.AttrsAttributeTypeIDString, schema.AttrsAttributeTypeIDText:
		var value sql.NullString

		success, err = s.db.Select(schema.AttrsValuesStringTableValueCol).From(schema.AttrsValuesStringTable).Where(
			schema.AttrsValuesStringTableAttributeIDCol.Eq(attributeID),
			schema.AttrsValuesStringTableItemIDCol.Eq(itemID),
		).ScanValContext(ctx, &value)
		if err != nil {
			return Value{}, err
		}

		return Value{
			Valid:       success,
			StringValue: value.String,
			Type:        attribute.TypeID.AttributeTypeID,
			IsEmpty:     !value.Valid,
		}, nil

	case schema.AttrsAttributeTypeIDInteger:
		var value sql.NullInt32

		success, err = s.db.Select(schema.AttrsValuesIntTableValueCol).From(schema.AttrsValuesIntTable).Where(
			schema.AttrsValuesIntTableAttributeIDCol.Eq(attributeID),
			schema.AttrsValuesIntTableItemIDCol.Eq(itemID),
		).ScanValContext(ctx, &value)
		if err != nil {
			return Value{}, err
		}

		return Value{
			Valid:    success,
			IntValue: value.Int32,
			Type:     attribute.TypeID.AttributeTypeID,
			IsEmpty:  !value.Valid,
		}, nil
	case schema.AttrsAttributeTypeIDBoolean:
		var value sql.NullBool

		success, err = s.db.Select(schema.AttrsValuesIntTableValueCol).From(schema.AttrsValuesIntTable).Where(
			schema.AttrsValuesIntTableAttributeIDCol.Eq(attributeID),
			schema.AttrsValuesIntTableItemIDCol.Eq(itemID),
		).ScanValContext(ctx, &value)
		if err != nil {
			return Value{}, err
		}

		return Value{
			Valid:     success,
			BoolValue: value.Bool,
			Type:      attribute.TypeID.AttributeTypeID,
			IsEmpty:   !value.Valid,
		}, nil

	case schema.AttrsAttributeTypeIDFloat:
		var value sql.NullFloat64

		success, err = s.db.Select(schema.AttrsValuesFloatTableValueCol).From(schema.AttrsValuesFloatTable).Where(
			schema.AttrsValuesFloatTableAttributeIDCol.Eq(attributeID),
			schema.AttrsValuesFloatTableItemIDCol.Eq(itemID),
		).ScanValContext(ctx, &value)
		if err != nil {
			return Value{}, err
		}

		return Value{
			Valid:      success,
			FloatValue: value.Float64,
			Type:       attribute.TypeID.AttributeTypeID,
			IsEmpty:    !value.Valid,
		}, nil
	case schema.AttrsAttributeTypeIDList, schema.AttrsAttributeTypeIDTree:
		var values []sql.NullInt64

		err = s.db.Select(schema.AttrsValuesListTableValueCol).From(schema.AttrsValuesListTable).Where(
			schema.AttrsValuesListTableAttributeIDCol.Eq(attributeID),
			schema.AttrsValuesListTableItemIDCol.Eq(itemID),
		).Order(schema.AttrsValuesListTableOrderingCol.Asc()).ScanValsContext(ctx, &values)
		if err != nil {
			return Value{}, err
		}

		vals := make([]int64, 0, len(values))
		isEmpty := false

		for _, val := range values {
			if !val.Valid {
				isEmpty = true

				break
			}

			vals = append(vals, val.Int64)
		}

		return Value{
			Valid:     len(vals) > 0 || isEmpty,
			ListValue: vals,
			Type:      attribute.TypeID.AttributeTypeID,
			IsEmpty:   isEmpty,
		}, nil

	case schema.AttrsAttributeTypeIDUnknown:
	}

	return Value{}, nil
}

func (s *Repository) ActualValueText(
	ctx context.Context, attributeID int64, itemID int64, language string,
) (Value, string, error) {
	value, err := s.ActualValue(ctx, attributeID, itemID)
	if err != nil {
		return Value{}, "", err
	}

	text, err := s.valueToText(ctx, attributeID, value, language)

	return value, text, err
}

func (s *Repository) valueToText(ctx context.Context, attributeID int64, value Value, lang string) (string, error) {
	if !value.Valid {
		return "", nil
	}

	success, attribute, err := s.Attribute(ctx, attributeID)
	if err != nil {
		return "", err
	}

	if !success {
		return "", fmt.Errorf("%w: `%d`", errAttributeNotFound, attributeID)
	}

	switch value.Type {
	case schema.AttrsAttributeTypeIDString:
		return value.StringValue, nil

	case schema.AttrsAttributeTypeIDInteger:
		tag := language.Make(lang)
		printer := message.NewPrinter(tag)
		dec := number.Decimal(value.IntValue)

		return printer.Sprint(dec), nil

	case schema.AttrsAttributeTypeIDFloat:
		tag := language.Make(lang)
		printer := message.NewPrinter(tag)
		opts := make([]number.Option, 0)

		if attribute.Precision.Valid {
			opts = append(opts,
				number.MinFractionDigits(int(attribute.Precision.Int32)),
				number.MaxFractionDigits(int(attribute.Precision.Int32)),
			)
		}

		dec := number.Decimal(value.FloatValue, opts...)

		return printer.Sprint(dec), nil

	case schema.AttrsAttributeTypeIDText:
		return value.StringValue, nil

	case schema.AttrsAttributeTypeIDBoolean:
		if value.BoolValue {
			return "да", nil
		}

		return "нет", nil

	case schema.AttrsAttributeTypeIDList, schema.AttrsAttributeTypeIDTree:
		text := make([]string, 0, len(value.ListValue))

		for _, v := range value.ListValue {
			option, err := s.ListOptionsText(ctx, attribute.ID, v, lang)
			if err != nil {
				return "", err
			}

			text = append(text, option)
		}

		return strings.Join(text, ", "), nil

	case schema.AttrsAttributeTypeIDUnknown:
	}

	return "", nil
}

func (s *Repository) ListOptionsText(ctx context.Context, attributeID int64, id int64, lang string) (string, error) {
	err := s.loadListOptions(ctx)
	if err != nil {
		return "", err
	}

	if _, ok := s.listOptions[attributeID][id]; !ok {
		return "", fmt.Errorf("%w: `%d`", errAttributeNotFound, id)
	}

	localizer := s.i18n.Localizer(lang)

	return localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: s.listOptions[attributeID][id],
		},
	})
}

func (s *Repository) loadListOptions(ctx context.Context) error {
	var rows []schema.AttrsListOptionRow

	err := s.db.Select(
		schema.AttrsListOptionsTableAttributeIDCol,
		schema.AttrsListOptionsTableIDCol,
		schema.AttrsListOptionsTableParentIDCol,
		schema.AttrsListOptionsTableNameCol,
	).
		From(schema.AttrsListOptionsTable).
		Order(schema.AttrsListOptionsTablePositionCol.Asc()).
		ScanStructsContext(ctx, &rows)
	if err != nil {
		return err
	}

	for _, row := range rows {
		aid := row.AttributeID
		id := row.ID

		var pid int64

		if row.ParentID.Valid {
			pid = row.ParentID.Int64
		}

		if _, ok := s.listOptions[aid]; !ok {
			s.listOptions[aid] = make(map[int64]string)
		}

		s.listOptions[aid][id] = row.Name

		if _, ok := s.listOptionsChilds[aid]; !ok {
			s.listOptionsChilds[aid] = make(map[int64][]int64)
		}

		if _, ok := s.listOptionsChilds[aid][pid]; !ok {
			s.listOptionsChilds[aid][pid] = []int64{id}
		} else {
			s.listOptionsChilds[aid][pid] = append(s.listOptionsChilds[aid][pid], id)
		}
	}

	return nil
}
