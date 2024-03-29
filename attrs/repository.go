package attrs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"

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
		"attrs_attributes.id", "attrs_attributes.name", "attrs_attributes.description",
		"attrs_attributes.type_id", "attrs_attributes.unit_id", "attrs_attributes.multiple",
		"attrs_attributes.precision", "attrs_attributes.parent_id",
	).
		From("attrs_attributes").
		Order(goqu.I("position").Asc()).Where(goqu.I("attrs_attributes.id").Eq(id))

	r := Attribute{}
	success, err := sqSelect.ScanStructContext(ctx, &r)

	return success, r, err
}

func (s *Repository) Attributes(ctx context.Context, zoneID int64, parentID int64) ([]Attribute, error) {
	sqSelect := s.db.Select(
		"attrs_attributes.id", "attrs_attributes.name", "attrs_attributes.description",
		"attrs_attributes.type_id", "attrs_attributes.unit_id", "attrs_attributes.multiple",
		"attrs_attributes.precision", "attrs_attributes.parent_id",
	).
		From("attrs_attributes")

	if zoneID > 0 {
		sqSelect = sqSelect.Join(
			goqu.T("attrs_zone_attributes"),
			goqu.On(goqu.I("attrs_attributes.id").Eq(goqu.I("attrs_zone_attributes.attribute_id"))),
		).
			Where(goqu.I("attrs_zone_attributes.zone_id").Eq(zoneID)).
			Order(goqu.I("attrs_zone_attributes.position").Asc())
	} else {
		sqSelect = sqSelect.Order(goqu.I("attrs_attributes.position").Asc())
	}

	if parentID > 0 {
		sqSelect = sqSelect.Where(goqu.I("attrs_attributes.parent_id").Eq(parentID))
	}

	r := make([]Attribute, 0)
	err := sqSelect.ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) AttributeTypes(ctx context.Context) ([]AttributeType, error) {
	r := make([]AttributeType, 0)
	err := s.db.Select("id", "name").From("attrs_types").ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) ListOptions(ctx context.Context, attributeID int64) ([]ListOption, error) {
	sqSelect := s.db.Select("id", "name", "attribute_id", "parent_id").
		From("attrs_list_options").Order(goqu.I("position").Asc())

	if attributeID > 0 {
		sqSelect = sqSelect.Where(goqu.I("attribute_id").Eq(attributeID))
	}

	r := make([]ListOption, 0)
	err := sqSelect.ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) Units(ctx context.Context) ([]Unit, error) {
	r := make([]Unit, 0)
	err := s.db.Select("id", "name", "abbr").From("attrs_units").ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) ZoneAttributes(ctx context.Context, zoneID int64) ([]ZoneAttribute, error) {
	r := make([]ZoneAttribute, 0)
	err := s.db.Select("zone_id", "attribute_id").
		From("attrs_zone_attributes").
		Where(goqu.I("zone_id").Eq(zoneID)).
		ScanStructsContext(ctx, &r)

	return r, err
}

func (s *Repository) Zones(ctx context.Context) ([]Zone, error) {
	r := make([]Zone, 0)
	err := s.db.Select("id", "name").From("attrs_zones").ScanStructsContext(ctx, &r)

	return r, err
}
