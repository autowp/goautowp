package schema

import (
	"database/sql"

	"github.com/doug-martin/goqu/v9"
)

const (
	AttrsListOptionsTableName = "attrs_list_options"
)

var (
	AttrsListOptionsTable               = goqu.T(AttrsListOptionsTableName)
	AttrsListOptionsTableIDCol          = AttrsListOptionsTable.Col("id")
	AttrsListOptionsTableNameCol        = AttrsListOptionsTable.Col("name")
	AttrsListOptionsTableAttributeIDCol = AttrsListOptionsTable.Col("attribute_id")
	AttrsListOptionsTableParentIDCol    = AttrsListOptionsTable.Col("parent_id")
	AttrsListOptionsTablePositionCol    = AttrsListOptionsTable.Col("position")
)

type AttrsListOptionRow struct {
	ID          int64         `db:"id"`
	Name        string        `db:"name"`
	AttributeID int64         `db:"attribute_id"`
	ParentID    sql.NullInt64 `db:"parent_id"`
}
