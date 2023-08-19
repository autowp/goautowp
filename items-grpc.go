package goautowp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/util"
	"github.com/autowp/goautowp/validation"
	"github.com/casbin/casbin"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const defaultCacheExpiration = 180 * time.Second

const itemLinkNameMaxLength = 255

type ItemsGRPCServer struct {
	UnimplementedItemsServer
	repository       *items.Repository
	db               *goqu.Database
	redis            *redis.Client
	auth             *Auth
	enforcer         *casbin.Enforcer
	contentLanguages []string
}

type BrandsCache struct {
	Items []items.Item
	Total int
}

func NewItemsGRPCServer(
	repository *items.Repository,
	db *goqu.Database,
	redis *redis.Client,
	auth *Auth,
	enforcer *casbin.Enforcer,
	contentLanguages []string,
) *ItemsGRPCServer {
	return &ItemsGRPCServer{
		repository:       repository,
		db:               db,
		redis:            redis,
		auth:             auth,
		enforcer:         enforcer,
		contentLanguages: contentLanguages,
	}
}

func (s *ItemsGRPCServer) GetTopBrandsList(
	ctx context.Context,
	in *GetTopBrandsListRequest,
) (*APITopBrandsList, error) {
	if s == nil {
		return nil, status.Error(codes.Internal, "self not initialized")
	}

	if s.redis == nil {
		return nil, status.Error(codes.Internal, "redis not initialized")
	}

	key := "GO_TOPBRANDSLIST_3_" + in.Language

	item, err := s.redis.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var cache BrandsCache

	if errors.Is(err, redis.Nil) {
		options := items.ListOptions{
			Language: in.Language,
			Fields: items.ListFields{
				Name:                true,
				DescendantsCount:    true,
				NewDescendantsCount: true,
			},
			TypeID:     []items.ItemType{items.BRAND},
			Limit:      items.TopBrandsCount,
			OrderBy:    []exp.OrderedExpression{goqu.I("descendants_count").Desc()},
			SortByName: true,
		}

		list, err := s.repository.List(ctx, options)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		count, err := s.repository.Count(ctx, options)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		cache.Items = list
		cache.Total = count

		b, err := json.Marshal(cache)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.redis.Set(ctx, key, string(b), defaultCacheExpiration).Err()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		err = json.Unmarshal([]byte(item), &cache)
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

func (s *ItemsGRPCServer) GetTopPersonsList(
	ctx context.Context,
	in *GetTopPersonsListRequest,
) (*APITopPersonsList, error) {
	var pictureItemType pictures.ItemPictureType

	switch in.PictureItemType { //nolint:exhaustive
	case PictureItemType_PICTURE_CONTENT:
		pictureItemType = pictures.ItemPictureContent
	case PictureItemType_PICTURE_AUTHOR:
		pictureItemType = pictures.ItemPictureAuthor
	default:
		return nil, status.Error(codes.InvalidArgument, "Unexpected picture_item_type")
	}

	key := fmt.Sprintf("GO_PERSONS_3_%d_%s", pictureItemType, in.Language)

	item, err := s.redis.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var res []items.Item

	if errors.Is(err, redis.Nil) {
		res, err = s.repository.List(ctx, items.ListOptions{
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
			OrderBy: []exp.OrderedExpression{goqu.L("COUNT(1)").Desc()},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		b, err := json.Marshal(res)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.redis.Set(ctx, key, string(b), defaultCacheExpiration).Err()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		err = json.Unmarshal([]byte(item), &res)
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

func (s *ItemsGRPCServer) GetTopFactoriesList(
	ctx context.Context,
	in *GetTopFactoriesListRequest,
) (*APITopFactoriesList, error) {
	key := fmt.Sprintf("GO_FACTORIES_3_%s", in.Language)

	item, err := s.redis.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var res []items.Item

	if errors.Is(err, redis.Nil) {
		res, err = s.repository.List(ctx, items.ListOptions{
			Language: in.Language,
			Fields: items.ListFields{
				Name:               true,
				ChildItemsCount:    true,
				NewChildItemsCount: true,
			},
			TypeID: []items.ItemType{items.FACTORY},
			ChildItems: &items.ListOptions{
				TypeID: []items.ItemType{items.VEHICLE, items.ENGINE},
			},
			Limit:   items.TopFactoriesCount,
			OrderBy: []exp.OrderedExpression{goqu.L("COUNT(1)").Desc()},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		b, err := json.Marshal(res)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.redis.Set(ctx, key, string(b), defaultCacheExpiration).Err()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		err = json.Unmarshal([]byte(item), &res)
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

func (s *ItemsGRPCServer) GetTopCategoriesList(
	ctx context.Context,
	in *GetTopCategoriesListRequest,
) (*APITopCategoriesList, error) {
	key := fmt.Sprintf("GO_CATEGORIES_6_%s", in.Language)

	item, err := s.redis.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var res []items.Item

	if errors.Is(err, redis.Nil) {
		res, err = s.repository.List(ctx, items.ListOptions{
			Language: in.Language,
			Fields: items.ListFields{
				Name:                true,
				DescendantsCount:    true,
				NewDescendantsCount: true,
			},
			NoParents: true,
			TypeID:    []items.ItemType{items.CATEGORY},
			Limit:     items.TopCategoriesCount,
			OrderBy:   []exp.OrderedExpression{goqu.I("descendants_count").Desc()},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		b, err := json.Marshal(res)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.redis.Set(ctx, key, string(b), defaultCacheExpiration).Err()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		err = json.Unmarshal([]byte(item), &res)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	is := make([]*APITopCategoriesListItem, len(res))
	for idx, b := range res {
		is[idx] = &APITopCategoriesListItem{
			Id:       b.ID,
			Name:     b.Name,
			Catname:  b.Catname,
			Count:    b.DescendantsCount,
			NewCount: b.NewDescendantsCount,
		}
	}

	return &APITopCategoriesList{
		Items: is,
	}, nil
}

func (s *ItemsGRPCServer) GetTopTwinsBrandsList(
	ctx context.Context,
	in *GetTopTwinsBrandsListRequest,
) (*APITopTwinsBrandsList, error) {
	key := fmt.Sprintf("GO_TWINS_5_%s", in.Language)

	item, err := s.redis.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, status.Error(codes.Internal, err.Error())
	}

	twinsData := struct {
		Count int
		Res   []items.Item
	}{
		0,
		nil,
	}

	if errors.Is(err, redis.Nil) {
		twinsData.Res, err = s.repository.List(ctx, items.ListOptions{
			Language: in.Language,
			Fields: items.ListFields{
				Name: true,
			},
			DescendantItems: &items.ListOptions{
				ParentItems: &items.ListOptions{
					TypeID: []items.ItemType{items.TWINS},
					Fields: items.ListFields{
						ItemsCount:    true,
						NewItemsCount: true,
					},
				},
			},
			TypeID:  []items.ItemType{items.BRAND},
			Limit:   items.TopTwinsBrandsCount,
			OrderBy: []exp.OrderedExpression{goqu.I("items_count").Desc()},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		twinsData.Count, err = s.repository.CountDistinct(ctx, items.ListOptions{
			DescendantItems: &items.ListOptions{
				ParentItems: &items.ListOptions{
					TypeID: []items.ItemType{items.TWINS},
				},
			},
			TypeID: []items.ItemType{items.BRAND},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		b, err := json.Marshal(twinsData)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.redis.Set(ctx, key, string(b), defaultCacheExpiration).Err()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		err = json.Unmarshal([]byte(item), &twinsData)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	is := make([]*APITwinsBrandsListItem, len(twinsData.Res))
	for idx, b := range twinsData.Res {
		is[idx] = &APITwinsBrandsListItem{
			Id:       b.ID,
			Catname:  b.Catname,
			Name:     b.Name,
			Count:    b.ItemsCount,
			NewCount: b.NewItemsCount,
		}
	}

	return &APITopTwinsBrandsList{
		Items: is,
		Count: int32(twinsData.Count),
	}, nil
}

func (s *ItemsGRPCServer) GetTwinsBrandsList(
	ctx context.Context,
	in *GetTwinsBrandsListRequest,
) (*APITwinsBrandsList, error) {
	twinsData, err := s.repository.List(ctx, items.ListOptions{
		Language: in.Language,
		Fields: items.ListFields{
			Name: true,
		},
		DescendantItems: &items.ListOptions{
			ParentItems: &items.ListOptions{
				TypeID: []items.ItemType{items.TWINS},
				Fields: items.ListFields{
					ItemsCount:    true,
					NewItemsCount: true,
				},
			},
		},
		TypeID:     []items.ItemType{items.BRAND},
		SortByName: true,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	is := make([]*APITwinsBrandsListItem, len(twinsData))
	for idx, b := range twinsData {
		is[idx] = &APITwinsBrandsListItem{
			Id:       b.ID,
			Catname:  b.Catname,
			Name:     b.Name,
			Count:    b.ItemsCount,
			NewCount: b.NewItemsCount,
		}
	}

	return &APITwinsBrandsList{
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

func (s *ItemsGRPCServer) List(ctx context.Context, in *ListItemsRequest) (*APIItemList, error) {
	options := items.ListOptions{
		Limit: in.Limit,
		Fields: items.ListFields{
			NameHTML:    in.Fields.NameHtml,
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

	res, err := s.repository.List(ctx, options)
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

func (s *ItemsGRPCServer) GetContentLanguages(_ context.Context, _ *emptypb.Empty) (*APIContentLanguages, error) {
	return &APIContentLanguages{
		Languages: s.contentLanguages,
	}, nil
}

func (s *ItemsGRPCServer) GetItemLink(ctx context.Context, in *APIItemLinkRequest) (*APIItemLink, error) {
	il := APIItemLink{}

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, url, type, item_id
		FROM links
		WHERE id = ?
	`, in.Id).Scan(&il.Id, &il.Name, &il.Url, &il.Type, &il.ItemId)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &il, nil
}

func (s *ItemsGRPCServer) GetItemLinks(ctx context.Context, in *APIGetItemLinksRequest) (*APIItemLinksResponse, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, url, type, item_id
		FROM links
		WHERE item_id = ?
	`, in.ItemId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	defer util.Close(rows)

	itemLinks := make([]*APIItemLink, 0)

	for rows.Next() {
		il := APIItemLink{}

		err = rows.Scan(&il.Id, &il.Name, &il.Url, &il.Type, &il.ItemId)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		itemLinks = append(itemLinks, &il)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &APIItemLinksResponse{
		Items: itemLinks,
	}, nil
}

func (s *ItemsGRPCServer) DeleteItemLink(ctx context.Context, in *APIItemLinkRequest) (*emptypb.Empty, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "car", "edit_meta") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM links WHERE id = ?", in.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ItemsGRPCServer) CreateItemLink(ctx context.Context, in *APIItemLink) (*APICreateItemLinkResponse, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "car", "edit_meta") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	InvalidParams, err := in.Validate()
	if err != nil {
		return nil, err
	}

	if len(InvalidParams) > 0 {
		return nil, wrapFieldViolations(InvalidParams)
	}

	res, err := s.db.ExecContext(
		ctx,
		"INSERT INTO links (name, url, type, item_id) VALUES (?, ?, ?, ?)",
		in.Name, in.Url, in.Type, in.ItemId,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APICreateItemLinkResponse{
		Id: id,
	}, nil
}

func (s *ItemsGRPCServer) UpdateItemLink(ctx context.Context, in *APIItemLink) (*emptypb.Empty, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "car", "edit_meta") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	InvalidParams, err := in.Validate()
	if err != nil {
		return nil, err
	}

	if len(InvalidParams) > 0 {
		return nil, wrapFieldViolations(InvalidParams)
	}

	_, err = s.db.ExecContext(
		ctx,
		"UPDATE links SET name = ?, url = ?, type = ?, item_id = ? WHERE id = ?",
		in.Name, in.Url, in.Type, in.ItemId, in.Id,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *APIItemLink) Validate() ([]*errdetails.BadRequest_FieldViolation, error) {
	var (
		result   = make([]*errdetails.BadRequest_FieldViolation, 0)
		problems []string
		err      error
	)

	nameInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}, &validation.StringSingleSpaces{}},
		Validators: []validation.ValidatorInterface{
			&validation.StringLength{Max: itemLinkNameMaxLength},
		},
	}
	s.Name, problems, err = nameInputFilter.IsValidString(s.Name)

	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "name",
			Description: fv,
		})
	}

	urlInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.URL{},
			&validation.StringLength{Max: itemLinkNameMaxLength},
		},
	}
	s.Url, problems, err = urlInputFilter.IsValidString(s.Url)

	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "url",
			Description: fv,
		})
	}

	typeInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.InArray{Haystack: []string{
				"default",
				"official",
				"club",
				"helper",
			}},
		},
	}
	s.Type, problems, err = typeInputFilter.IsValidString(s.Type)

	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "type",
			Description: fv,
		})
	}

	return result, nil
}

func (s *ItemsGRPCServer) GetItemVehicleTypes(
	ctx context.Context,
	in *APIGetItemVehicleTypesRequest,
) (*APIGetItemVehicleTypesResponse, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	if in.ItemId == 0 && in.VehicleTypeId == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	sqlSelect := s.db.Select("vehicle_id", "vehicle_type_id").From("vehicle_vehicle_type").Where(
		goqu.L("NOT inherited"),
	)

	if in.ItemId != 0 {
		sqlSelect = sqlSelect.Where(goqu.I("vehicle_id").Eq(in.ItemId))
	}

	if in.VehicleTypeId != 0 {
		sqlSelect = sqlSelect.Where(goqu.I("vehicle_type_id").Eq(in.VehicleTypeId))
	}

	rows, err := sqlSelect.Executor().QueryContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	defer util.Close(rows)

	res := make([]*APIItemVehicleType, 0)

	for rows.Next() {
		var i APIItemVehicleType

		err = rows.Scan(&i.ItemId, &i.VehicleTypeId)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		res = append(res, &i)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &APIGetItemVehicleTypesResponse{
		Items: res,
	}, nil
}

func (s *ItemsGRPCServer) GetItemVehicleType(
	ctx context.Context,
	in *APIItemVehicleTypeRequest,
) (*APIItemVehicleType, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	if in.ItemId == 0 && in.VehicleTypeId == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	sqlSelect := s.db.Select("vehicle_id", "vehicle_type_id").From("vehicle_vehicle_type").Where(
		goqu.L("NOT inherited"),
		goqu.I("vehicle_id").Eq(in.ItemId),
		goqu.I("vehicle_type_id").Eq(in.VehicleTypeId),
	)

	var i APIItemVehicleType

	rows, err := sqlSelect.Executor().QueryContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	defer util.Close(rows)

	rows.Next()

	err = rows.Scan(&i.ItemId, &i.VehicleTypeId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &i, nil
}

func (s *ItemsGRPCServer) CreateItemVehicleType(ctx context.Context, in *APIItemVehicleType) (*emptypb.Empty, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "car", "move") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	var found bool

	err = s.db.QueryRowContext(
		ctx,
		"SELECT 1 FROM item WHERE id = ? AND item_type_id IN (?, ?)",
		in.ItemId, items.VEHICLE, items.TWINS,
	).Scan(&found)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.repository.AddItemVehicleType(ctx, in.ItemId, in.VehicleTypeId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ItemsGRPCServer) DeleteItemVehicleType(
	ctx context.Context,
	in *APIItemVehicleTypeRequest,
) (*emptypb.Empty, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "car", "move") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	err = s.repository.RemoveItemVehicleType(ctx, in.ItemId, in.VehicleTypeId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}
