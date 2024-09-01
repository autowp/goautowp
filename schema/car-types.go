package schema

import "github.com/doug-martin/goqu/v9"

const (
	CarTypesTableName = "car_types"
)

var (
	CarTypesTable            = goqu.T(CarTypesTableName)
	CarTypesTableIDCol       = CarTypesTable.Col("id")
	CarTypesTableNameCol     = CarTypesTable.Col("name")
	CarTypesTableCatnameCol  = CarTypesTable.Col("catname")
	CarTypesTablePositionCol = CarTypesTable.Col("position")
	CarTypesTableParentIDCol = CarTypesTable.Col("parent_id")
)
