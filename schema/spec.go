package schema

import "github.com/doug-martin/goqu/v9"

const (
	SpecTableName             = "spec"
	SpecTableIDColName        = "id"
	SpecTableNameColName      = "name"
	SpecTableShortNameColName = "short_name"
	SpecTableParentIDColName  = "parent_id"
)

var (
	SpecTable             = goqu.T(SpecTableName)
	SpecTableIDCol        = SpecTable.Col(SpecTableIDColName)
	SpecTableNameCol      = SpecTable.Col(SpecTableNameColName)
	SpecTableShortNameCol = SpecTable.Col(SpecTableShortNameColName)
	SpecTableParentIDCol  = SpecTable.Col(SpecTableParentIDColName)
)
