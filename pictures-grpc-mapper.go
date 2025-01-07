package goautowp

import (
	"time"

	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
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
	options.ExcludePerspectiveID = in.GetExcludePerspectiveId()
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
	options.AcceptedInDays = in.GetAcceptedInDays()

	addDate := in.GetAddDate()
	if addDate != nil {
		options.AddDate = &util.Date{
			Year:  int(addDate.GetYear()),
			Month: time.Month(addDate.GetMonth()),
			Day:   int(addDate.GetDay()),
		}
	}

	acceptDate := in.GetAcceptDate()
	if acceptDate != nil {
		options.AcceptDate = &util.Date{
			Year:  int(acceptDate.GetYear()),
			Month: time.Month(acceptDate.GetMonth()),
			Day:   int(acceptDate.GetDay()),
		}
	}

	pictureItem := in.GetPictureItem()
	if pictureItem != nil {
		options.PictureItem = &query.PictureItemListOptions{}

		err := mapPictureItemListOptions(pictureItem, options.PictureItem)
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

func convertPicturesOrder(order GetPicturesRequest_Order) pictures.OrderBy {
	switch order {
	case GetPicturesRequest_NONE:
		return pictures.OrderByNone
	case GetPicturesRequest_ADD_DATE_DESC:
		return pictures.OrderByAddDateDesc
	case GetPicturesRequest_ADD_DATE_ASC:
		return pictures.OrderByAddDateAsc
	case GetPicturesRequest_RESOLUTION_DESC:
		return pictures.OrderByResolutionDesc
	case GetPicturesRequest_RESOLUTION_ASC:
		return pictures.OrderByResolutionAsc
	case GetPicturesRequest_LIKES:
		return pictures.OrderByLikes
	case GetPicturesRequest_DISLIKES:
		return pictures.OrderByDislikes
	case GetPicturesRequest_ACCEPT_DATETIME_DESC:
		return pictures.OrderByAcceptDatetimeDesc
	case GetPicturesRequest_PERSPECTIVES:
		return pictures.OrderByPerspectives
	}

	return pictures.OrderByNone
}
