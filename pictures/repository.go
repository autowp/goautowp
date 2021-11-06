package pictures

type Status string

const (
	StatusAccepted Status = "accepted"
	StatusRemoving Status = "removing"
	StatusRemoved  Status = "removed"
	StatusInbox    Status = "inbox"
)

type ItemPictureType int

const (
	ItemPictureContent    ItemPictureType = 1
	ItemPictureAuthor     ItemPictureType = 2
	ItemPictureCopyrights ItemPictureType = 3
)
