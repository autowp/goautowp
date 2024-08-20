package schema

import (
	"database/sql"
	"database/sql/driver"
	"errors"

	"github.com/doug-martin/goqu/v9"
)

var errUnsupportedAttributeTypeID = errors.New("unsupported type for AttributeTypeID")

type NullAttributeTypeID struct {
	AttributeTypeID AttrsAttributeTypeID
	Valid           bool // Valid is true if AttributeTypeID is not NULL
}

// Scan implements the Scanner interface.
func (n *NullAttributeTypeID) Scan(value any) error {
	if value == nil {
		n.AttributeTypeID, n.Valid = AttrsAttributeTypeIDUnknown, false

		return nil
	}

	n.Valid = true

	v, ok := value.(int64)
	if !ok {
		return errUnsupportedAttributeTypeID
	}

	n.AttributeTypeID = AttrsAttributeTypeID(v) //nolint: gosec

	return nil
}

// Value implements the driver Valuer interface.
func (n NullAttributeTypeID) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil //nolint: nilnil
	}

	return n.AttributeTypeID, nil
}

const (
	AttrsAttributesTableName = "attrs_attributes"
)

var (
	AttrsAttributesTable               = goqu.T(AttrsAttributesTableName)
	AttrsAttributesTableIDCol          = AttrsAttributesTable.Col("id")
	AttrsAttributesTableNameCol        = AttrsAttributesTable.Col("name")
	AttrsAttributesTableDescriptionCol = AttrsAttributesTable.Col("description")
	AttrsAttributesTableTypeIDCol      = AttrsAttributesTable.Col("type_id")
	AttrsAttributesTableUnitIDCol      = AttrsAttributesTable.Col("unit_id")
	AttrsAttributesTableMultipleCol    = AttrsAttributesTable.Col("multiple")
	AttrsAttributesTablePrecisionCol   = AttrsAttributesTable.Col("precision")
	AttrsAttributesTableParentIDCol    = AttrsAttributesTable.Col("parent_id")
	AttrsAttributesTablePositionCol    = AttrsAttributesTable.Col("position")
)

type AttrsAttributeRow struct {
	ID          int64               `db:"id"`
	Name        string              `db:"name"`
	ParentID    sql.NullInt64       `db:"parent_id"`
	Description sql.NullString      `db:"description"`
	TypeID      NullAttributeTypeID `db:"type_id"`
	UnitID      sql.NullInt64       `db:"unit_id"`
	Multiple    bool                `db:"multiple"`
	Precision   sql.NullInt32       `db:"precision"`
}
