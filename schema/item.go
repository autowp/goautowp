package schema

import (
	"database/sql"

	"github.com/doug-martin/goqu/v9"
)

type ItemTableItemTypeID int

const (
	ItemTableItemTypeIDVehicle   ItemTableItemTypeID = 1
	ItemTableItemTypeIDEngine    ItemTableItemTypeID = 2
	ItemTableItemTypeIDCategory  ItemTableItemTypeID = 3
	ItemTableItemTypeIDTwins     ItemTableItemTypeID = 4
	ItemTableItemTypeIDBrand     ItemTableItemTypeID = 5
	ItemTableItemTypeIDFactory   ItemTableItemTypeID = 6
	ItemTableItemTypeIDMuseum    ItemTableItemTypeID = 7
	ItemTableItemTypeIDPerson    ItemTableItemTypeID = 8
	ItemTableItemTypeIDCopyright ItemTableItemTypeID = 9

	ItemTableName                          = "item"
	ItemTableNameColName                   = "name"
	ItemTableCatnameColName                = "catname"
	ItemTableEngineItemIDColName           = "engine_item_id"
	ItemTableItemTypeIDColName             = "item_type_id"
	ItemTableIsConceptColName              = "is_concept"
	ItemTableIsConceptInheritColName       = "is_concept_inherit"
	ItemTableSpecIDColName                 = "spec_id"
	ItemTableIDColName                     = "id"
	ItemTableFullNameColName               = "full_name"
	ItemTableLogoIDColName                 = "logo_id"
	ItemTableBeginYearColName              = "begin_year"
	ItemTableEndYearColName                = "end_year"
	ItemTableBeginMonthColName             = "begin_month"
	ItemTableEndMonthColName               = "end_month"
	ItemTableBeginModelYearColName         = "begin_model_year"
	ItemTableEndModelYearColName           = "end_model_year"
	ItemTableBeginModelYearFractionColName = "begin_model_year_fraction"
	ItemTableEndModelYearFractionColName   = "end_model_year_fraction"
	ItemTableTodayColName                  = "today"
	ItemTableBodyColName                   = "body"
	ItemTableIsGroupColName                = "is_group"
	ItemTableProducedExactlyColName        = "produced_exactly"
	ItemTableAddDatetimeColName            = "add_datetime"
)

var (
	ItemTable                  = goqu.T(ItemTableName)
	ItemTableIDCol             = ItemTable.Col(ItemTableIDColName)
	ItemTableBodyCol           = ItemTable.Col(ItemTableBodyColName)
	ItemTableSpecIDCol         = ItemTable.Col(ItemTableSpecIDColName)
	ItemTableCatnameCol        = ItemTable.Col(ItemTableCatnameColName)
	ItemTableNameCol           = ItemTable.Col(ItemTableNameColName)
	ItemTableBeginYearCol      = ItemTable.Col(ItemTableBeginYearColName)
	ItemTableEndYearCol        = ItemTable.Col(ItemTableEndYearColName)
	ItemTableBeginModelYearCol = ItemTable.Col(ItemTableBeginModelYearColName)
	ItemTableEndModelYearCol   = ItemTable.Col(ItemTableEndModelYearColName)
	ItemTableIsGroupCol        = ItemTable.Col(ItemTableIsGroupColName)
	ItemTableItemTypeIDCol     = ItemTable.Col(ItemTableItemTypeIDColName)
	ItemTableTodayCol          = ItemTable.Col(ItemTableTodayColName)
)

type ItemRow struct {
	ID             int64               `db:"id"`
	Name           string              `db:"name"`
	ItemType       ItemTableItemTypeID `db:"item_type_id"`
	Body           string              `db:"body"`
	BeginYear      sql.NullInt32       `db:"begin_year"`
	EndYear        sql.NullInt32       `db:"end_year"`
	BeginModelYear sql.NullInt32       `db:"begin_model_year"`
	EndModelYear   sql.NullInt32       `db:"end_model_year"`
	SpecID         sql.NullInt32       `db:"spec_id"`
}