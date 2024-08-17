package schema

import "database/sql"

type PictureRow struct {
	OwnerID            sql.NullInt64 `db:"owner_id"`
	ChangeStatusUserID sql.NullInt64 `db:"change_status_user_id"`
	Identity           string        `db:"identity"`
	Status             PictureStatus `db:"status"`
	ImageID            int64         `db:"image_id"`
	Width              uint16        `db:"width"`
	Height             uint16        `db:"height"`
}

type PictureItemRow struct {
	PictureID  int64           `db:"picture_id"`
	ItemID     int64           `db:"item_id"`
	Type       PictureItemType `db:"type"`
	CropLeft   sql.NullInt32   `db:"crop_left"`
	CropTop    sql.NullInt32   `db:"crop_top"`
	CropWidth  sql.NullInt32   `db:"crop_width"`
	CropHeight sql.NullInt32   `db:"crop_height"`
}
