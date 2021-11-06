package goautowp

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/bradfitz/gomemcache/memcache"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ItemsGRPCServer struct {
	UnimplementedItemsServer
	repository *items.Repository
	memcached  *memcache.Client
}

func NewItemsGRPCServer(
	repository *items.Repository,
	memcached *memcache.Client,
) *ItemsGRPCServer {
	return &ItemsGRPCServer{
		repository: repository,
		memcached:  memcached,
	}
}

func (s *ItemsGRPCServer) GetTopBrandsList(_ context.Context, in *GetTopBrandsListRequest) (*APITopBrandsList, error) {

	if s == nil {
		return nil, status.Error(codes.Internal, "self not initialized")
	}

	if s.memcached == nil {
		return nil, status.Error(codes.Internal, "memcached not initialized")
	}

	key := "GO_TOPBRANDSLIST_" + in.Language

	item, err := s.memcached.Get(key)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var res *items.TopBrandsListResult

	if err == memcache.ErrCacheMiss {
		res, err = s.repository.TopBrandList(in.Language)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		b := new(bytes.Buffer)
		err = gob.NewEncoder(b).Encode(res)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.memcached.Set(&memcache.Item{
			Key:        key,
			Value:      b.Bytes(),
			Expiration: 180,
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

	} else {
		decoder := gob.NewDecoder(bytes.NewBuffer(item.Value))
		err = decoder.Decode(&res)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	brands := make([]*APITopBrandsListItem, len(res.Brands))
	for idx, b := range res.Brands {
		brands[idx] = &APITopBrandsListItem{
			Id:            b.ID,
			Catname:       b.Catname,
			Name:          b.Name,
			ItemsCount:    b.ItemsCount,
			NewItemsCount: b.NewItemsCount,
		}
	}

	return &APITopBrandsList{
		Brands: brands,
		Total:  int32(res.Total),
	}, nil
}

func (s *ItemsGRPCServer) GetTopPersonsList(_ context.Context, in *GetTopPersonsListRequest) (*APITopPersonsList, error) {
	var pictureItemType pictures.ItemPictureType

	switch in.PictureItemType {
	case PictureItemType_PICTURE_CONTENT:
		pictureItemType = pictures.ItemPictureContent
	case PictureItemType_PICTURE_AUTHOR:
		pictureItemType = pictures.ItemPictureAuthor
	default:
		return nil, status.Error(codes.InvalidArgument, "Unexpected picture_item_type")
	}

	key := fmt.Sprintf("GO_PERSONS_%d_%s", pictureItemType, in.Language)

	item, err := s.memcached.Get(key)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var res *items.ListResult

	if err == memcache.ErrCacheMiss {

		res, err = s.repository.List(items.ListOptions{
			Language: in.Language,
			Fields: items.ListFields{
				Name: true,
			},
			TypeID: items.PERSON,
			DescendantPictures: &items.ItemPicturesOptions{
				TypeID: pictureItemType,
				Pictures: &items.PicturesOptions{
					Status: pictures.StatusAccepted,
				},
			},
			Limit:   items.TopPersonsCount,
			OrderBy: "COUNT(1) DESC",
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		b := new(bytes.Buffer)
		err = gob.NewEncoder(b).Encode(res)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.memcached.Set(&memcache.Item{
			Key:        key,
			Value:      b.Bytes(),
			Expiration: 180,
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

	} else {
		decoder := gob.NewDecoder(bytes.NewBuffer(item.Value))
		err = decoder.Decode(&res)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	is := make([]*APITopPersonsListItem, len(res.Items))
	for idx, b := range res.Items {
		is[idx] = &APITopPersonsListItem{
			Id:   b.ID,
			Name: b.Name,
		}
	}

	return &APITopPersonsList{
		Items: is,
	}, nil
}

func mapPicturesRequest(request *PicturesRequest, dest *items.PicturesOptions) {
	switch request.Status {
	case PictureStatus_PICTURE_STATUS_UNKNOWN:
	case PictureStatus_PICTURE_STATUS_ACCEPTED:
		dest.Status = pictures.StatusAccepted
	case PictureStatus_PICTURE_STATUS_REMOVING:
		dest.Status = pictures.StatusRemoving
	case PictureStatus_PICTURE_STATUS_INBOX:
		dest.Status = pictures.StatusInbox
	case PictureStatus_PICTURE_STATUS_REMOVED:
		dest.Status = pictures.StatusRemoved
	}

	if request.ItemPicture != nil {
		mapItemPicturesRequest(request.ItemPicture, dest.ItemPicture)
	}
}

func mapItemPicturesRequest(request *ItemPicturesRequest, dest *items.ItemPicturesOptions) {
	if request.Pictures != nil {
		mapPicturesRequest(request.Pictures, dest.Pictures)
	}

	switch request.TypeId {
	case ItemPictureType_ITEM_PICTURE_UNKNOWN:
	case ItemPictureType_ITEM_PICTURE_CONTENT:
		dest.TypeID = pictures.ItemPictureContent
	case ItemPictureType_ITEM_PICTURE_AUTHOR:
		dest.TypeID = pictures.ItemPictureAuthor
	case ItemPictureType_ITEM_PICTURE_COPYRIGHTS:
		dest.TypeID = pictures.ItemPictureCopyrights
	}
	dest.PerspectiveID = request.PerspectiveId
}

func (s *ItemsGRPCServer) List(_ context.Context, in *ListItemsRequest) (*APIItemList, error) {

	options := items.ListOptions{
		Limit: in.Limit,
		Fields: items.ListFields{
			NameHtml:    in.Fields.NameHtml,
			NameDefault: in.Fields.NameDefault,
			Description: in.Fields.Description,
			HasText:     in.Fields.HasText,
			PreviewPictures: items.ListPreviewPicturesFields{
				Route: in.Fields.PreviewPictures.Route,
				Picture: items.ListPreviewPicturesPictureFields{
					NameText: in.Fields.PreviewPictures.Picture.NameText,
				},
			},
			TotalPictures: in.Fields.TotalPictures,
		},
	}

	switch in.TypeId {
	case ItemType_ITEM_TYPE_UNKNOWN:
	case ItemType_ITEM_TYPE_VEHICLE:
		options.TypeID = items.VEHICLE
	case ItemType_ITEM_TYPE_ENGINE:
		options.TypeID = items.ENGINE
	case ItemType_ITEM_TYPE_CATEGORY:
		options.TypeID = items.CATEGORY
	case ItemType_ITEM_TYPE_TWINS:
		options.TypeID = items.TWINS
	case ItemType_ITEM_TYPE_BRAND:
		options.TypeID = items.BRAND
	case ItemType_ITEM_TYPE_FACTORY:
		options.TypeID = items.FACTORY
	case ItemType_ITEM_TYPE_MUSEUM:
		options.TypeID = items.MUSEUM
	case ItemType_ITEM_TYPE_PERSON:
		options.TypeID = items.PERSON
	case ItemType_ITEM_TYPE_COPYRIGHT:
		options.TypeID = items.COPYRIGHT
	default:
		return nil, status.Error(codes.InvalidArgument, "Unexpected item_type")
	}

	if in.DescendantPictures != nil {
		mapItemPicturesRequest(in.DescendantPictures, options.DescendantPictures)
	}
	if in.PreviewPictures != nil {
		mapItemPicturesRequest(in.PreviewPictures, options.PreviewPictures)
	}

	/**
	fields: name_html,name_default,description,has_text,preview_pictures.route,preview_pictures.picture.name_text,total_pictures
	*/

	res, err := s.repository.List(options)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	is := make([]*APIItem, len(res.Items))
	for idx, i := range res.Items {
		is[idx] = &APIItem{
			Id:      i.ID,
			Catname: i.Catname,
			Name:    i.Name,
		}
	}

	return &APIItemList{
		Items: is,
	}, nil
}
