package schema

import "github.com/doug-martin/goqu/v9"

const (
	TelegramBrandTableName = "telegram_brand"
)

var (
	TelegramBrandTable          = goqu.T(TelegramBrandTableName)
	TelegramBrandTableChatIDCol = TelegramBrandTable.Col("chat_id")
	TelegramBrandTableItemIDCol = TelegramBrandTable.Col("item_id")
	TelegramBrandTableNewCol    = TelegramBrandTable.Col("new")
)
