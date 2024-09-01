package schema

import "github.com/doug-martin/goqu/v9"

const (
	PerspectivesGroupsPerspectivesTableName = "perspectives_groups_perspectives"
)

var (
	PerspectivesGroupsPerspectivesTable                 = goqu.T(PerspectivesGroupsPerspectivesTableName)
	PerspectivesGroupsPerspectivesTablePerspectiveIDCol = PerspectivesGroupsPerspectivesTable.Col("perspective_id")
	PerspectivesGroupsPerspectivesTableGroupIDCol       = PerspectivesGroupsPerspectivesTable.Col("group_id")
	PerspectivesGroupsPerspectivesTablePositionCol      = PerspectivesGroupsPerspectivesTable.Col("position")
)
