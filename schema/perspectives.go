package schema

import "github.com/doug-martin/goqu/v9"

const (
	PerspectivesTableName = "perspectives"
)

var (
	PerspectivesTable            = goqu.T(PerspectivesTableName)
	PerspectivesTableIDCol       = PerspectivesTable.Col("id")
	PerspectivesTablePositionCol = PerspectivesTable.Col("position")
	PerspectivesTableNameCol     = PerspectivesTable.Col("name")
)
