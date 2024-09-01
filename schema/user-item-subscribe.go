package schema

import "github.com/doug-martin/goqu/v9"

var (
	UserItemSubscribeTable          = goqu.T("user_item_subscribe")
	UserItemSubscribeTableUserIDCol = UserItemSubscribeTable.Col("user_id")
)
