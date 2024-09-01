package schema

import "github.com/doug-martin/goqu/v9"

const (
	AttrsZoneAttributesTableName = "attrs_zone_attributes"
)

var (
	AttrsZoneAttributesTable               = goqu.T(AttrsZoneAttributesTableName)
	AttrsZoneAttributesTableZoneIDCol      = AttrsZoneAttributesTable.Col("zone_id")
	AttrsZoneAttributesTableAttributeIDCol = AttrsZoneAttributesTable.Col("attribute_id")
	AttrsZoneAttributesTablePositionCol    = AttrsZoneAttributesTable.Col("position")
)

type AttrsZoneAttributeRow struct {
	ZoneID      int64 `db:"zone_id"`
	AttributeID int64 `db:"attribute_id"`
}
