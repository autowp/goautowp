package goautowp

import (
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
)

func extractPictureModerVoteTemplate(tpl *schema.PictureModerVoteTemplateRow) *ModerVoteTemplate {
	return &ModerVoteTemplate{
		Id:      tpl.ID,
		UserId:  tpl.UserID,
		Message: tpl.Message,
		Vote:    int32(tpl.Vote),
	}
}

func reverseConvertPictureItemType(pictureItemType schema.PictureItemType) PictureItemType {
	switch pictureItemType {
	case 0:
		return PictureItemType_PICTURE_ITEM_UNKNOWN
	case schema.PictureItemContent:
		return PictureItemType_PICTURE_ITEM_CONTENT
	case schema.PictureItemAuthor:
		return PictureItemType_PICTURE_ITEM_AUTHOR
	case schema.PictureItemCopyrights:
		return PictureItemType_PICTURE_ITEM_COPYRIGHTS
	}

	return PictureItemType_PICTURE_ITEM_UNKNOWN
}

func convertPictureItemType(pictureItemType PictureItemType) schema.PictureItemType {
	switch pictureItemType {
	case PictureItemType_PICTURE_ITEM_UNKNOWN:
		return 0
	case PictureItemType_PICTURE_ITEM_CONTENT:
		return schema.PictureItemContent
	case PictureItemType_PICTURE_ITEM_AUTHOR:
		return schema.PictureItemAuthor
	case PictureItemType_PICTURE_ITEM_COPYRIGHTS:
		return schema.PictureItemCopyrights
	}

	return 0
}

func convertPictureStatus(status PictureStatus) schema.PictureStatus {
	switch status {
	case PictureStatus_PICTURE_STATUS_UNKNOWN:
		return ""
	case PictureStatus_PICTURE_STATUS_ACCEPTED:
		return schema.PictureStatusAccepted
	case PictureStatus_PICTURE_STATUS_REMOVING:
		return schema.PictureStatusRemoving
	case PictureStatus_PICTURE_STATUS_REMOVED:
		return schema.PictureStatusRemoved
	case PictureStatus_PICTURE_STATUS_INBOX:
		return schema.PictureStatusInbox
	}

	return ""
}

func mapPictureItemListOptions(in *PictureItemOptions, options *query.PictureItemListOptions) error {
	options.TypeID = convertPictureItemType(in.GetTypeId())
	options.PerspectiveID = in.GetPerspectiveId()
	options.ItemID = in.GetItemId()

	if in.GetPictures() != nil {
		options.Pictures = &query.PictureListOptions{}

		err := mapPictureListOptions(in.GetPictures(), options.Pictures)
		if err != nil {
			return err
		}
	}

	if in.GetItemParentCacheAncestor() != nil {
		options.ItemParentCacheAncestor = &query.ItemParentCacheListOptions{}

		err := mapItemParentCacheListOptions(in.GetItemParentCacheAncestor(), options.ItemParentCacheAncestor)
		if err != nil {
			return err
		}
	}

	return nil
}

func mapPictureListOptions(in *PicturesOptions, options *query.PictureListOptions) error {
	options.ID = in.GetId()
	options.Status = convertPictureStatus(in.GetStatus())

	if in.GetPictureItem() != nil {
		options.PictureItem = &query.PictureItemListOptions{}

		err := mapPictureItemListOptions(in.GetPictureItem(), options.PictureItem)
		if err != nil {
			return err
		}
	}

	return nil
}

func convertPictureFields(fields *PictureFields) pictures.PictureFields {
	return pictures.PictureFields{
		NameText: fields.GetNameText(),
	}
}
