package schema

import "github.com/doug-martin/goqu/v9"

const (
	DfDistanceTableName                = "df_distance"
	DfDistanceTableDistanceColName     = "distance"
	DfDistanceTableSrcPictureIDColName = "src_picture_id"
	DfDistanceTableDstPictureIDColName = "dst_picture_id"
	DfDistanceTableHideColName         = "hide"
)

var (
	DfDistanceTable                = goqu.T(DfDistanceTableName)
	DfDistanceTableDistanceCol     = DfDistanceTable.Col(DfDistanceTableDistanceColName)
	DfDistanceTableSrcPictureIDCol = DfDistanceTable.Col(DfDistanceTableSrcPictureIDColName)
	DfDistanceTableDstPictureIDCol = DfDistanceTable.Col(DfDistanceTableDstPictureIDColName)
)
