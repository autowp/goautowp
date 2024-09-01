package schema

import "github.com/doug-martin/goqu/v9"

const (
	OfDayTableName           = "of_day"
	OfDayTableItemIDColName  = "item_id"
	OfDayTableUserIDColName  = "user_id"
	OfDayTableDayDateColName = "day_date"
)

var (
	OfDayTable           = goqu.T(OfDayTableName)
	OfDayTableDayDateCol = OfDayTable.Col(OfDayTableDayDateColName)
	OfDayTableItemIDCol  = OfDayTable.Col(OfDayTableItemIDColName)
)
