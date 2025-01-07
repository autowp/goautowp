package schema

import "github.com/doug-martin/goqu/v9"

const (
	PerspectivesTableName = "perspectives"

	PerspectiveIDUnderTheHood = 17
)

var (
	PerspectivesTable            = goqu.T(PerspectivesTableName)
	PerspectivesTableIDCol       = PerspectivesTable.Col("id")
	PerspectivesTablePositionCol = PerspectivesTable.Col("position")
	PerspectivesTableNameCol     = PerspectivesTable.Col("name")
)

type PerspectiveRow struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}
