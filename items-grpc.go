package goautowp

import (
	"bytes"
	"context"
	"encoding/gob"
	"github.com/autowp/goautowp/items"
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
