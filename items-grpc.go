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

type BrandsCache struct {
	Items []items.Item
	Total int
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

	key := "GO_TOPBRANDSLIST_2_" + in.Language

	item, err := s.memcached.Get(key)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var cache BrandsCache

	if err == memcache.ErrCacheMiss {
		options := items.ItemsOptions{
			Language: in.Language,
			Fields: items.ListFields{
				Name:                true,
				DescendantsCount:    true,
				NewDescendantsCount: true,
			},
			TypeID:     []items.ItemType{items.BRAND},
			Limit:      items.TopBrandsCount,
			OrderBy:    "descendants_count DESC",
			SortByName: true,
		}
		list, err := s.repository.List(options)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		count, err := s.repository.Count(options)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		cache.Items = list
		cache.Total = count

		b := new(bytes.Buffer)
		err = gob.NewEncoder(b).Encode(cache)
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
		err = decoder.Decode(&cache)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	brands := make([]*APITopBrandsListItem, len(cache.Items))
	for idx, b := range cache.Items {
		brands[idx] = &APITopBrandsListItem{
			Id:            b.ID,
			Catname:       b.Catname,
			Name:          b.Name,
			ItemsCount:    b.DescendantsCount,
			NewItemsCount: b.NewDescendantsCount,
		}
	}

	return &APITopBrandsList{
		Brands: brands,
		Total:  int32(cache.Total),
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

	var res []items.Item

	if err == memcache.ErrCacheMiss {

		res, err = s.repository.List(items.ItemsOptions{
			Language: in.Language,
			Fields: items.ListFields{
				Name: true,
			},
			TypeID: []items.ItemType{items.PERSON},
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

	is := make([]*APITopPersonsListItem, len(res))
	for idx, b := range res {
		is[idx] = &APITopPersonsListItem{
			Id:   b.ID,
			Name: b.Name,
		}
	}

	return &APITopPersonsList{
		Items: is,
	}, nil
}

func (s *ItemsGRPCServer) GetTopFactoriesList(_ context.Context, in *GetTopFactoriesListRequest) (*APITopFactoriesList, error) {

	key := fmt.Sprintf("GO_FACTORIES_%s", in.Language)

	item, err := s.memcached.Get(key)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var res []items.Item

	if err == memcache.ErrCacheMiss {

		res, err = s.repository.List(items.ItemsOptions{
			Language: in.Language,
			Fields: items.ListFields{
				Name:               true,
				ChildItemsCount:    true,
				NewChildItemsCount: true,
			},
			TypeID: []items.ItemType{items.FACTORY},
			ChildItems: &items.ItemsOptions{
				TypeID: []items.ItemType{items.VEHICLE, items.ENGINE},
			},
			Limit:   items.TopFactoriesCount,
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

	is := make([]*APITopFactoriesListItem, len(res))
	for idx, b := range res {
		is[idx] = &APITopFactoriesListItem{
			Id:       b.ID,
			Name:     b.Name,
			Count:    b.ChildItemsCount,
			NewCount: b.NewChildItemsCount,
		}
	}

	return &APITopFactoriesList{
		Items: is,
	}, nil
}

func (s *ItemsGRPCServer) GetTopCategoriesList(_ context.Context, in *GetTopCategoriesListRequest) (*APITopCategoriesList, error) {

	key := fmt.Sprintf("GO_CATEGORIES_3_%s", in.Language)

	item, err := s.memcached.Get(key)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var res []items.Item

	if err == memcache.ErrCacheMiss {

		res, err = s.repository.List(items.ItemsOptions{
			Language: in.Language,
			Fields: items.ListFields{
				Name:                true,
				DescendantsCount:    true,
				NewDescendantsCount: true,
			},
			NoParents: true,
			TypeID:    []items.ItemType{items.CATEGORY},
			Limit:     items.TopCategoriesCount,
			OrderBy:   "descendants_count DESC",
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

	is := make([]*APITopCategoriesListItem, len(res))
	for idx, b := range res {
		is[idx] = &APITopCategoriesListItem{
			Id:       b.ID,
			Name:     b.Name,
			Count:    b.DescendantsCount,
			NewCount: b.NewDescendantsCount,
		}
	}

	return &APITopCategoriesList{
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

	options := items.ItemsOptions{
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
		options.TypeID = []items.ItemType{items.VEHICLE}
	case ItemType_ITEM_TYPE_ENGINE:
		options.TypeID = []items.ItemType{items.ENGINE}
	case ItemType_ITEM_TYPE_CATEGORY:
		options.TypeID = []items.ItemType{items.CATEGORY}
	case ItemType_ITEM_TYPE_TWINS:
		options.TypeID = []items.ItemType{items.TWINS}
	case ItemType_ITEM_TYPE_BRAND:
		options.TypeID = []items.ItemType{items.BRAND}
	case ItemType_ITEM_TYPE_FACTORY:
		options.TypeID = []items.ItemType{items.FACTORY}
	case ItemType_ITEM_TYPE_MUSEUM:
		options.TypeID = []items.ItemType{items.MUSEUM}
	case ItemType_ITEM_TYPE_PERSON:
		options.TypeID = []items.ItemType{items.PERSON}
	case ItemType_ITEM_TYPE_COPYRIGHT:
		options.TypeID = []items.ItemType{items.COPYRIGHT}
	default:
		return nil, status.Error(codes.InvalidArgument, "Unexpected item_type")
	}

	if in.DescendantPictures != nil {
		mapItemPicturesRequest(in.DescendantPictures, options.DescendantPictures)
	}
	if in.PreviewPictures != nil {
		mapItemPicturesRequest(in.PreviewPictures, options.PreviewPictures)
	}

	res, err := s.repository.List(options)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	is := make([]*APIItem, len(res))
	for idx, i := range res {
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
