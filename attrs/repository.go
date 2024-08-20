package attrs

import (
	"context"
	"database/sql"

	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

type TopUserBrand struct {
	ID      int64  `db:"id"`
	Name    string `db:"name"`
	Catname string `db:"catname"`
	Volume  int64  `db:"volume"`
}

// Repository Main Object.
type Repository struct {
	db *goqu.Database
}

// NewRepository constructor.
func NewRepository(
	db *goqu.Database,
) *Repository {
	return &Repository{
		db: db,
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
	var result int32

	sqSelect := s.db.Select(goqu.COUNT(goqu.Star())).From(schema.AttrsValuesTable)

	success, err := sqSelect.ScanValContext(ctx, &result)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	return result, nil
}

func (s *Repository) TotalZoneAttrs(ctx context.Context, zoneID int64) (int32, error) {
	var result int32

	sqSelect := s.db.Select(goqu.COUNT(goqu.Star())).From(schema.AttrsAttributesTable).
		Join(
			schema.AttrsZoneAttributesTable,
			goqu.On(schema.AttrsAttributesTableIDCol.Eq(schema.AttrsZoneAttributesTableAttributeIDCol)),
		).
		Where(schema.AttrsZoneAttributesTableZoneIDCol.Eq(zoneID))

	success, err := sqSelect.ScanValContext(ctx, &result)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	return result, nil
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
