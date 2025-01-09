package goautowp

import (
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
)

func convertChildsCounts(childCounts []items.ChildCount) []*ChildsCount {
	result := make([]*ChildsCount, 0, len(childCounts))

	for _, childCount := range childCounts {
		result = append(result, &ChildsCount{
			Type:  convertItemParentType(childCount.Type),
			Count: childCount.Count,
		})
	}

	return result
}

func convertItemTypeID(itemTypeID schema.ItemTableItemTypeID) ItemType {
	switch itemTypeID {
	case schema.ItemTableItemTypeIDVehicle:
		return ItemType_ITEM_TYPE_VEHICLE
	case schema.ItemTableItemTypeIDEngine:
		return ItemType_ITEM_TYPE_ENGINE
	case schema.ItemTableItemTypeIDCategory:
		return ItemType_ITEM_TYPE_CATEGORY
	case schema.ItemTableItemTypeIDTwins:
		return ItemType_ITEM_TYPE_TWINS
	case schema.ItemTableItemTypeIDBrand:
		return ItemType_ITEM_TYPE_BRAND
	case schema.ItemTableItemTypeIDFactory:
		return ItemType_ITEM_TYPE_FACTORY
	case schema.ItemTableItemTypeIDMuseum:
		return ItemType_ITEM_TYPE_MUSEUM
	case schema.ItemTableItemTypeIDPerson:
		return ItemType_ITEM_TYPE_PERSON
	case schema.ItemTableItemTypeIDCopyright:
		return ItemType_ITEM_TYPE_COPYRIGHT
	}

	return ItemType_ITEM_TYPE_UNKNOWN
}

func reverseConvertItemTypeID(itemTypeID ItemType) schema.ItemTableItemTypeID {
	switch itemTypeID {
	case ItemType_ITEM_TYPE_UNKNOWN:
		return 0
	case ItemType_ITEM_TYPE_VEHICLE:
		return schema.ItemTableItemTypeIDVehicle
	case ItemType_ITEM_TYPE_ENGINE:
		return schema.ItemTableItemTypeIDEngine
	case ItemType_ITEM_TYPE_CATEGORY:
		return schema.ItemTableItemTypeIDCategory
	case ItemType_ITEM_TYPE_TWINS:
		return schema.ItemTableItemTypeIDTwins
	case ItemType_ITEM_TYPE_BRAND:
		return schema.ItemTableItemTypeIDBrand
	case ItemType_ITEM_TYPE_FACTORY:
		return schema.ItemTableItemTypeIDFactory
	case ItemType_ITEM_TYPE_MUSEUM:
		return schema.ItemTableItemTypeIDMuseum
	case ItemType_ITEM_TYPE_PERSON:
		return schema.ItemTableItemTypeIDPerson
	case ItemType_ITEM_TYPE_COPYRIGHT:
		return schema.ItemTableItemTypeIDCopyright
	}

	return 0
}

func convertItemParentType(itemParentType schema.ItemParentType) ItemParentType {
	switch itemParentType {
	case schema.ItemParentTypeDefault:
		return ItemParentType_ITEM_TYPE_DEFAULT
	case schema.ItemParentTypeTuning:
		return ItemParentType_ITEM_TYPE_TUNING
	case schema.ItemParentTypeSport:
		return ItemParentType_ITEM_TYPE_SPORT
	case schema.ItemParentTypeDesign:
		return ItemParentType_ITEM_TYPE_DESIGN
	}

	return ItemParentType_ITEM_TYPE_DEFAULT
}

func reverseConvertItemParentType(itemParentType ItemParentType) schema.ItemParentType {
	switch itemParentType {
	case ItemParentType_ITEM_TYPE_DEFAULT:
		return schema.ItemParentTypeDefault
	case ItemParentType_ITEM_TYPE_TUNING:
		return schema.ItemParentTypeTuning
	case ItemParentType_ITEM_TYPE_SPORT:
		return schema.ItemParentTypeSport
	case ItemParentType_ITEM_TYPE_DESIGN:
		return schema.ItemParentTypeDesign
	}

	return schema.ItemParentTypeDefault
}

func mapPicturesRequest(request *PicturesOptions, dest *query.PictureListOptions) {
	dest.OwnerID = request.GetOwnerId()

	switch request.GetStatus() {
	case PictureStatus_PICTURE_STATUS_UNKNOWN:
	case PictureStatus_PICTURE_STATUS_ACCEPTED:
		dest.Status = schema.PictureStatusAccepted
	case PictureStatus_PICTURE_STATUS_REMOVING:
		dest.Status = schema.PictureStatusRemoving
	case PictureStatus_PICTURE_STATUS_INBOX:
		dest.Status = schema.PictureStatusInbox
	case PictureStatus_PICTURE_STATUS_REMOVED:
		dest.Status = schema.PictureStatusRemoved
	}

	if request.GetPictureItem() != nil {
		dest.PictureItem = &query.PictureItemListOptions{}
		mapPictureItemRequest(request.GetPictureItem(), dest.PictureItem)
	}
}

func mapItemParentListOptions(in *ItemParentListOptions, options *query.ItemParentListOptions) error {
	options.ParentID = in.GetParentId()
	options.ItemID = in.GetItemId()
	options.Type = reverseConvertItemParentType(in.GetType())
	options.Catname = in.GetCatname()

	if in.GetParent() != nil {
		options.ParentItems = &query.ItemsListOptions{}

		err := mapItemListOptions(in.GetParent(), options.ParentItems)
		if err != nil {
			return err
		}
	}

	if in.GetItemParentParentByChild() != nil {
		options.ItemParentParentByChildID = &query.ItemParentListOptions{}

		err := mapItemParentListOptions(in.GetItemParentParentByChild(), options.ItemParentParentByChildID)
		if err != nil {
			return err
		}
	}

	if in.GetItem() != nil {
		options.ChildItems = &query.ItemsListOptions{}

		err := mapItemListOptions(in.GetItem(), options.ChildItems)
		if err != nil {
			return err
		}
	}

	if in.GetItemParentCacheItemByChild() != nil {
		options.ItemParentCacheAncestorByChildID = &query.ItemParentCacheListOptions{}

		err := mapItemParentCacheListOptions(in.GetItemParentCacheItemByChild(), options.ItemParentCacheAncestorByChildID)
		if err != nil {
			return err
		}
	}

	return nil
}

func mapItemParentCacheListOptions(in *ItemParentCacheListOptions, options *query.ItemParentCacheListOptions) error {
	options.ItemID = in.GetItemId()
	options.ParentID = in.GetParentId()

	if in.GetItemsByItemId() != nil {
		options.ItemsByItemID = &query.ItemsListOptions{}

		err := mapItemListOptions(in.GetItemsByItemId(), options.ItemsByItemID)
		if err != nil {
			return err
		}
	}

	if in.GetPictureItemsByItemId() != nil {
		options.PictureItemsByItemID = &query.PictureItemListOptions{}

		mapPictureItemRequest(in.GetPictureItemsByItemId(), options.PictureItemsByItemID)
	}

	if in.GetItemParentByItemId() != nil {
		options.ItemParentByItemID = &query.ItemParentListOptions{}

		err := mapItemParentListOptions(in.GetItemParentByItemId(), options.ItemParentByItemID)
		if err != nil {
			return err
		}
	}

	return nil
}

func mapItemListOptions(in *ItemListOptions, options *query.ItemsListOptions) error {
	options.NoParents = in.GetNoParent()
	options.Catname = in.GetCatname()
	options.IsConcept = in.GetIsConcept()
	options.IsNotConcept = in.GetIsNotConcept()
	options.Name = in.GetName()
	options.ItemID = in.GetId()
	options.EngineItemID = in.GetEngineId()
	options.IsGroup = in.GetIsGroup()
	options.Autocomplete = in.GetAutocomplete()
	options.SuggestionsTo = in.GetSuggestionsTo()

	if in.GetAncestor() != nil {
		options.ItemParentCacheAncestor = &query.ItemParentCacheListOptions{}

		err := mapItemParentCacheListOptions(in.GetAncestor(), options.ItemParentCacheAncestor)
		if err != nil {
			return err
		}
	}

	itemTypeID := reverseConvertItemTypeID(in.GetTypeId())
	if itemTypeID != 0 {
		options.TypeID = []schema.ItemTableItemTypeID{itemTypeID}
	}

	typeIDs := in.GetTypeIds()
	if len(in.GetTypeIds()) > 0 {
		ids := make([]schema.ItemTableItemTypeID, 0, len(typeIDs))
		for _, id := range in.GetTypeIds() {
			ids = append(ids, reverseConvertItemTypeID(id))
		}

		options.TypeID = ids
	}

	if in.GetDescendant() != nil {
		options.ItemParentCacheDescendant = &query.ItemParentCacheListOptions{}

		err := mapItemParentCacheListOptions(in.GetDescendant(), options.ItemParentCacheDescendant)
		if err != nil {
			return err
		}
	}

	if in.GetParent() != nil {
		options.ItemParentParent = &query.ItemParentListOptions{}

		err := mapItemParentListOptions(in.GetParent(), options.ItemParentParent)
		if err != nil {
			return err
		}
	}

	if in.GetChild() != nil {
		options.ItemParentChild = &query.ItemParentListOptions{}

		err := mapItemParentListOptions(in.GetChild(), options.ItemParentChild)
		if err != nil {
			return err
		}
	}

	if in.GetPreviewPictures() != nil {
		options.PreviewPictures = &query.PictureItemListOptions{}
		mapPictureItemRequest(in.GetPreviewPictures(), options.PreviewPictures)
	}

	parentTypesOf := reverseConvertItemTypeID(in.GetParentTypesOf())
	if parentTypesOf != 0 {
		options.ParentTypesOf = parentTypesOf
	}

	if in.GetExcludeSelfAndChilds() != 0 {
		options.ExcludeSelfAndChilds = in.GetExcludeSelfAndChilds()
	}

	return nil
}

func mapPictureItemRequest(request *PictureItemOptions, dest *query.PictureItemListOptions) {
	if request.GetPictures() != nil {
		dest.Pictures = &query.PictureListOptions{}
		mapPicturesRequest(request.GetPictures(), dest.Pictures)
	}

	switch request.GetTypeId() {
	case PictureItemType_PICTURE_ITEM_UNKNOWN:
	case PictureItemType_PICTURE_ITEM_CONTENT:
		dest.TypeID = schema.PictureItemContent
	case PictureItemType_PICTURE_ITEM_AUTHOR:
		dest.TypeID = schema.PictureItemAuthor
	case PictureItemType_PICTURE_ITEM_COPYRIGHTS:
		dest.TypeID = schema.PictureItemCopyrights
	}

	dest.PerspectiveID = request.GetPerspectiveId()
}

func convertFields(fields *ItemFields) items.ListFields {
	if fields == nil {
		return items.ListFields{}
	}

	previewPictures := items.ListPreviewPicturesFields{}
	if fields.GetPreviewPictures() != nil {
		previewPictures.Route = fields.GetPreviewPictures().GetRoute()
		previewPictures.Picture = items.ListPreviewPicturesPictureFields{
			NameText: fields.GetPreviewPictures().GetPicture().GetNameText(),
		}
	}

	result := items.ListFields{
		NameOnly:        fields.GetNameOnly(),
		NameHTML:        fields.GetNameHtml(),
		NameText:        fields.GetNameText(),
		NameDefault:     fields.GetNameDefault(),
		Description:     fields.GetDescription(),
		FullText:        fields.GetFullText(),
		HasText:         fields.GetHasText(),
		PreviewPictures: previewPictures,
		// TotalPictures:              fields.GetTotalPictures(),
		DescendantsCount:           fields.GetDescendantsCount(),
		DescendantPicturesCount:    fields.GetDescendantPicturesCount(),
		ChildsCount:                fields.GetChildsCount(),
		DescendantTwinsGroupsCount: fields.GetDescendantTwinsGroupsCount(),
		InboxPicturesCount:         fields.GetInboxPicturesCount(),
		AcceptedPicturesCount:      fields.GetAcceptedPicturesCount(),
		FullName:                   fields.GetFullName(),
		Logo:                       fields.GetLogo120(),
		MostsActive:                fields.GetMostsActive(),
		CommentsAttentionsCount:    fields.GetCommentsAttentionsCount(),
		HasChildSpecs:              fields.GetHasChildSpecs(),
		HasSpecs:                   fields.GetHasSpecs() || fields.GetSpecsRoute(),
	}

	return result
}

func convertItemParentFields(_ *ItemParentFields) items.ItemParentFields {
	return items.ItemParentFields{}
}
