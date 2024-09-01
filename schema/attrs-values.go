package schema

import "github.com/doug-martin/goqu/v9"

const (
	AttrsValuesTableName = "attrs_values"
)

var (
	AttrsValuesTable = goqu.T(AttrsValuesTableName)
)
