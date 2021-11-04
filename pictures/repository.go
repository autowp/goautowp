package pictures

type Status string

const (
	STATUS_ACCEPTED Status = "accepted"
	STATUS_REMOVING Status = "removing"
	STATUS_REMOVED  Status = "removed"
	STATUS_INBOX    Status = "inbox"
)

type PictureItemType int

const (
	PICTURE_CONTENT    PictureItemType = 1
	PICTURE_AUTHOR     PictureItemType = 2
	PICTURE_COPYRIGHTS PictureItemType = 3
)
