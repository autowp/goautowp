package schema

import "github.com/doug-martin/goqu/v9"

const (
	CarTypesParentsTableName            = "car_types_parents"
	CarTypesParentsTableIDColName       = "id"
	CarTypesParentsTableParentIDColName = "parent_id"
)

var (
	CarTypesParentsTable            = goqu.T(CarTypesParentsTableName)
	CarTypesParentsTableIDCol       = CarTypesParentsTable.Col(CarTypesParentsTableIDColName)
	CarTypesParentsTableParentIDCol = CarTypesParentsTable.Col(CarTypesParentsTableParentIDColName)
)
