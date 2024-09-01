package schema

import "github.com/doug-martin/goqu/v9"

const (
	ItemLanguageTableName              = "item_language"
	ItemLanguageTableItemIDColName     = "item_id"
	ItemLanguageTableLanguageColName   = "language"
	ItemLanguageTableNameColName       = "name"
	ItemLanguageTableTextIDColName     = "text_id"
	ItemLanguageTableFullTextIDColName = "full_text_id"
)

var (
	ItemLanguageTable              = goqu.T(ItemLanguageTableName)
	ItemLanguageTableItemIDCol     = ItemLanguageTable.Col(ItemLanguageTableItemIDColName)
	ItemLanguageTableLanguageCol   = ItemLanguageTable.Col(ItemLanguageTableLanguageColName)
	ItemLanguageTableNameCol       = ItemLanguageTable.Col(ItemLanguageTableNameColName)
	ItemLanguageTableTextIDCol     = ItemLanguageTable.Col(ItemLanguageTableTextIDColName)
	ItemLanguageTableFullTextIDCol = ItemLanguageTable.Col(ItemLanguageTableFullTextIDColName)
)
