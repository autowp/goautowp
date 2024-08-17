package schema

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
)

type PictureStatus string

const (
	PictureStatusAccepted PictureStatus = "accepted"
	PictureStatusRemoving PictureStatus = "removing"
	PictureStatusRemoved  PictureStatus = "removed"
	PictureStatusInbox    PictureStatus = "inbox"
)

type PictureItemType int

const (
	PictureItemContent    PictureItemType = 1
	PictureItemAuthor     PictureItemType = 2
	PictureItemCopyrights PictureItemType = 3
)
