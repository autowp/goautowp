package schema

import "github.com/doug-martin/goqu/v9"

const (
	AttrsUserValuesTableName          = "attrs_user_values"
	AttrsUserValuesTableUserIDColName = "user_id"
)

var (
	AttrsUserValuesTable          = goqu.T(AttrsUserValuesTableName)
	AttrsUserValuesTableUserIDCol = AttrsUserValuesTable.Col(AttrsUserValuesTableUserIDColName)
	AttrsUserValuesTableItemIDCol = AttrsUserValuesTable.Col("item_id")
	AttrsUserValuesTableWeightCol = AttrsUserValuesTable.Col("weight")
)
