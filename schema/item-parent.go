package schema

import "github.com/doug-martin/goqu/v9"

type ItemParentType int8

const (
	ItemParentTypeDefault ItemParentType = 0
	ItemParentTypeTuning  ItemParentType = 1
	ItemParentTypeSport   ItemParentType = 2
	ItemParentTypeDesign  ItemParentType = 3

	ItemParentTableName             = "item_parent"
	ItemParentTableParentIDColName  = "parent_id"
	ItemParentTableItemIDColName    = "item_id"
	ItemParentTableTypeColName      = "type"
	ItemParentTableCatnameColName   = "catname"
	ItemParentTableTimestampColName = "timestamp"
)

var (
	ItemParentTable            = goqu.T(ItemParentTableName)
	ItemParentTableParentIDCol = ItemParentTable.Col(ItemParentTableParentIDColName)
	ItemParentTableItemIDCol   = ItemParentTable.Col(ItemParentTableItemIDColName)
	ItemParentTableTypeCol     = ItemParentTable.Col(ItemParentTableTypeColName)
	ItemParentTableCatnameCol  = ItemParentTable.Col(ItemParentTableCatnameColName)
)
