package attrs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

type ValuesOrderBy int

var (
	errAttributeNotFound = errors.New("attribute not found")
	errListOptionFound   = errors.New("listOption not found")
	errInvalidItemID     = errors.New("invalid itemID provided")
)

const (
	defaultZoneID = 1
	engineZoneID  = 5
	busZoneID     = 3
)

const (
	ValuesOrderByNone ValuesOrderBy = iota
	ValuesOrderByUpdateDate
)

type TopUserBrand struct {
	ID      int64  `db:"id"`
	Name    string `db:"name"`
	Catname string `db:"catname"`
	Volume  int64  `db:"volume"`
}

// Repository Main Object.
type Repository struct {
	db                *goqu.Database
	i18n              *i18nbundle.I18n
	listOptions       map[int64]map[int64]string
	listOptionsMutex  sync.RWMutex
	listOptionsChilds map[int64]map[int64][]int64
	engineAttributes  []int64
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
		listOptionsMutex:  sync.RWMutex{},
		listOptionsChilds: make(map[int64]map[int64][]int64),
		engineAttributes:  make([]int64, 0),
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

func (s *Repository) ValuesSelect(options query.AttrsValuesListOptions, orderBy ValuesOrderBy) *goqu.SelectDataset {
	aliasTable := goqu.T(query.AttrsValuesAlias)

	sqSelect := options.Select(s.db).Select(
		aliasTable.Col(schema.AttrsValuesTableAttributeIDColName),
		aliasTable.Col(schema.AttrsValuesTableItemIDColName),
	)

	if orderBy == ValuesOrderByUpdateDate {
		sqSelect = sqSelect.Order(aliasTable.Col(schema.AttrsValuesTableUpdateDateColName).Desc())
	}

	return sqSelect
}

func (s *Repository) ValuesPaginated(
	ctx context.Context, options query.AttrsValuesListOptions, orderBy ValuesOrderBy, page int32, limit int32,
) ([]schema.AttrsValueRow, *util.Pages, error) {
	sqSelect := s.ValuesSelect(options, orderBy)

	paginator := util.Paginator{
		SQLSelect:         sqSelect,
		CurrentPageNumber: page,
		ItemCountPerPage:  limit,
	}

	pages, err := paginator.GetPages(ctx)
	if err != nil {
		return nil, nil, err
	}

	res := make([]schema.AttrsValueRow, 0)
	err = sqSelect.ScanStructsContext(ctx, &res)

	return res, pages, err
}

func (s *Repository) Values(
	ctx context.Context, options query.AttrsValuesListOptions, orderBy ValuesOrderBy,
) ([]schema.AttrsValueRow, error) {
	sqSelect := s.ValuesSelect(options, orderBy)
	res := make([]schema.AttrsValueRow, 0)
	err := sqSelect.ScanStructsContext(ctx, &res)

	return res, err
}

func (s *Repository) UserValueRows(
	ctx context.Context, options query.AttrsUserValuesListOptions,
) ([]schema.AttrsUserValueRow, error) {
	res := make([]schema.AttrsUserValueRow, 0)

	err := options.Select(s.db).Select(
		goqu.T(query.AttrsUserValuesAlias).Col(schema.AttrsUserValuesTableAttributeIDColName),
		goqu.T(query.AttrsUserValuesAlias).Col(schema.AttrsUserValuesTableItemIDColName),
		goqu.T(query.AttrsUserValuesAlias).Col(schema.AttrsUserValuesTableUserIDColName),
		goqu.T(query.AttrsUserValuesAlias).Col(schema.AttrsUserValuesTableUpdateDateColName),
	).
		ScanStructsContext(ctx, &res)

	return res, err
}

func (s *Repository) UserValueRow(
	ctx context.Context, options query.AttrsUserValuesListOptions,
) (schema.AttrsUserValueRow, bool, error) {
	var row schema.AttrsUserValueRow

	success, err := options.Select(s.db).Select(
		goqu.T(query.AttrsUserValuesAlias).Col(schema.AttrsUserValuesTableAttributeIDColName),
		goqu.T(query.AttrsUserValuesAlias).Col(schema.AttrsUserValuesTableItemIDColName),
		goqu.T(query.AttrsUserValuesAlias).Col(schema.AttrsUserValuesTableUserIDColName),
		goqu.T(query.AttrsUserValuesAlias).Col(schema.AttrsUserValuesTableUpdateDateColName),
	).ScanStructContext(ctx, &row)

	return row, success, err
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

func (s *Repository) UserValue(ctx context.Context, attributeID int64, itemID int64, userID int64) (Value, error) {
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

		success, err = s.db.Select(schema.AttrsUserValuesStringTableValueCol).From(schema.AttrsUserValuesStringTable).Where(
			schema.AttrsUserValuesStringTableAttributeIDCol.Eq(attributeID),
			schema.AttrsUserValuesStringTableItemIDCol.Eq(itemID),
			schema.AttrsUserValuesStringTableUserIDCol.Eq(userID),
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

		success, err = s.db.Select(schema.AttrsUserValuesIntTableValueCol).From(schema.AttrsUserValuesIntTable).Where(
			schema.AttrsUserValuesIntTableAttributeIDCol.Eq(attributeID),
			schema.AttrsUserValuesIntTableItemIDCol.Eq(itemID),
			schema.AttrsUserValuesIntTableUserIDCol.Eq(userID),
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

		success, err = s.db.Select(schema.AttrsUserValuesIntTableValueCol).From(schema.AttrsUserValuesIntTable).Where(
			schema.AttrsUserValuesIntTableAttributeIDCol.Eq(attributeID),
			schema.AttrsUserValuesIntTableItemIDCol.Eq(itemID),
			schema.AttrsUserValuesIntTableUserIDCol.Eq(userID),
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

		success, err = s.db.Select(schema.AttrsUserValuesFloatTableValueCol).From(schema.AttrsUserValuesFloatTable).Where(
			schema.AttrsUserValuesFloatTableAttributeIDCol.Eq(attributeID),
			schema.AttrsUserValuesFloatTableItemIDCol.Eq(itemID),
			schema.AttrsUserValuesFloatTableUserIDCol.Eq(userID),
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

		err = s.db.Select(schema.AttrsUserValuesListTableValueCol).From(schema.AttrsUserValuesListTable).Where(
			schema.AttrsUserValuesListTableAttributeIDCol.Eq(attributeID),
			schema.AttrsUserValuesListTableItemIDCol.Eq(itemID),
			schema.AttrsUserValuesListTableUserIDCol.Eq(userID),
		).Order(schema.AttrsUserValuesListTableOrderingCol.Asc()).ScanValsContext(ctx, &values)
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

func (s *Repository) UserValueText(
	ctx context.Context, attributeID int64, itemID int64, userID int64, language string,
) (Value, string, error) {
	value, err := s.UserValue(ctx, attributeID, itemID, userID)
	if err != nil {
		return Value{}, "", err
	}

	text, err := s.valueToText(ctx, attributeID, value, language)

	return value, text, err
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
		return "", fmt.Errorf("%w: `%d`", errListOptionFound, id)
	}

	localizer := s.i18n.Localizer(lang)

	return localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: s.listOptions[attributeID][id],
		},
	})
}

func (s *Repository) loadListOptions(ctx context.Context) error {
	s.listOptionsMutex.Lock()
	defer s.listOptionsMutex.Unlock()

	if len(s.listOptions) > 0 {
		return nil
	}

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

func (s *Repository) DeleteUserValue(ctx context.Context, attributeID, itemID, userID int64) error {
	success, attribute, err := s.Attribute(ctx, attributeID)
	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("%w: `%d`", errAttributeNotFound, attributeID)
	}

	switch attribute.TypeID.AttributeTypeID {
	case schema.AttrsAttributeTypeIDString, schema.AttrsAttributeTypeIDText:
		_, err = s.db.Delete(schema.AttrsUserValuesStringTable).Where(
			schema.AttrsUserValuesStringTableAttributeIDCol.Eq(attributeID),
			schema.AttrsUserValuesStringTableItemIDCol.Eq(itemID),
			schema.AttrsUserValuesStringTableUserIDCol.Eq(userID),
		).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}

	case schema.AttrsAttributeTypeIDInteger, schema.AttrsAttributeTypeIDBoolean:
		_, err = s.db.Delete(schema.AttrsUserValuesIntTable).Where(
			schema.AttrsUserValuesIntTableAttributeIDCol.Eq(attributeID),
			schema.AttrsUserValuesIntTableItemIDCol.Eq(itemID),
			schema.AttrsUserValuesIntTableUserIDCol.Eq(userID),
		).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}

	case schema.AttrsAttributeTypeIDFloat:
		_, err = s.db.Delete(schema.AttrsUserValuesFloatTable).Where(
			schema.AttrsUserValuesFloatTableAttributeIDCol.Eq(attributeID),
			schema.AttrsUserValuesFloatTableItemIDCol.Eq(itemID),
			schema.AttrsUserValuesFloatTableUserIDCol.Eq(userID),
		).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}

	case schema.AttrsAttributeTypeIDList, schema.AttrsAttributeTypeIDTree:
		_, err = s.db.Delete(schema.AttrsUserValuesListTable).Where(
			schema.AttrsUserValuesListTableAttributeIDCol.Eq(attributeID),
			schema.AttrsUserValuesListTableItemIDCol.Eq(itemID),
			schema.AttrsUserValuesListTableUserIDCol.Eq(userID),
		).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}

	case schema.AttrsAttributeTypeIDUnknown:
	}

	_, err = s.db.Delete(schema.AttrsUserValuesTable).Where(
		schema.AttrsUserValuesTableAttributeIDCol.Eq(attributeID),
		schema.AttrsUserValuesTableItemIDCol.Eq(itemID),
		schema.AttrsUserValuesTableUserIDCol.Eq(userID),
	).Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	err = s.updateActualValue(ctx, attributeID, itemID)
	if err != nil {
		return fmt.Errorf("%w: updateActualValue(%d, %d)", errAttributeNotFound, attributeID, itemID)
	}

	return nil
}

func (s *Repository) updateActualValue(ctx context.Context, attributeID, itemID int64) error {
	success, attribute, err := s.Attribute(ctx, attributeID)
	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("%w: `%d`", errAttributeNotFound, attributeID)
	}

	_, err = s.updateAttributeActualValue(ctx, attribute, itemID)
	if err != nil {
		return fmt.Errorf("%w: updateAttributeActualValue(%d, %d)", errAttributeNotFound, attribute.ID, itemID)
	}

	return nil
}

func (s *Repository) updateAttributeActualValue(
	ctx context.Context, attribute schema.AttrsAttributeRow, itemID int64,
) (bool, error) {
	actualValue, err := s.calcAvgUserValue(ctx, attribute, itemID)
	if err != nil {
		return false, err
	}

	if !actualValue.Valid {
		actualValue, err = s.calcEngineValue(ctx, attribute.ID, itemID)
		if err != nil {
			return false, err
		}
	}

	if !actualValue.Valid {
		actualValue, err = s.calcInheritedValue(ctx, attribute.ID, itemID)
		if err != nil {
			return false, err
		}
	}

	return s.setActualValue(ctx, attribute, itemID, actualValue)
}

type valueItem struct {
	Value Value
	Row   schema.AttrsUserValueRow
}

func (s *Repository) topValue(ctx context.Context, data []valueItem) (Value, error) {
	if len(data) == 0 {
		return Value{}, nil
	}

	idx := 0
	registry := make(map[int]Value, 0)
	freshness := make(map[int]time.Time)
	ratios := make(map[int]float32)

	for _, valueRow := range data {
		// look for same value
		matchRegIdx := -1

		for regIdx, regVal := range registry {
			if regVal.Equals(valueRow.Value) {
				matchRegIdx = regIdx

				break
			}
		}

		if matchRegIdx == -1 {
			registry[idx] = valueRow.Value
			matchRegIdx = idx
			idx++
		}

		if _, ok := ratios[matchRegIdx]; !ok {
			ratios[matchRegIdx] = 0
		}

		weight, err := s.getUserValueWeight(ctx, valueRow.Row.UserID)
		if err != nil {
			return Value{}, err
		}

		ratios[matchRegIdx] += weight

		_, freshnessExists := freshness[matchRegIdx]
		if !freshnessExists || freshness[matchRegIdx].Before(valueRow.Row.UpdateDate) {
			freshness[matchRegIdx] = valueRow.Row.UpdateDate
		}
	}

	// select max
	var (
		maxValueRatio float32
		maxValueIdx   = -1
	)

	for idx, ratio := range ratios {
		if maxValueIdx == -1 || maxValueRatio <= ratio {
			maxValueIdx = idx
			maxValueRatio = ratio
		}
	}

	actualValue := registry[maxValueIdx]

	return actualValue, nil
}

func (s *Repository) calcAvgUserValue(
	ctx context.Context, attribute schema.AttrsAttributeRow, itemID int64,
) (Value, error) {
	userValueRows, err := s.UserValueRows(ctx, query.AttrsUserValuesListOptions{
		AttributeID: attribute.ID,
		ItemID:      itemID,
	})
	if err != nil {
		return Value{}, err
	}

	// group by users
	data := make([]valueItem, 0, len(userValueRows))

	for _, userValueRow := range userValueRows {
		uid := userValueRow.UserID

		value, err := s.UserValue(ctx, userValueRow.AttributeID, userValueRow.ItemID, uid)
		if err != nil {
			return Value{}, err
		}

		data = append(data, valueItem{
			Value: value,
			Row:   userValueRow,
		})
	}

	return s.topValue(ctx, data)
}

func (s *Repository) getUserValueWeight(ctx context.Context, userID int64) (float32, error) {
	var weight float32

	success, err := s.db.Select(schema.UserTableSpecsWeightCol).
		From(schema.UserTable).
		Where(schema.UserTableIDCol.Eq(userID)).
		ScanValContext(ctx, &weight)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, nil
	}

	return weight, nil
}

func (s *Repository) getEngineAttributeIDs(ctx context.Context) ([]int64, error) {
	if len(s.engineAttributes) > 0 {
		return s.engineAttributes, nil
	}

	err := s.db.Select(schema.AttrsZoneAttributesTableAttributeIDCol).
		From(schema.AttrsZoneAttributesTable).
		Where(schema.AttrsZoneAttributesTableZoneIDCol.Eq(engineZoneID)).ScanValsContext(ctx, &s.engineAttributes)

	return s.engineAttributes, err
}

func (s *Repository) isEngineAttributeID(ctx context.Context, attrID int64) (bool, error) {
	ids, err := s.getEngineAttributeIDs(ctx)
	if err != nil {
		return false, err
	}

	return util.Contains(ids, attrID), nil
}

func (s *Repository) calcEngineValue(ctx context.Context, attributeID int64, itemID int64) (Value, error) {
	isEngineAttributeID, err := s.isEngineAttributeID(ctx, attributeID)
	if err != nil {
		return Value{}, err
	}

	if !isEngineAttributeID {
		return Value{}, nil
	}

	var engineItemID sql.NullInt64

	success, err := s.db.Select(schema.ItemTableEngineItemIDCol).
		From(schema.ItemTable).
		Where(schema.ItemTableIDCol.Eq(itemID)).
		ScanValContext(ctx, &engineItemID)
	if err != nil {
		return Value{}, err
	}

	if !success || !engineItemID.Valid {
		return Value{}, nil
	}

	return s.ActualValue(ctx, attributeID, engineItemID.Int64)
}

func (s *Repository) calcInheritedValue(ctx context.Context, attributeID int64, itemID int64) (Value, error) {
	valueRows, err := s.Values(ctx, query.AttrsValuesListOptions{
		AttributeID: attributeID,
		ChildItemID: itemID,
	}, ValuesOrderByNone)
	if err != nil {
		return Value{}, err
	}

	idx := 0
	registry := make(map[int]Value)
	ratios := make(map[int]int)

	for _, valueRow := range valueRows {
		value, err := s.ActualValue(ctx, valueRow.AttributeID, valueRow.ItemID)
		if err != nil {
			return Value{}, err
		}

		// look for same value
		matchRegIdx := -1

		for regIdx, regVal := range registry {
			if regVal.Equals(value) {
				matchRegIdx = regIdx

				break
			}
		}

		if matchRegIdx == -1 {
			registry[idx] = value
			matchRegIdx = idx
			idx++
		}

		if _, ok := ratios[matchRegIdx]; !ok {
			ratios[matchRegIdx] = 0
		}

		ratios[matchRegIdx]++
	}

	// select max
	maxValueRatio := 0
	maxValueIdx := -1

	for idx, ratio := range ratios {
		if maxValueIdx == -1 || maxValueRatio <= ratio {
			maxValueIdx = idx
			maxValueRatio = ratio
		}
	}

	if maxValueIdx != -1 {
		return registry[maxValueIdx], nil
	}

	return Value{}, nil
}

func (s *Repository) clearStringValue(ctx context.Context, attributeID, itemID int64) (bool, error) {
	res, err := s.db.Delete(schema.AttrsValuesStringTable).Where(
		schema.AttrsValuesStringTableAttributeIDCol.Eq(attributeID),
		schema.AttrsValuesStringTableItemIDCol.Eq(itemID),
	).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, err
}

func (s *Repository) clearIntValue(ctx context.Context, attributeID, itemID int64) (bool, error) {
	res, err := s.db.Delete(schema.AttrsValuesIntTable).Where(
		schema.AttrsValuesIntTableAttributeIDCol.Eq(attributeID),
		schema.AttrsValuesIntTableItemIDCol.Eq(itemID),
	).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, err
}

func (s *Repository) clearFloatValue(ctx context.Context, attributeID, itemID int64) (bool, error) {
	res, err := s.db.Delete(schema.AttrsValuesFloatTable).Where(
		schema.AttrsValuesFloatTableAttributeIDCol.Eq(attributeID),
		schema.AttrsValuesFloatTableItemIDCol.Eq(itemID),
	).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, err
}

func (s *Repository) clearListValue(ctx context.Context, attributeID, itemID int64) (bool, error) {
	res, err := s.db.Delete(schema.AttrsValuesListTable).Where(
		schema.AttrsValuesListTableAttributeIDCol.Eq(attributeID),
		schema.AttrsValuesListTableItemIDCol.Eq(itemID),
	).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, err
}

func (s *Repository) clearValue(ctx context.Context, attribute schema.AttrsAttributeRow, itemID int64) (bool, error) {
	var (
		somethingChanges = false
		err              error
		res              sql.Result
	)

	// value
	switch attribute.TypeID.AttributeTypeID {
	case schema.AttrsAttributeTypeIDString, schema.AttrsAttributeTypeIDText:
		somethingChanges, err = s.clearStringValue(ctx, attribute.ID, itemID)

	case schema.AttrsAttributeTypeIDInteger, schema.AttrsAttributeTypeIDBoolean:
		somethingChanges, err = s.clearIntValue(ctx, attribute.ID, itemID)

	case schema.AttrsAttributeTypeIDFloat:
		somethingChanges, err = s.clearFloatValue(ctx, attribute.ID, itemID)

	case schema.AttrsAttributeTypeIDList, schema.AttrsAttributeTypeIDTree:
		somethingChanges, err = s.clearListValue(ctx, attribute.ID, itemID)

	case schema.AttrsAttributeTypeIDUnknown:
	}

	if err != nil {
		return false, err
	}

	// descriptor
	res, err = s.db.Delete(schema.AttrsValuesTable).Where(
		schema.AttrsValuesTableAttributeIDCol.Eq(attribute.ID),
		schema.AttrsValuesTableItemIDCol.Eq(itemID),
	).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return somethingChanges || affected > 0, nil
}

func (s *Repository) setStringValue(
	ctx context.Context, attributeID, itemID int64, value string, isEmpty bool,
) (bool, error) {
	res, err := s.db.Insert(schema.AttrsValuesStringTable).Rows(goqu.Record{
		schema.AttrsValuesStringTableAttributeIDColName: attributeID,
		schema.AttrsValuesStringTableItemIDColName:      itemID,
		schema.AttrsValuesStringTableValueColName: sql.NullString{
			String: value,
			Valid:  !isEmpty,
		},
	}).OnConflict(
		goqu.DoUpdate(
			schema.AttrsValuesStringTableAttributeIDColName+","+schema.AttrsValuesStringTableItemIDColName,
			goqu.Record{
				schema.AttrsValuesStringTableValueColName: goqu.Func(
					"VALUES",
					goqu.C(schema.AttrsValuesStringTableValueColName),
				),
			},
		)).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()

	return affected > 0, err
}

func (s *Repository) setIntValue(
	ctx context.Context, attributeID, itemID int64, value int32, isEmpty bool,
) (bool, error) {
	res, err := s.db.Insert(schema.AttrsValuesIntTable).Rows(goqu.Record{
		schema.AttrsValuesIntTableAttributeIDColName: attributeID,
		schema.AttrsValuesIntTableItemIDColName:      itemID,
		schema.AttrsValuesIntTableValueColName: sql.NullInt32{
			Int32: value,
			Valid: !isEmpty,
		},
	}).OnConflict(
		goqu.DoUpdate(
			schema.AttrsValuesIntTableAttributeIDColName+","+schema.AttrsValuesIntTableItemIDColName,
			goqu.Record{
				schema.AttrsValuesIntTableValueColName: goqu.Func(
					"VALUES",
					goqu.C(schema.AttrsValuesIntTableValueColName),
				),
			},
		)).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()

	return affected > 0, err
}

func (s *Repository) setFloatValue(
	ctx context.Context, attributeID, itemID int64, value float64, isEmpty bool,
) (bool, error) {
	res, err := s.db.Insert(schema.AttrsValuesFloatTable).Rows(goqu.Record{
		schema.AttrsValuesFloatTableAttributeIDColName: attributeID,
		schema.AttrsValuesFloatTableItemIDColName:      itemID,
		schema.AttrsValuesFloatTableValueColName: sql.NullFloat64{
			Float64: value,
			Valid:   !isEmpty,
		},
	}).OnConflict(
		goqu.DoUpdate(
			schema.AttrsValuesFloatTableAttributeIDColName+","+schema.AttrsValuesFloatTableItemIDColName,
			goqu.Record{
				schema.AttrsValuesFloatTableValueColName: goqu.Func(
					"VALUES",
					goqu.C(schema.AttrsValuesFloatTableValueColName),
				),
			},
		)).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()

	return affected > 0, err
}

func (s *Repository) setListValue(
	ctx context.Context, attributeID, itemID int64, value []int64, isEmpty bool,
) (bool, error) {
	var ( //nolint: prealloc
		records   []goqu.Record
		orderings []int
	)

	if isEmpty {
		records = []goqu.Record{{
			schema.AttrsValuesListTableAttributeIDColName: attributeID,
			schema.AttrsValuesListTableItemIDColName:      itemID,
			schema.AttrsValuesListTableOrderingColName:    0,
			schema.AttrsValuesListTableValueColName:       nil,
		}}
		orderings = []int{0}
	} else {
		records = make([]goqu.Record, 0, len(value))
		orderings = make([]int, 0, len(value))
	}

	for index, listValue := range value {
		records = append(records, goqu.Record{
			schema.AttrsValuesListTableAttributeIDColName: attributeID,
			schema.AttrsValuesListTableItemIDColName:      itemID,
			schema.AttrsValuesListTableOrderingColName:    index,
			schema.AttrsValuesListTableValueColName:       listValue,
		})

		orderings = append(orderings, index)
	}

	res, err := s.db.Insert(schema.AttrsValuesListTable).Rows(records).OnConflict(
		goqu.DoUpdate(
			schema.AttrsValuesListTableAttributeIDColName+","+
				schema.AttrsValuesListTableItemIDColName+","+
				schema.AttrsValuesListTableOrderingColName,
			goqu.Record{
				schema.AttrsValuesListTableValueColName: goqu.Func(
					"VALUES",
					goqu.C(schema.AttrsValuesListTableValueColName),
				),
			},
		)).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	somethingChanges := affected > 0

	_, err = s.db.Delete(schema.AttrsValuesListTable).Where(
		schema.AttrsValuesListTableAttributeIDCol.Eq(attributeID),
		schema.AttrsValuesListTableItemIDCol.Eq(itemID),
		schema.AttrsValuesListTableOrderingCol.NotIn(orderings),
	).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err = res.RowsAffected()

	return somethingChanges || affected > 0, err
}

func (s *Repository) setActualValue(
	ctx context.Context, attribute schema.AttrsAttributeRow, itemID int64, actualValue Value,
) (bool, error) {
	if !attribute.TypeID.Valid {
		return false, nil
	}

	if !actualValue.Valid {
		return s.clearValue(ctx, attribute, itemID)
	}

	var err error

	// descriptor
	_, err = s.db.Insert(schema.AttrsValuesTable).Rows(goqu.Record{
		schema.AttrsValuesTableAttributeIDColName: attribute.ID,
		schema.AttrsValuesTableItemIDColName:      itemID,
		schema.AttrsValuesTableUpdateDateColName:  goqu.Func("NOW"),
	}).OnConflict(
		goqu.DoUpdate(
			schema.AttrsValuesTableAttributeIDColName+","+schema.AttrsValuesTableItemIDColName,
			goqu.Record{
				schema.AttrsValuesTableUpdateDateColName: goqu.Func(
					"VALUES",
					goqu.C(schema.AttrsValuesTableUpdateDateColName),
				),
			},
		)).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	// value
	valueChanged := false

	switch attribute.TypeID.AttributeTypeID {
	case schema.AttrsAttributeTypeIDString, schema.AttrsAttributeTypeIDText:
		valueChanged, err = s.setStringValue(ctx, attribute.ID, itemID, actualValue.StringValue, actualValue.IsEmpty)

	case schema.AttrsAttributeTypeIDInteger:
		valueChanged, err = s.setIntValue(ctx, attribute.ID, itemID, actualValue.IntValue, actualValue.IsEmpty)

	case schema.AttrsAttributeTypeIDBoolean:
		var value int32
		if actualValue.BoolValue {
			value = 1
		}

		valueChanged, err = s.setIntValue(ctx, attribute.ID, itemID, value, actualValue.IsEmpty)

	case schema.AttrsAttributeTypeIDFloat:
		valueChanged, err = s.setFloatValue(ctx, attribute.ID, itemID, actualValue.FloatValue, actualValue.IsEmpty)

	case schema.AttrsAttributeTypeIDList, schema.AttrsAttributeTypeIDTree:
		valueChanged, err = s.setListValue(ctx, attribute.ID, itemID, actualValue.ListValue, actualValue.IsEmpty)

	case schema.AttrsAttributeTypeIDUnknown:
	}

	return valueChanged, err
}

func (s *Repository) setStringUserValue(
	ctx context.Context, attributeID, itemID, userID int64, value string, isEmpty bool,
) (bool, error) {
	res, err := s.db.Insert(schema.AttrsUserValuesStringTable).Rows(goqu.Record{
		schema.AttrsUserValuesStringTableAttributeIDColName: attributeID,
		schema.AttrsUserValuesStringTableItemIDColName:      itemID,
		schema.AttrsUserValuesStringTableUserIDColName:      userID,
		schema.AttrsUserValuesStringTableValueColName: sql.NullString{
			String: value,
			Valid:  !isEmpty,
		},
	}).OnConflict(
		goqu.DoUpdate(
			schema.AttrsUserValuesStringTableAttributeIDColName+
				","+schema.AttrsUserValuesStringTableItemIDColName+
				","+schema.AttrsUserValuesStringTableUserIDColName,
			goqu.Record{
				schema.AttrsUserValuesStringTableValueColName: goqu.Func(
					"VALUES",
					goqu.C(schema.AttrsUserValuesStringTableValueColName),
				),
			},
		)).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()

	return affected > 0, err
}

func (s *Repository) setIntUserValue(
	ctx context.Context, attributeID, itemID, userID int64, value int32, isEmpty bool,
) (bool, error) {
	res, err := s.db.Insert(schema.AttrsUserValuesIntTable).Rows(goqu.Record{
		schema.AttrsUserValuesIntTableAttributeIDColName: attributeID,
		schema.AttrsUserValuesIntTableItemIDColName:      itemID,
		schema.AttrsUserValuesIntTableUserIDColName:      userID,
		schema.AttrsUserValuesIntTableValueColName: sql.NullInt32{
			Int32: value,
			Valid: !isEmpty,
		},
	}).OnConflict(
		goqu.DoUpdate(
			schema.AttrsUserValuesIntTableAttributeIDColName+
				","+schema.AttrsUserValuesIntTableItemIDColName+
				","+schema.AttrsUserValuesIntTableUserIDColName,
			goqu.Record{
				schema.AttrsUserValuesIntTableValueColName: goqu.Func(
					"VALUES",
					goqu.C(schema.AttrsUserValuesIntTableValueColName),
				),
			},
		)).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()

	return affected > 0, err
}

func (s *Repository) setFloatUserValue(
	ctx context.Context, attributeID, itemID, userID int64, value float64, isEmpty bool,
) (bool, error) {
	res, err := s.db.Insert(schema.AttrsUserValuesFloatTable).Rows(goqu.Record{
		schema.AttrsUserValuesFloatTableAttributeIDColName: attributeID,
		schema.AttrsUserValuesFloatTableItemIDColName:      itemID,
		schema.AttrsUserValuesFloatTableUserIDColName:      userID,
		schema.AttrsUserValuesFloatTableValueColName: sql.NullFloat64{
			Float64: value,
			Valid:   !isEmpty,
		},
	}).OnConflict(
		goqu.DoUpdate(
			schema.AttrsUserValuesFloatTableAttributeIDColName+
				","+schema.AttrsUserValuesFloatTableItemIDColName+
				","+schema.AttrsUserValuesFloatTableUserIDColName,
			goqu.Record{
				schema.AttrsUserValuesFloatTableValueColName: goqu.Func(
					"VALUES",
					goqu.C(schema.AttrsUserValuesFloatTableValueColName),
				),
			},
		)).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()

	return affected > 0, err
}

func (s *Repository) setListUserValue(
	ctx context.Context, attributeID, itemID, userID int64, value []int64, isEmpty bool,
) (bool, error) {
	var (
		err      error
		affected int64
		res      sql.Result
	)

	insertExpr := s.db.Insert(schema.AttrsUserValuesListTable).Cols(
		schema.AttrsUserValuesListTableAttributeIDCol,
		schema.AttrsUserValuesListTableItemIDCol,
		schema.AttrsUserValuesListTableUserIDCol,
		schema.AttrsUserValuesListTableOrderingCol,
		schema.AttrsUserValuesListTableValueCol,
	).
		OnConflict(
			goqu.DoUpdate(
				schema.AttrsUserValuesListTableAttributeIDColName+","+
					schema.AttrsUserValuesListTableItemIDColName+","+
					schema.AttrsUserValuesListTableUserIDColName+","+
					schema.AttrsUserValuesListTableOrderingColName,
				goqu.Record{
					schema.AttrsUserValuesListTableValueColName: goqu.Func(
						"VALUES",
						goqu.C(schema.AttrsUserValuesListTableValueColName),
					),
				},
			))
	if isEmpty {
		res, err = insertExpr.Vals([]interface{}{attributeID, itemID, userID, 0, nil}).Executor().ExecContext(ctx)
		if err != nil {
			return false, err
		}

		affected, err = res.RowsAffected()
		if err != nil {
			return false, err
		}
	} else if len(value) > 0 {
		res, err = insertExpr.
			FromQuery(
				s.db.Select(
					schema.AttrsListOptionsTableAttributeIDCol,
					goqu.V(itemID),
					goqu.V(userID),
					goqu.L("ROW_NUMBER() OVER(ORDER BY ?)", schema.AttrsListOptionsTablePositionCol),
					schema.AttrsListOptionsTableIDCol,
				).
					From(schema.AttrsListOptionsTable).
					Where(
						schema.AttrsListOptionsTableAttributeIDCol.Eq(attributeID),
						schema.AttrsListOptionsTableIDCol.In(value),
					),
			).Executor().ExecContext(ctx)
		if err != nil {
			return false, err
		}

		affected, err = res.RowsAffected()
		if err != nil {
			return false, err
		}
	}

	deleteExpr := s.db.Delete(schema.AttrsUserValuesListTable).Where(
		schema.AttrsUserValuesListTableAttributeIDCol.Eq(attributeID),
		schema.AttrsUserValuesListTableItemIDCol.Eq(itemID),
		schema.AttrsUserValuesListTableUserIDCol.Eq(userID),
	)

	if isEmpty {
		deleteExpr = deleteExpr.Where(schema.AttrsUserValuesListTableValueCol.IsNotNull())
	} else if len(value) > 0 {
		deleteExpr = deleteExpr.Where(schema.AttrsUserValuesListTableValueCol.NotIn(value))
	}

	res, err = deleteExpr.Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	deleted, err := res.RowsAffected()

	return deleted > 0 || affected > 0, err
}

func (s *Repository) SetUserValue(ctx context.Context, userID, attributeID, itemID int64, value Value) error {
	success, attribute, err := s.Attribute(ctx, attributeID)
	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("%w: `%d`", errAttributeNotFound, attributeID)
	}

	if !attribute.TypeID.Valid {
		return nil
	}

	if !value.Valid {
		return s.DeleteUserValue(ctx, attributeID, itemID, userID)
	}

	oldValue, err := s.UserValue(ctx, attributeID, itemID, userID)
	if err != nil {
		return err
	}

	if oldValue.Equals(value) {
		return nil
	}

	_, err = s.db.Insert(schema.AttrsUserValuesTable).Rows(goqu.Record{
		schema.AttrsUserValuesTableAttributeIDColName: attribute.ID,
		schema.AttrsUserValuesTableItemIDColName:      itemID,
		schema.AttrsUserValuesTableUserIDColName:      userID,
		schema.AttrsUserValuesTableAddDateColName:     goqu.Func("NOW"),
		schema.AttrsUserValuesTableUpdateDateColName:  goqu.Func("NOW"),
	}).OnConflict(
		goqu.DoUpdate(
			schema.AttrsUserValuesTableAttributeIDColName+
				","+schema.AttrsUserValuesTableItemIDColName+
				","+schema.AttrsUserValuesTableUserIDColName,
			goqu.Record{
				schema.AttrsUserValuesTableUpdateDateColName: goqu.Func(
					"VALUES",
					goqu.C(schema.AttrsUserValuesTableUpdateDateColName),
				),
			},
		)).Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	valueChanged := false

	switch attribute.TypeID.AttributeTypeID {
	case schema.AttrsAttributeTypeIDString, schema.AttrsAttributeTypeIDText:
		valueChanged, err = s.setStringUserValue(ctx, attribute.ID, itemID, userID, value.StringValue, value.IsEmpty)

	case schema.AttrsAttributeTypeIDInteger:
		valueChanged, err = s.setIntUserValue(ctx, attribute.ID, itemID, userID, value.IntValue, value.IsEmpty)

	case schema.AttrsAttributeTypeIDBoolean:
		var intValue int32
		if value.BoolValue {
			intValue = 1
		}

		valueChanged, err = s.setIntUserValue(ctx, attribute.ID, itemID, userID, intValue, value.IsEmpty)

	case schema.AttrsAttributeTypeIDFloat:
		valueChanged, err = s.setFloatUserValue(ctx, attribute.ID, itemID, userID, value.FloatValue, value.IsEmpty)

	case schema.AttrsAttributeTypeIDList, schema.AttrsAttributeTypeIDTree:
		valueChanged, err = s.setListUserValue(ctx, attribute.ID, itemID, userID, value.ListValue, value.IsEmpty)

	case schema.AttrsAttributeTypeIDUnknown:
	}

	if err != nil {
		return err
	}

	somethingChanged, err := s.updateAttributeActualValue(ctx, attribute, itemID)
	if err != nil {
		return fmt.Errorf("%w: updateAttributeActualValue(%d, %d)", errAttributeNotFound, attribute.ID, itemID)
	}

	if somethingChanged || valueChanged {
		err = s.propagateInheritance(ctx, attribute, itemID)
		if err != nil {
			return err
		}

		err = s.propagateEngine(ctx, attribute, itemID)
		if err != nil {
			return err
		}

		err = s.refreshConflictFlag(ctx, attribute.ID, itemID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) propagateInheritance(ctx context.Context, attribute schema.AttrsAttributeRow, itemID int64) error {
	var childIDs []int64

	err := s.db.Select(schema.ItemParentTableItemIDCol).
		From(schema.ItemParentTable).
		Where(schema.ItemParentTableParentIDCol.Eq(itemID)).
		ScanValsContext(ctx, &childIDs)
	if err != nil {
		return err
	}

	for _, childID := range childIDs {
		// update only if row use inheritance
		haveValue, err := s.haveOwnAttributeValue(ctx, attribute.ID, childID)
		if err != nil {
			return err
		}

		if !haveValue {
			value, err := s.calcInheritedValue(ctx, attribute.ID, childID)
			if err != nil {
				return err
			}

			changed, err := s.setActualValue(ctx, attribute, childID, value)
			if err != nil {
				return err
			}

			if changed {
				err = s.propagateInheritance(ctx, attribute, childID)
				if err != nil {
					return err
				}

				err = s.propagateEngine(ctx, attribute, childID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *Repository) haveOwnAttributeValue(ctx context.Context, attributeID, itemID int64) (bool, error) {
	var exists bool
	success, err := s.db.Select(goqu.V(true)).From(schema.AttrsUserValuesTable).Where(
		schema.AttrsUserValuesTableAttributeIDCol.Eq(attributeID),
		schema.AttrsUserValuesTableItemIDCol.Eq(itemID),
	).ScanValContext(ctx, &exists)

	return success && exists, err
}

func (s *Repository) propagateEngine(ctx context.Context, attribute schema.AttrsAttributeRow, itemID int64) error {
	isEngineAttributeID, err := s.isEngineAttributeID(ctx, attribute.ID)
	if err != nil {
		return err
	}

	if !isEngineAttributeID {
		return nil
	}

	if !attribute.TypeID.Valid {
		return nil
	}

	var vehicleIDs []int64

	err = s.db.Select(schema.ItemTableIDCol).From(schema.ItemTable).Where(
		schema.ItemTableEngineItemIDCol.Eq(itemID),
	).ScanValsContext(ctx, &vehicleIDs)
	if err != nil {
		return err
	}

	for _, vehicleID := range vehicleIDs {
		_, err = s.updateAttributeActualValue(ctx, attribute, vehicleID)
		if err != nil {
			return fmt.Errorf("%w: updateAttributeActualValue(%d, %d)", errAttributeNotFound, attribute.ID, itemID)
		}
	}

	return nil
}

func (s *Repository) refreshConflictFlag(ctx context.Context, attributeID, itemID int64) error {
	if itemID <= 0 {
		return errInvalidItemID
	}

	success, attribute, err := s.Attribute(ctx, attributeID)
	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("%w: `%d`", errAttributeNotFound, attributeID)
	}

	userValueRows, err := s.UserValueRows(ctx, query.AttrsUserValuesListOptions{
		AttributeID: attributeID,
		ItemID:      itemID,
	})
	if err != nil {
		return err
	}

	type userValueItem struct {
		Value Value
		Row   schema.AttrsUserValueRow
	}

	userValues := make(map[int64]userValueItem)
	hasConflict := false

	for _, userValueRow := range userValueRows {
		val, err := s.UserValue(
			ctx,
			attribute.ID,
			itemID,
			userValueRow.UserID,
		)
		if err != nil {
			return err
		}

		userValues[userValueRow.UserID] = userValueItem{
			Value: val,
			Row:   userValueRow,
		}

		if !hasConflict {
			for _, uv := range userValues {
				if !uv.Value.Equals(val) {
					hasConflict = true

					break
				}
			}
		}
	}

	s.db.Update(schema.AttrsValuesTable).Set(goqu.Record{
		schema.AttrsValuesTableConflictColName: hasConflict,
	}).Where(
		schema.AttrsValuesTableAttributeIDCol.Eq(attributeID),
		schema.AttrsValuesTableItemIDCol.Eq(itemID),
	)

	affectedUserIDs := make([]int64, 0)

	if hasConflict {
		actualValue, err := s.ActualValue(ctx, attributeID, itemID)
		if err != nil {
			return err
		}

		minDate := time.Now() // min date of actual value
		actualValueVoters := 0

		for _, userValue := range userValues {
			if userValue.Value.Equals(actualValue) {
				actualValueVoters++

				if minDate.After(userValue.Row.UpdateDate) {
					minDate = userValue.Row.UpdateDate
				}
			}
		}

		for userID, userValue := range userValues {
			matchActual := userValue.Value.Equals(actualValue)

			conflict := -1
			if matchActual {
				conflict = 1
			}

			weight := weightNone
			if actualValueVoters > 1 {
				weight = weightWrong

				if matchActual {
					weight = weightFirstActual

					if userValue.Row.UpdateDate == minDate {
						weight = weightSecondActual
					}
				}
			}

			res, err := s.db.Update(schema.AttrsUserValuesTable).Set(goqu.Record{
				schema.AttrsUserValuesTableConflictColName: conflict,
				schema.AttrsUserValuesTableWeightColName:   weight,
			}).Where(
				schema.AttrsUserValuesTableUserIDCol.Eq(userID),
				schema.AttrsUserValuesTableAttributeIDCol.Eq(attributeID),
				schema.AttrsUserValuesTableItemIDCol.Eq(itemID),
			).Executor().ExecContext(ctx)
			if err != nil {
				return err
			}

			affected, err := res.RowsAffected()
			if err != nil {
				return err
			}

			if affected > 0 {
				affectedUserIDs = append(affectedUserIDs, userID)
			}
		}
	} else {
		res, err := s.db.Update(schema.AttrsUserValuesTable).Set(goqu.Record{
			schema.AttrsUserValuesTableConflictColName: 0,
			schema.AttrsUserValuesTableWeightColName:   weightNone,
		}).Where(
			schema.AttrsUserValuesTableAttributeIDCol.Eq(attributeID),
			schema.AttrsUserValuesTableItemIDCol.Eq(itemID),
		).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}

		if affected > 0 {
			for _, userValue := range userValues {
				affectedUserIDs = append(affectedUserIDs, userValue.Row.UserID)
			}
		}
	}

	return s.RefreshUserConflictsStat(ctx, affectedUserIDs, false)
}

func (s *Repository) RefreshItemConflictFlags(ctx context.Context, itemID int64) error {
	var ids []int64

	err := s.db.Select(schema.AttrsUserValuesTableAttributeIDCol).Distinct().
		From(schema.AttrsUserValuesTable).
		Where(schema.AttrsUserValuesTableItemIDCol.Eq(itemID)).
		ScanValsContext(ctx, &ids)
	if err != nil {
		return err
	}

	for _, id := range ids {
		err = s.refreshConflictFlag(ctx, id, itemID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) RefreshUserConflictsStat(ctx context.Context, userIDs []int64, all bool) error {
	if len(userIDs) == 0 && !all {
		return nil
	}

	pSelect := s.db.Select(goqu.SUM(schema.AttrsUserValuesTableWeightCol)).
		From(schema.AttrsUserValuesTable).
		Where(
			schema.AttrsUserValuesTableUserIDCol.Eq(schema.UserTableIDCol),
			schema.AttrsUserValuesTableWeightCol.Gt(0),
		)

	nSelect := s.db.Select(goqu.Func("ABS", goqu.SUM(schema.AttrsUserValuesTableWeightCol))).
		From(schema.AttrsUserValuesTable).
		Where(
			schema.AttrsUserValuesTableUserIDCol.Eq(schema.UserTableIDCol),
			schema.AttrsUserValuesTableWeightCol.Lt(0),
		)

	expr := s.db.Update(schema.UserTable).Set(goqu.Record{
		schema.UserTableSpecsWeightColName: goqu.L(
			"1.5 * ((1 + IFNULL((?), 0)) / (1 + IFNULL((?), 0)))",
			pSelect,
			nSelect,
		),
	})

	if !all {
		expr = expr.Where(schema.UserTableIDCol.In(userIDs))
	}

	_, err := expr.Executor().ExecContext(ctx)

	return err
}

func (s *Repository) RefreshConflictFlags(ctx context.Context) error {
	var rows []schema.AttrsUserValueRow

	err := s.db.Select(schema.AttrsUserValuesTableAttributeIDCol, schema.AttrsUserValuesTableItemIDCol).
		Distinct().
		From(schema.AttrsUserValuesTable).
		Where(schema.AttrsUserValuesTableConflictCol.IsTrue()).ScanStructsContext(ctx, &rows)
	if err != nil {
		return err
	}

	for _, row := range rows {
		err = s.refreshConflictFlag(ctx, row.AttributeID, row.ItemID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) MoveUserValues(ctx context.Context, srcItemID, destItemID int64) error {
	rows, err := s.UserValueRows(ctx, query.AttrsUserValuesListOptions{
		ItemID: srcItemID,
	})
	if err != nil {
		return err
	}

	for _, row := range rows {
		_, err = s.db.Update(schema.AttrsUserValuesTable).Set(goqu.Record{
			schema.AttrsUserValuesTableItemIDColName: destItemID,
		}).Where(
			schema.AttrsUserValuesTableAttributeIDCol.Eq(row.AttributeID),
			schema.AttrsUserValuesTableItemIDCol.Eq(row.ItemID),
			schema.AttrsUserValuesTableUserIDCol.Eq(row.UserID),
		).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}

		_, err = s.db.Update(schema.AttrsUserValuesFloatTable).Set(goqu.Record{
			schema.AttrsUserValuesFloatTableItemIDColName: destItemID,
		}).Where(
			schema.AttrsUserValuesFloatTableAttributeIDCol.Eq(row.AttributeID),
			schema.AttrsUserValuesFloatTableItemIDCol.Eq(row.ItemID),
			schema.AttrsUserValuesFloatTableUserIDCol.Eq(row.UserID),
		).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}

		_, err = s.db.Update(schema.AttrsUserValuesIntTable).Set(goqu.Record{
			schema.AttrsUserValuesIntTableItemIDColName: destItemID,
		}).Where(
			schema.AttrsUserValuesIntTableAttributeIDCol.Eq(row.AttributeID),
			schema.AttrsUserValuesIntTableItemIDCol.Eq(row.ItemID),
			schema.AttrsUserValuesIntTableUserIDCol.Eq(row.UserID),
		).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}

		_, err = s.db.Update(schema.AttrsUserValuesStringTable).Set(goqu.Record{
			schema.AttrsUserValuesStringTableItemIDColName: destItemID,
		}).Where(
			schema.AttrsUserValuesStringTableAttributeIDCol.Eq(row.AttributeID),
			schema.AttrsUserValuesStringTableItemIDCol.Eq(row.ItemID),
			schema.AttrsUserValuesStringTableUserIDCol.Eq(row.UserID),
		).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}

		_, err = s.db.Update(schema.AttrsUserValuesListTable).Set(goqu.Record{
			schema.AttrsUserValuesListTableItemIDColName: destItemID,
		}).Where(
			schema.AttrsUserValuesListTableAttributeIDCol.Eq(row.AttributeID),
			schema.AttrsUserValuesListTableItemIDCol.Eq(row.ItemID),
			schema.AttrsUserValuesListTableUserIDCol.Eq(row.UserID),
		).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}

		err = s.updateActualValue(ctx, row.AttributeID, row.ItemID)
		if err != nil {
			return err
		}

		err = s.updateActualValue(ctx, row.AttributeID, destItemID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) UpdateAllActualValues(ctx context.Context) error {
	attributes, err := s.Attributes(ctx, 0, 0)
	if err != nil {
		return err
	}

	var itemIDs []int64

	err = s.db.Select(schema.AttrsUserValuesTableItemIDCol).Distinct().
		From(schema.AttrsUserValuesTable).
		ScanValsContext(ctx, &itemIDs)
	if err != nil {
		return err
	}

	for _, itemID := range itemIDs {
		for _, attribute := range attributes {
			if attribute.TypeID.Valid {
				_, err = s.updateAttributeActualValue(ctx, attribute, itemID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
