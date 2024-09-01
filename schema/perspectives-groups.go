package schema

import "github.com/doug-martin/goqu/v9"

const (
	PerspectivesGroupsTableName = "perspectives_groups"
)

var (
	PerspectivesGroupsTable            = goqu.T(PerspectivesGroupsTableName)
	PerspectivesGroupsTableIDCol       = PerspectivesGroupsTable.Col("id")
	PerspectivesGroupsTableNameCol     = PerspectivesGroupsTable.Col("name")
	PerspectivesGroupsTablePageIDCol   = PerspectivesGroupsTable.Col("page_id")
	PerspectivesGroupsTablePositionCol = PerspectivesGroupsTable.Col("position")
)
