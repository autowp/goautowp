package attrs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"

	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

type AttributeTypeID int32

const (
	TypeUnknown AttributeTypeID = 0
	TypeString  AttributeTypeID = 1
	TypeInteger AttributeTypeID = 2
	TypeFloat   AttributeTypeID = 3
	TypeText    AttributeTypeID = 4
	TypeBoolean AttributeTypeID = 5
	TypeList    AttributeTypeID = 6
	TypeTree    AttributeTypeID = 7
)

type NullAttributeTypeID struct {
	AttributeTypeID AttributeTypeID
	Valid           bool // Valid is true if AttributeTypeID is not NULL
}

// Scan implements the Scanner interface.
func (n *NullAttributeTypeID) Scan(value any) error {
	if value == nil {
		n.AttributeTypeID, n.Valid = TypeUnknown, false

		return nil
	}

	n.Valid = true

	v, ok := value.(int64)
	if !ok {
		return errors.New("unsupported type for AttributeTypeID")
	}

	n.AttributeTypeID = AttributeTypeID(v)

	return nil
}

// Value implements the driver Valuer interface.
func (n NullAttributeTypeID) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	return n.AttributeTypeID, nil
}

type Attribute struct {
	ID          int64               `db:"id"`
	Name        string              `db:"name"`
	ParentID    sql.NullInt64       `db:"parent_id"`
	Description sql.NullString      `db:"description"`
	TypeID      NullAttributeTypeID `db:"type_id"`
	UnitID      sql.NullInt64       `db:"unit_id"`
	Multiple    bool                `db:"multiple"`
	Precision   sql.NullInt32       `db:"precision"`
}

type AttributeType struct {
	ID   AttributeTypeID `db:"id"`
	Name string          `db:"name"`
}

type ListOption struct {
	ID          int64         `db:"id"`
	Name        string        `db:"name"`
	AttributeID int64         `db:"attribute_id"`
	ParentID    sql.NullInt64 `db:"parent_id"`
}

type Unit struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
	Abbr string `db:"abbr"`
}

type Zone struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

type ZoneAttribute struct {
	ZoneID      int64 `db:"zone_id"`
	AttributeID int64 `db:"attribute_id"`
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

func (s *Repository) Attribute(ctx context.Context, id int64) (bool, Attribute, error) {
	sqSelect := s.db.Select(
		schema.AttrsAttributesTableIDCol, schema.AttrsAttributesTableNameCol, schema.AttrsAttributesTableDescriptionCol,
		schema.AttrsAttributesTableTypeIDCol, schema.AttrsAttributesTableUnitIDCol, schema.AttrsAttributesTableMultipleCol,
		schema.AttrsAttributesTablePrecisionCol, schema.AttrsAttributesTableParentIDCol,
	).
		From(schema.AttrsAttributesTable).
		Order(schema.AttrsAttributesTablePositionCol.Asc()).
		Where(schema.AttrsAttributesTableIDCol.Eq(id))

	r := Attribute{}
	success, err := sqSelect.ScanStructContext(ctx, &r)

	return success, r, err
}

func (s *Repository) Attributes(ctx context.Context, zoneID int64, parentID int64) ([]Attribute, error) {
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

	r := make([]Attribute, 0)
	err := sqSelect.ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) AttributeTypes(ctx context.Context) ([]AttributeType, error) {
	r := make([]AttributeType, 0)
	err := s.db.Select(schema.AttrsTypesTableIDCol, schema.AttrsTypesTableNameCol).
		From(schema.AttrsTypesTable).
		ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) ListOptions(ctx context.Context, attributeID int64) ([]ListOption, error) {
	sqSelect := s.db.Select(schema.AttrsListOptionsTableIDCol, schema.AttrsListOptionsTableNameCol,
		schema.AttrsListOptionsTableAttributeIDCol, schema.AttrsListOptionsTableParentIDCol).
		From(schema.AttrsListOptionsTable).
		Order(schema.AttrsListOptionsTablePositionCol.Asc())

	if attributeID > 0 {
		sqSelect = sqSelect.Where(schema.AttrsListOptionsTableAttributeIDCol.Eq(attributeID))
	}

	r := make([]ListOption, 0)
	err := sqSelect.ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) Units(ctx context.Context) ([]Unit, error) {
	r := make([]Unit, 0)
	err := s.db.Select(schema.AttrsUnitsTableIDCol, schema.AttrsUnitsTableNameCol, schema.AttrsUnitsTableAbbrCol).
		From(schema.AttrsUnitsTable).
		ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) ZoneAttributes(ctx context.Context, zoneID int64) ([]ZoneAttribute, error) {
	r := make([]ZoneAttribute, 0)
	err := s.db.Select(schema.AttrsZoneAttributesTableZoneIDCol, schema.AttrsZoneAttributesTableAttributeIDCol).
		From(schema.AttrsZoneAttributesTable).
		Where(schema.AttrsZoneAttributesTableZoneIDCol.Eq(zoneID)).
		ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) Zones(ctx context.Context) ([]Zone, error) {
	r := make([]Zone, 0)
	err := s.db.Select(schema.AttrsZonesTableIDCol, schema.AttrsZonesTableNameCol).
		From(schema.AttrsZonesTable).
		ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) TotalValues(ctx context.Context) (int32, error) {
	var result int32

	sqSelect := s.db.Select(goqu.COUNT(goqu.Star())).From(goqu.T(schema.TableAttrsValues))

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
