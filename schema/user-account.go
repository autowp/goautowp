package schema

import "github.com/doug-martin/goqu/v9"

const (
	UserAccountTableName = "user_account"
)

var (
	UserAccountTable             = goqu.T(UserAccountTableName)
	UserAccountTableUserIDCol    = UserAccountTable.Col("user_id")
	UserAccountTableServiceIDCol = UserAccountTable.Col("service_id")
)
