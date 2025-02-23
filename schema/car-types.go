package schema

import "github.com/doug-martin/goqu/v9"

const (
	CarTypesTableName = "car_types"

	CarTypesTableIDColName       = "id"
	CarTypesTableCatnameColName  = "catname"
	CarTypesTableParentIDColName = "parent_id"
	CarTypesTableNameRpColName   = "name_rp"
	CarTypesTablePositionColName = "position"
)

var (
	CarTypesTable            = goqu.T(CarTypesTableName)
	CarTypesTableIDCol       = CarTypesTable.Col(CarTypesTableIDColName)
	CarTypesTableNameCol     = CarTypesTable.Col("name")
	CarTypesTableCatnameCol  = CarTypesTable.Col(CarTypesTableCatnameColName)
	CarTypesTablePositionCol = CarTypesTable.Col(CarTypesTablePositionColName)
	CarTypesTableParentIDCol = CarTypesTable.Col(CarTypesTableParentIDColName)
)

type CarTypeRow struct {
	ID      int64  `db:"id"`
	Catname string `db:"catname"`
	NameRp  string `db:"name_rp"`
}
