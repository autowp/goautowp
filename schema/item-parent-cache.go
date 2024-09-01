package schema

import "github.com/doug-martin/goqu/v9"

const (
	ItemParentCacheTableName            = "item_parent_cache"
	ItemParentCacheTableItemIDColName   = "item_id"
	ItemParentCacheTableParentIDColName = "parent_id"
	ItemParentCacheTableDiffColName     = "diff"
	ItemParentCacheTableTuningColName   = "tuning"
	ItemParentCacheTableSportColName    = "sport"
	ItemParentCacheTableDesignColName   = "design"
)

var (
	ItemParentCacheTable            = goqu.T(ItemParentCacheTableName)
	ItemParentCacheTableItemIDCol   = ItemParentCacheTable.Col(ItemParentCacheTableItemIDColName)
	ItemParentCacheTableParentIDCol = ItemParentCacheTable.Col(ItemParentCacheTableParentIDColName)
)
