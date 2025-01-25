package schema

import "github.com/doug-martin/goqu/v9"

const (
	LinksTableName          = "links"
	LinksTableIDColName     = "id"
	LinksTableNameColName   = "name"
	LinksTableURLColName    = "url"
	LinksTableTypeColName   = "type"
	LinksTableItemIDColName = "item_id"
)

var (
	LinksTable          = goqu.T(LinksTableName)
	LinksTableIDCol     = LinksTable.Col(LinksTableIDColName)
	LinksTableNameCol   = LinksTable.Col(LinksTableNameColName)
	LinksTableURLCol    = LinksTable.Col(LinksTableURLColName)
	LinksTableTypeCol   = LinksTable.Col(LinksTableTypeColName)
	LinksTableItemIDCol = LinksTable.Col(LinksTableItemIDColName)
)

type LinkRow struct {
	ID     int64  `db:"id"`
	Name   string `db:"name"`
	URL    string `db:"url"`
	Type   string `db:"type"`
	ItemID int64  `db:"item_id"`
}
