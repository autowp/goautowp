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

const (
	ItemParentTypeDefault = 0
	ItemParentTypeTuning  = 1
	ItemParentTypeSport   = 2
	ItemParentTypeDesign  = 3
)

type AttrsTypesID int32

const (
	AttrsTypesIDUnknown AttrsTypesID = 0
	AttrsTypesIDString  AttrsTypesID = 1
	AttrsTypesIDInteger AttrsTypesID = 2
	AttrsTypesIDFloat   AttrsTypesID = 3
	AttrsTypesIDText    AttrsTypesID = 4
	AttrsTypesIDBoolean AttrsTypesID = 5
	AttrsTypesIDList    AttrsTypesID = 6
	AttrsTypesIDTree    AttrsTypesID = 7
)

type CommentMessageModeratorAttention int32

const (
	CommentMessageModeratorAttentionNone      CommentMessageModeratorAttention = 0
	CommentMessageModeratorAttentionRequired  CommentMessageModeratorAttention = 1
	CommentMessageModeratorAttentionCompleted CommentMessageModeratorAttention = 2
)

type CommentMessageType int32

const (
	CommentMessageTypeIDPictures CommentMessageType = 1
	CommentMessageTypeIDItems    CommentMessageType = 2
	CommentMessageTypeIDVotings  CommentMessageType = 3
	CommentMessageTypeIDArticles CommentMessageType = 4
	CommentMessageTypeIDForums   CommentMessageType = 5
)
