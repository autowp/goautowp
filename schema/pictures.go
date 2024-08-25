package schema

import (
	"database/sql"

	"github.com/doug-martin/goqu/v9"
)

type PictureStatus string

const (
	PictureStatusAccepted PictureStatus = "accepted"
	PictureStatusRemoving PictureStatus = "removing"
	PictureStatusRemoved  PictureStatus = "removed"
	PictureStatusInbox    PictureStatus = "inbox"

	PictureTableName                      = "pictures"
	PictureTableIDColName                 = "id"
	PictureTableImageIDColName            = "image_id"
	PictureTableIdentityColName           = "identity"
	PictureTableIPColName                 = "ip"
	PictureTableOwnerIDColName            = "owner_id"
	PictureTableStatusColName             = "status"
	PictureTableChangeStatusUserIDColName = "change_status_user_id"
	PictureTableWidthColName              = "width"
	PictureTableHeightColName             = "height"
	PictureTableContentCountColName       = "content_count"
	PictureTableReplacePictureIDColName   = "replace_picture_id"
	PictureTablePointColName              = "point"
)

var (
	PictureTable                      = goqu.T(PictureTableName)
	PictureTableIDCol                 = PictureTable.Col(PictureTableIDColName)
	PictureTableIdentityCol           = PictureTable.Col(PictureTableIdentityColName)
	PictureTableOwnerIDCol            = PictureTable.Col(PictureTableOwnerIDColName)
	PictureTableStatusCol             = PictureTable.Col(PictureTableStatusColName)
	PictureTableImageIDCol            = PictureTable.Col(PictureTableImageIDColName)
	PictureTableChangeStatusUserIDCol = PictureTable.Col(PictureTableChangeStatusUserIDColName)
	PictureTableWidthCol              = PictureTable.Col(PictureTableWidthColName)
	PictureTableHeightCol             = PictureTable.Col(PictureTableHeightColName)
	PictureTableReplacePictureIDCol   = PictureTable.Col(PictureTableReplacePictureIDColName)
	PictureTablePointCol              = PictureTable.Col(PictureTablePointColName)
)

type PictureRow struct {
	OwnerID            sql.NullInt64 `db:"owner_id"`
	ChangeStatusUserID sql.NullInt64 `db:"change_status_user_id"`
	Identity           string        `db:"identity"`
	Status             PictureStatus `db:"status"`
	ImageID            int64         `db:"image_id"`
	Width              uint16        `db:"width"`
	Height             uint16        `db:"height"`
	Point              *NullPoint    `db:"point"`
}
