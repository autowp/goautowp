package goautowp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/autowp/goautowp/attrs"
	"github.com/autowp/goautowp/frontend"
	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/index"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/textstorage"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"github.com/autowp/goautowp/validation"
	"github.com/casbin/casbin"
	"github.com/doug-martin/goqu/v9"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const itemLinkNameMaxLength = 255

const typicalPicturesInList = 4

func (s *ItemParent) Validate() ([]*errdetails.BadRequest_FieldViolation, error) {
	var (
		result   = make([]*errdetails.BadRequest_FieldViolation, 0)
		problems []string
		err      error
	)

	catnameInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{
			&validation.StringTrimFilter{},
			&validation.StringSingleSpaces{},
			&validation.StringToLower{},
			&validation.StringSanitizeFilename{},
		},
		Validators: []validation.ValidatorInterface{
			&validation.StringLength{Max: schema.ItemParentMaxCatname},
		},
	}

	s.Catname, problems, err = catnameInputFilter.IsValidString(s.GetCatname())
	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "name",
			Description: fv,
		})
	}

	return result, nil
}

type ItemsGRPCServer struct {
	UnimplementedItemsServer
	repository            *items.Repository
	db                    *goqu.Database
	auth                  *Auth
	enforcer              *casbin.Enforcer
	contentLanguages      []string
	textstorageRepository *textstorage.Repository
	extractor             *ItemExtractor
	i18n                  *i18nbundle.I18n
	attrsRepository       *attrs.Repository
	picturesRepository    *pictures.Repository
	index                 *index.Index
	events                *Events
	usersRepository       *users.Repository
	messagingRepository   *messaging.Repository
	hostManager           *hosts.Manager
}

func NewItemsGRPCServer(
	repository *items.Repository,
	db *goqu.Database,
	auth *Auth,
	enforcer *casbin.Enforcer,
	contentLanguages []string,
	textstorageRepository *textstorage.Repository,
	extractor *ItemExtractor,
	i18n *i18nbundle.I18n,
	attrsRepository *attrs.Repository,
	picturesRepository *pictures.Repository,
	index *index.Index,
	events *Events,
	usersRepository *users.Repository,
	messagingRepository *messaging.Repository,
	hostManager *hosts.Manager,
) *ItemsGRPCServer {
	return &ItemsGRPCServer{
		repository:            repository,
		db:                    db,
		auth:                  auth,
		enforcer:              enforcer,
		contentLanguages:      contentLanguages,
		textstorageRepository: textstorageRepository,
		extractor:             extractor,
		i18n:                  i18n,
		attrsRepository:       attrsRepository,
		picturesRepository:    picturesRepository,
		index:                 index,
		events:                events,
		usersRepository:       usersRepository,
		messagingRepository:   messagingRepository,
		hostManager:           hostManager,
	}
}

func (s *ItemsGRPCServer) GetBrands(ctx context.Context, in *GetBrandsRequest) (*APIBrandsList, error) {
	if s == nil {
		return nil, status.Error(codes.Internal, "self not initialized")
	}

	lang := in.GetLanguage()

	resultArray, err := s.index.BrandsCache(ctx, lang)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*APIBrandsListLine, 0, len(resultArray))

	for _, line := range resultArray {
		characters := make([]*APIBrandsListCharacter, 0, len(line.Characters))

		for _, character := range line.Characters {
			rows := make([]*APIBrandsListItem, 0, len(character.Items))
			for _, item := range character.Items {
				rows = append(rows, &APIBrandsListItem{
					Id:                    item.ID,
					Catname:               item.Catname,
					Name:                  item.Name,
					ItemsCount:            item.ItemsCount,
					NewItemsCount:         item.NewItemsCount,
					AcceptedPicturesCount: item.AcceptedPicturesCount,
				})
			}

			characters = append(characters, &APIBrandsListCharacter{
				Id:        character.ID,
				Character: character.Character,
				Items:     rows,
			})
		}

		var category APIBrandsListLine_Category

		switch line.Category {
		case items.BrandsListCategoryDefault:
			category = APIBrandsListLine_DEFAULT
		case items.BrandsListCategoryNumber:
			category = APIBrandsListLine_NUMBER
		case items.BrandsListCategoryCyrillic:
			category = APIBrandsListLine_CYRILLIC
		case items.BrandsListCategoryLatin:
			category = APIBrandsListLine_LATIN
		}

		result = append(result, &APIBrandsListLine{
			Category:   category,
			Characters: characters,
		})
	}

	return &APIBrandsList{Lines: result}, nil
}

func (s *ItemsGRPCServer) GetTopBrandsList(
	ctx context.Context,
	in *GetTopBrandsListRequest,
) (*APITopBrandsList, error) {
	if s == nil {
		return nil, status.Error(codes.Internal, "self not initialized")
	}

	cache, err := s.index.TopBrandsCache(ctx, in.GetLanguage())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	brands := make([]*APITopBrandsListItem, len(cache.Items))
	for idx, brand := range cache.Items {
		brands[idx] = &APITopBrandsListItem{
			Id:            brand.ID,
			Catname:       util.NullStringToString(brand.Catname),
			Name:          brand.NameOnly,
			ItemsCount:    brand.DescendantsCount,
			NewItemsCount: brand.NewDescendantsCount,
		}
	}

	return &APITopBrandsList{
		Brands: brands,
		Total:  int32(cache.Total), //nolint: gosec
	}, nil
}

func (s *ItemsGRPCServer) GetTopPersonsList(
	ctx context.Context,
	in *GetTopPersonsListRequest,
) (*APITopPersonsList, error) {
	var pictureItemType schema.PictureItemType

	switch in.GetPictureItemType() { //nolint:exhaustive
	case PictureItemType_PICTURE_ITEM_CONTENT:
		pictureItemType = schema.PictureItemContent
	case PictureItemType_PICTURE_ITEM_AUTHOR:
		pictureItemType = schema.PictureItemAuthor
	default:
		return nil, status.Error(codes.InvalidArgument, "Unexpected picture_item_type")
	}

	res, err := s.index.PersonsCache(ctx, pictureItemType, in.GetLanguage())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	is := make([]*APITopPersonsListItem, len(res))
	for idx, b := range res {
		is[idx] = &APITopPersonsListItem{
			Id:   b.ID,
			Name: b.NameOnly,
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
	res, err := s.index.FactoriesCache(ctx, in.GetLanguage())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	is := make([]*APITopFactoriesListItem, len(res))
	for idx, b := range res {
		is[idx] = &APITopFactoriesListItem{
			Id:       b.ID,
			Name:     b.NameOnly,
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
	res, err := s.index.CategoriesCache(ctx, in.GetLanguage())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	is := make([]*APITopCategoriesListItem, len(res))
	for idx, itm := range res {
		is[idx] = &APITopCategoriesListItem{
			Id:       itm.ID,
			Name:     itm.NameOnly,
			Catname:  util.NullStringToString(itm.Catname),
			Count:    itm.DescendantsCount,
			NewCount: itm.NewDescendantsCount,
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
	twinsData, err := s.index.TwinsCache(ctx, in.GetLanguage())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	is := make([]*APITwinsBrandsListItem, len(twinsData.Res))
	for idx, twin := range twinsData.Res {
		is[idx] = &APITwinsBrandsListItem{
			Id:       twin.ID,
			Catname:  util.NullStringToString(twin.Catname),
			Name:     twin.NameOnly,
			Count:    twin.DescendantsParentsCount,
			NewCount: twin.NewDescendantsParentsCount,
		}
	}

	return &APITopTwinsBrandsList{
		Items: is,
		Count: int32(twinsData.Count), //nolint: gosec
	}, nil
}

func (s *ItemsGRPCServer) GetTwinsBrandsList(
	ctx context.Context,
	in *GetTwinsBrandsListRequest,
) (*APITwinsBrandsList, error) {
	twinsData, _, err := s.repository.List(ctx, query.ItemsListOptions{
		Language: in.GetLanguage(),
		ItemParentCacheDescendant: &query.ItemParentCacheListOptions{
			ItemParentByItemID: &query.ItemParentListOptions{
				ParentItems: &query.ItemsListOptions{
					TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDTwins},
				},
			},
		},
		TypeID:     []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
		SortByName: true,
	}, items.ListFields{
		NameOnly:                   true,
		DescendantsParentsCount:    true,
		NewDescendantsParentsCount: true,
	}, items.OrderByNone, false)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	is := make([]*APITwinsBrandsListItem, len(twinsData))
	for idx, brand := range twinsData {
		is[idx] = &APITwinsBrandsListItem{
			Id:       brand.ID,
			Catname:  util.NullStringToString(brand.Catname),
			Name:     brand.NameOnly,
			Count:    brand.DescendantsParentsCount,
			NewCount: brand.NewDescendantsParentsCount,
		}
	}

	return &APITwinsBrandsList{
		Items: is,
	}, nil
}

func (s *ItemsGRPCServer) Item(ctx context.Context, in *ItemRequest) (*APIItem, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	fields := convertFields(in.GetFields())

	if (fields.InboxPicturesCount || fields.CommentsAttentionsCount) && !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	res, err := s.repository.Item(ctx, query.ItemsListOptions{
		ItemID:   in.GetId(),
		Language: in.GetLanguage(),
	}, fields)
	if err != nil {
		if errors.Is(err, items.ErrItemNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	return s.extractor.Extract(ctx, res, in.GetFields(), in.GetLanguage())
}

func (s *ItemsGRPCServer) List(ctx context.Context, in *ListItemsRequest) (*APIItemList, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	fields := convertFields(in.GetFields())
	if (fields.InboxPicturesCount || fields.CommentsAttentionsCount) && !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	inOptions := in.GetOptions()

	if (inOptions.GetExcludeSelfAndChilds() > 0 || inOptions.GetAutocomplete() != "" ||
		inOptions.GetSuggestionsTo() != 0) && !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	options := query.ItemsListOptions{
		Language: in.GetLanguage(),
		Limit:    in.GetLimit(),
		Page:     in.GetPage(),
	}

	err = mapItemListOptions(inOptions, &options)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	order := items.OrderByName

	switch in.GetOrder() {
	case ListItemsRequest_NAME_NAT:
		options.SortByName = true
	case ListItemsRequest_NAME, ListItemsRequest_DEFAULT:
		order = items.OrderByName
	case ListItemsRequest_CHILDS_COUNT:
		order = items.OrderByChildsCount
	}

	res, pages, err := s.repository.List(ctx, options, fields, order, true)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	is := make([]*APIItem, len(res))
	for idx, i := range res {
		is[idx], err = s.extractor.Extract(ctx, i, in.GetFields(), in.GetLanguage())
		if err != nil {
			return nil, err
		}
	}

	var paginator *Pages
	if pages != nil {
		paginator = &Pages{
			PageCount:        pages.PageCount,
			First:            pages.First,
			Last:             pages.Last,
			Current:          pages.Current,
			FirstPageInRange: pages.FirstPageInRange,
			LastPageInRange:  pages.LastPageInRange,
			PagesInRange:     pages.PagesInRange,
			TotalItemCount:   pages.TotalItemCount,
			Next:             pages.Next,
			Previous:         pages.Previous,
		}
	}

	return &APIItemList{
		Items:     is,
		Paginator: paginator,
	}, nil
}

func (s *ItemsGRPCServer) GetContentLanguages(_ context.Context, _ *emptypb.Empty) (*APIContentLanguages, error) {
	return &APIContentLanguages{
		Languages: s.contentLanguages,
	}, nil
}

func (s *ItemsGRPCServer) GetItemLink(ctx context.Context, in *APIItemLinkRequest) (*APIItemLink, error) {
	st := struct {
		ID     int64  `db:"id"`
		Name   string `db:"name"`
		URL    string `db:"url"`
		Type   string `db:"type"`
		ItemID int64  `db:"item_id"`
	}{}

	success, err := s.db.Select(
		schema.LinksTableIDCol, schema.LinksTableNameCol, schema.LinksTableURLCol, schema.LinksTableTypeCol,
		schema.LinksTableItemIDCol,
	).
		From(schema.LinksTable).
		Where(schema.LinksTableIDCol.Eq(in.GetId())).
		Executor().ScanStructContext(ctx, &st)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !success {
		return nil, status.Error(codes.NotFound, "item not found")
	}

	return &APIItemLink{
		Id:     st.ID,
		Name:   st.Name,
		Url:    st.URL,
		Type:   st.Type,
		ItemId: st.ItemID,
	}, nil
}

func (s *ItemsGRPCServer) GetItemLinks(ctx context.Context, in *APIGetItemLinksRequest) (*APIItemLinksResponse, error) {
	rows, err := s.db.Select(
		schema.LinksTableIDCol, schema.LinksTableNameCol, schema.LinksTableURLCol, schema.LinksTableTypeCol,
		schema.LinksTableItemIDCol,
	).
		From(schema.LinksTable).
		Where(schema.LinksTableItemIDCol.Eq(in.GetItemId())).
		Executor().QueryContext(ctx) //nolint:sqlclosecheck
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

	_, err = s.db.Delete(schema.LinksTable).
		Where(schema.LinksTableIDCol.Eq(in.GetId())).
		Executor().ExecContext(ctx)
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
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(InvalidParams) > 0 {
		return nil, wrapFieldViolations(InvalidParams)
	}

	res, err := s.db.Insert(schema.LinksTable).Rows(goqu.Record{
		schema.LinksTableNameColName:   in.GetName(),
		schema.LinksTableURLColName:    in.GetUrl(),
		schema.LinksTableTypeColName:   in.GetType(),
		schema.LinksTableItemIDColName: in.GetItemId(),
	}).Executor().ExecContext(ctx)
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

	_, err = s.db.Update(schema.LinksTable).
		Set(goqu.Record{
			schema.LinksTableNameColName:   in.GetName(),
			schema.LinksTableURLColName:    in.GetUrl(),
			schema.LinksTableTypeColName:   in.GetType(),
			schema.LinksTableItemIDColName: in.GetItemId(),
		}).
		Where(schema.LinksTableIDCol.Eq(in.GetId())).
		Executor().ExecContext(ctx)
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
			&validation.StringLength{Min: 0, Max: itemLinkNameMaxLength},
		},
	}

	s.Name, problems, err = nameInputFilter.IsValidString(s.GetName())
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
			&validation.StringLength{Min: 0, Max: itemLinkNameMaxLength},
		},
	}

	s.Url, problems, err = urlInputFilter.IsValidString(s.GetUrl())
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
			&validation.InArray{HaystackString: []string{
				"default",
				"official",
				"club",
				"helper",
			}},
		},
	}

	s.Type, problems, err = typeInputFilter.IsValidString(s.GetType())
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

	if in.GetItemId() == 0 && in.GetVehicleTypeId() == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	sqlSelect := s.db.Select(schema.VehicleVehicleTypeTableVehicleIDCol, schema.VehicleVehicleTypeTableVehicleTypeIDCol).
		From(schema.VehicleVehicleTypeTable).
		Where(schema.VehicleVehicleTypeTableInheritedCol.IsFalse())

	if in.GetItemId() != 0 {
		sqlSelect = sqlSelect.Where(schema.VehicleVehicleTypeTableVehicleIDCol.Eq(in.GetItemId()))
	}

	if in.GetVehicleTypeId() != 0 {
		sqlSelect = sqlSelect.Where(schema.VehicleVehicleTypeTableVehicleTypeIDCol.Eq(in.GetVehicleTypeId()))
	}

	rows, err := sqlSelect.Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	defer util.Close(rows)

	res := make([]*APIItemVehicleType, 0)

	for rows.Next() {
		var ivt APIItemVehicleType

		err = rows.Scan(&ivt.ItemId, &ivt.VehicleTypeId)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		res = append(res, &ivt)
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

	if in.GetItemId() == 0 && in.GetVehicleTypeId() == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	sqlSelect := s.db.Select(schema.VehicleVehicleTypeTableVehicleIDCol, schema.VehicleVehicleTypeTableVehicleTypeIDCol).
		From(schema.VehicleVehicleTypeTable).
		Where(
			schema.VehicleVehicleTypeTableInheritedCol.IsFalse(),
			schema.VehicleVehicleTypeTableVehicleIDCol.Eq(in.GetItemId()),
			schema.VehicleVehicleTypeTableVehicleTypeIDCol.Eq(in.GetVehicleTypeId()),
		)

	var ivt APIItemVehicleType

	rows, err := sqlSelect.Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	defer util.Close(rows)

	rows.Next()

	err = rows.Scan(&ivt.ItemId, &ivt.VehicleTypeId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &ivt, nil
}

func (s *ItemsGRPCServer) CreateItemVehicleType(ctx context.Context, in *APIItemVehicleType) (*emptypb.Empty, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "car", "move") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	count, err := s.repository.Count(ctx, query.ItemsListOptions{
		ItemID: in.GetItemId(),
		TypeID: []schema.ItemTableItemTypeID{
			schema.ItemTableItemTypeIDVehicle, schema.ItemTableItemTypeIDTwins,
		},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if count == 0 {
		return nil, status.Error(codes.NotFound, sql.ErrNoRows.Error())
	}

	err = s.repository.AddItemVehicleType(ctx, in.GetItemId(), in.GetVehicleTypeId())
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

	err = s.repository.RemoveItemVehicleType(ctx, in.GetItemId(), in.GetVehicleTypeId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ItemsGRPCServer) GetItemLanguages(
	ctx context.Context, in *APIGetItemLanguagesRequest,
) (*ItemLanguages, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	rows, err := s.repository.LanguageList(ctx, in.GetItemId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*ItemLanguage, len(rows))

	for idx, row := range rows {
		text := ""
		if row.TextID > 0 {
			text, err = s.textstorageRepository.Text(ctx, row.TextID)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}

		fullText := ""
		if row.FullTextID > 0 {
			fullText, err = s.textstorageRepository.Text(ctx, row.FullTextID)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}

		result[idx] = &ItemLanguage{
			ItemId:     row.ItemID,
			Name:       row.Name,
			Language:   row.Language,
			TextId:     row.TextID,
			Text:       text,
			FullTextId: row.FullTextID,
			FullText:   fullText,
		}
	}

	return &ItemLanguages{
		Items: result,
	}, nil
}

func (s *ItemsGRPCServer) UpdateItemLanguage(ctx context.Context, in *ItemLanguage) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	InvalidParams, err := in.Validate()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(InvalidParams) > 0 {
		return nil, wrapFieldViolations(InvalidParams)
	}

	itemID := in.GetItemId()

	item, err := s.repository.Item(
		ctx, query.ItemsListOptions{ItemID: itemID, Language: EventsDefaultLanguage},
		items.ListFields{NameText: true},
	)
	if err != nil {
		if errors.Is(err, items.ErrItemNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	changes, err := s.repository.UpdateItemLanguage(
		ctx, itemID, in.GetLanguage(), in.GetName(), in.GetText(), in.GetFullText(), userID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(changes) > 0 {
		err := s.repository.UserItemSubscribe(ctx, userID, in.GetItemId())
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		author, err := s.usersRepository.User(ctx, query.UserListOptions{ID: userID}, users.UserFields{})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		language := in.GetLanguage()

		falseRef := false

		subscribers, _, err := s.usersRepository.Users(ctx, query.UserListOptions{
			Deleted: &falseRef,
			ItemSubscribe: &query.UserItemSubscribeListOptions{
				ItemIDs: []int64{itemID},
			},
			ExcludeIDs: []int64{userID},
		}, users.UserFields{})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		for _, subscriber := range subscribers {
			uri, err := s.hostManager.URIByLanguage(subscriber.Language)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			itemNameText, err := s.formatItemNameText(item, subscriber.Language)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			localizer := s.i18n.Localizer(subscriber.Language)

			changesStrs := make([]string, 0, len(changes))

			for _, field := range changes {
				changesStr, err := localizer.Localize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID: field,
					},
				})
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}

				changesStrs = append(changesStrs, changesStr+" ("+language+")")
			}

			err = s.messagingRepository.CreateMessageFromTemplate(
				ctx, 0, subscriber.ID, "pm/user-%s-edited-item-language-%s-%s",
				map[string]interface{}{
					"UserURL":      frontend.UserURL(uri, author.ID, author.Identity),
					"ItemName":     itemNameText,
					"ItemModerURL": frontend.ItemModerURL(uri, itemID),
					"Changes":      strings.Join(changesStrs, "\n"),
				},
				subscriber.Language,
			)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}

		itemNameText, err := s.formatItemNameText(item, EventsDefaultLanguage)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.events.Add(ctx, Event{
			UserID:  userID,
			Message: "Редактирование языковых названия, описания и полного описания автомобиля " + itemNameText,
			Items:   []int64{itemID},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *ItemsGRPCServer) GetItemParentLanguages(
	ctx context.Context, in *APIGetItemParentLanguagesRequest,
) (*ItemParentLanguages, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	rows, err := s.repository.ParentLanguageList(ctx, in.GetItemId(), in.GetParentId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*ItemParentLanguage, len(rows))
	for idx, row := range rows {
		result[idx] = &ItemParentLanguage{
			ItemId:   row.ItemID,
			ParentId: row.ParentID,
			Name:     row.Name,
			Language: row.Language,
		}
	}

	return &ItemParentLanguages{
		Items: result,
	}, nil
}

func (s *ItemsGRPCServer) GetStats(ctx context.Context, _ *emptypb.Empty) (*StatsResponse, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	totalBrands, err := s.repository.Count(ctx, query.ItemsListOptions{
		TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	totalCars, err := s.repository.Count(ctx, query.ItemsListOptions{
		TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDVehicle},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	totalCarAttrs, err := s.attrsRepository.TotalZoneAttrs(ctx, 1)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	carAttrsValues, err := s.attrsRepository.TotalValues(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	carsWith4OrMorePictures, err := s.repository.ItemsWithPicturesCount(ctx, typicalPicturesInList)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	itemsWithBeginYear, err := s.repository.Count(ctx, query.ItemsListOptions{
		HasBeginYear: true,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	itemsWithBeginAndEndYears, err := s.repository.Count(ctx, query.ItemsListOptions{
		HasBeginYear: true,
		HasEndYear:   true,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	itemsWithBeginAndEndYearsAndMonths, err := s.repository.Count(ctx, query.ItemsListOptions{
		HasBeginYear:  true,
		HasEndYear:    true,
		HasBeginMonth: true,
		HasEndMonth:   true,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	brandsWithLogo, err := s.repository.Count(ctx, query.ItemsListOptions{
		HasLogo: true,
		TypeID:  []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	totalPictures, err := s.picturesRepository.Count(ctx, &query.PictureListOptions{})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	picturesWithCopyrights, err := s.picturesRepository.Count(ctx, &query.PictureListOptions{
		HasCopyrights: true,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &StatsResponse{
		Values: []*StatsValue{
			{
				Name:  "moder/statistics/photos-with-copyrights",
				Total: int32(totalPictures),          //nolint: gosec
				Value: int32(picturesWithCopyrights), //nolint: gosec
			},
			{
				Name:  "moder/statistics/vehicles-with-4-or-more-photos",
				Total: int32(totalCars), //nolint: gosec
				Value: carsWith4OrMorePictures,
			},
			{
				Name:  "moder/statistics/specifications-values",
				Total: int32(totalCars) * totalCarAttrs, //nolint: gosec
				Value: carAttrsValues,
			},
			{
				Name:  "moder/statistics/brand-logos",
				Total: int32(totalBrands),    //nolint: gosec
				Value: int32(brandsWithLogo), //nolint: gosec
			},
			{
				Name:  "moder/statistics/from-years",
				Total: int32(totalCars),          //nolint: gosec
				Value: int32(itemsWithBeginYear), //nolint: gosec
			},
			{
				Name:  "moder/statistics/from-and-to-years",
				Total: int32(totalCars),                 //nolint: gosec
				Value: int32(itemsWithBeginAndEndYears), //nolint: gosec
			},
			{
				Name:  "moder/statistics/from-and-to-years-and-months",
				Total: int32(totalCars),                          //nolint: gosec
				Value: int32(itemsWithBeginAndEndYearsAndMonths), //nolint: gosec
			},
		},
	}, nil
}

func (s *ItemLanguage) Validate() ([]*errdetails.BadRequest_FieldViolation, error) {
	var (
		result   = make([]*errdetails.BadRequest_FieldViolation, 0)
		problems []string
		err      error
	)

	nameInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}, &validation.StringSingleSpaces{}},
		Validators: []validation.ValidatorInterface{
			&validation.StringLength{Min: 0, Max: schema.ItemLanguageNameMaxLength},
		},
	}

	s.Name, problems, err = nameInputFilter.IsValidString(s.GetName())
	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "name",
			Description: fv,
		})
	}

	textInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.StringLength{Min: 0, Max: items.ItemLanguageTextMaxLength},
		},
	}

	s.Text, problems, err = textInputFilter.IsValidString(s.GetText())
	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "text",
			Description: fv,
		})
	}

	fullTextInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}},
		Validators: []validation.ValidatorInterface{
			&validation.StringLength{Min: 0, Max: items.ItemLanguageFullTextMaxLength},
		},
	}

	s.FullText, problems, err = fullTextInputFilter.IsValidString(s.GetFullText())
	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "full_text",
			Description: fv,
		})
	}

	return result, nil
}

func (s *ItemParentLanguage) Validate() ([]*errdetails.BadRequest_FieldViolation, error) {
	var (
		result   = make([]*errdetails.BadRequest_FieldViolation, 0)
		problems []string
		err      error
	)

	nameInputFilter := validation.InputFilter{
		Filters: []validation.FilterInterface{&validation.StringTrimFilter{}, &validation.StringSingleSpaces{}},
		Validators: []validation.ValidatorInterface{
			&validation.StringLength{Min: 0, Max: schema.ItemLanguageNameMaxLength},
		},
	}

	s.Name, problems, err = nameInputFilter.IsValidString(s.GetName())
	if err != nil {
		return nil, err
	}

	for _, fv := range problems {
		result = append(result, &errdetails.BadRequest_FieldViolation{
			Field:       "name",
			Description: fv,
		})
	}

	return result, nil
}

func (s *ItemsGRPCServer) SetItemParentLanguage(ctx context.Context, in *ItemParentLanguage) (*emptypb.Empty, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "global", "moderate") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	InvalidParams, err := in.Validate()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(InvalidParams) > 0 {
		return nil, wrapFieldViolations(InvalidParams)
	}

	err = s.repository.SetItemParentLanguage(ctx, in.GetParentId(), in.GetItemId(), in.GetLanguage(), in.GetName(), false)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ItemsGRPCServer) GetBrandNewItems(ctx context.Context, in *NewItemsRequest) (*NewItemsResponse, error) {
	const (
		newItemsLimit = 30
		daysLimit     = 7
	)

	lang := in.GetLanguage()

	brand, err := s.repository.Item(ctx, query.ItemsListOptions{
		TypeID:   []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
		ItemID:   in.GetItemId(),
		Language: lang,
	}, items.ListFields{Logo: true})
	if err != nil {
		if errors.Is(err, items.ErrItemNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	extractedBrand, err := s.extractor.Extract(ctx, brand, &ItemFields{Brandicon: true}, lang)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	carList, _, err := s.repository.List(ctx, query.ItemsListOptions{
		Language: lang,
		ItemParentCacheAncestor: &query.ItemParentCacheListOptions{
			ItemsByParentID: &query.ItemsListOptions{
				Language: lang,
				ItemID:   brand.ID,
			},
		},
		CreatedInDays: daysLimit,
		Limit:         newItemsLimit,
	}, items.ListFields{
		NameHTML: true,
	}, items.OrderByAddDatetime, false)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	extractedItems := make([]*APIItem, 0, len(carList))

	for _, car := range carList {
		extractedItem, err := s.extractor.Extract(ctx, car, &ItemFields{NameHtml: true}, lang)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		extractedItems = append(extractedItems, extractedItem)
	}

	return &NewItemsResponse{
		Brand: extractedBrand,
		Items: extractedItems,
	}, nil
}

func (s *ItemsGRPCServer) GetNewItems(ctx context.Context, in *NewItemsRequest) (*NewItemsResponse, error) {
	const (
		newItemsLimit = 20
		daysLimit     = 7
	)

	lang := in.GetLanguage()

	category, err := s.repository.Item(ctx, query.ItemsListOptions{
		TypeID:   []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDCategory},
		ItemID:   in.GetItemId(),
		Language: lang,
	}, items.ListFields{})
	if err != nil {
		if errors.Is(err, items.ErrItemNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	extractedBrand, err := s.extractor.Extract(ctx, category, nil, lang)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	carList, _, err := s.repository.List(ctx, query.ItemsListOptions{
		Language: lang,
		TypeID:   []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDVehicle, schema.ItemTableItemTypeIDEngine},
		ItemParentParent: &query.ItemParentListOptions{
			LinkedInDays: daysLimit,
			ItemParentCacheAncestorByParentID: &query.ItemParentCacheListOptions{
				ItemsByParentID: &query.ItemsListOptions{
					TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDCategory, schema.ItemTableItemTypeIDFactory},
					ItemID: category.ID,
				},
			},
		},
		Limit: newItemsLimit,
	}, items.ListFields{
		NameHTML: true,
	}, items.OrderByItemParentParentTimestamp, false)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	extractedItems := make([]*APIItem, 0, len(carList))

	for _, car := range carList {
		extractedItem, err := s.extractor.Extract(ctx, car, &ItemFields{NameHtml: true}, lang)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		extractedItems = append(extractedItems, extractedItem)
	}

	return &NewItemsResponse{
		Brand: extractedBrand,
		Items: extractedItems,
	}, nil
}

func (s *ItemsGRPCServer) formatItemNameText(row items.Item, lang string) (string, error) {
	nameFormatter := items.NewItemNameFormatter(s.i18n)

	return nameFormatter.FormatText(items.ItemNameFormatterOptions{
		BeginModelYear:         util.NullInt32ToScalar(row.BeginModelYear),
		EndModelYear:           util.NullInt32ToScalar(row.EndModelYear),
		BeginModelYearFraction: util.NullStringToString(row.BeginModelYearFraction),
		EndModelYearFraction:   util.NullStringToString(row.EndModelYearFraction),
		Spec:                   row.SpecShortName,
		SpecFull:               row.SpecName,
		Body:                   row.Body,
		Name:                   row.NameOnly,
		BeginYear:              util.NullInt32ToScalar(row.BeginYear),
		EndYear:                util.NullInt32ToScalar(row.EndYear),
		Today:                  util.NullBoolToBoolPtr(row.Today),
		BeginMonth:             util.NullInt16ToScalar(row.BeginMonth),
		EndMonth:               util.NullInt16ToScalar(row.EndMonth),
	}, lang)
}

func (s *ItemsGRPCServer) CreateItemParent(ctx context.Context, in *ItemParent) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "car", "move") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	InvalidParams, err := in.Validate()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(InvalidParams) > 0 {
		return nil, wrapFieldViolations(InvalidParams)
	}

	item, err := s.repository.Item(
		ctx, query.ItemsListOptions{ItemID: in.GetItemId(), Language: EventsDefaultLanguage},
		items.ListFields{NameText: true},
	)
	if err != nil {
		if errors.Is(err, items.ErrItemNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	parentItem, err := s.repository.Item(
		ctx, query.ItemsListOptions{ItemID: in.GetParentId(), Language: EventsDefaultLanguage},
		items.ListFields{NameText: true},
	)
	if err != nil {
		if errors.Is(err, items.ErrItemNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	_, err = s.repository.CreateItemParent(
		ctx, item.ID, parentItem.ID, reverseConvertItemParentType(in.GetType()), in.GetCatname(),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.repository.UpdateInheritance(ctx, item.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.repository.RefreshItemVehicleTypeInheritanceFromParents(ctx, item.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.attrsRepository.UpdateActualValues(ctx, item.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	itemNameText, err := s.formatItemNameText(item, "en")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	parentNameText, err := s.formatItemNameText(parentItem, "en")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.events.Add(ctx, Event{
		UserID: userID,
		Message: fmt.Sprintf(
			"%s выбран как родительский для %s",
			parentNameText,
			itemNameText,
		),
		Items: []int64{item.ID, parentItem.ID},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.notifyItemParentSubscribers(ctx, item, parentItem, userID, "pm/user-%s-adds-item-%s-%s-to-item-%s-%s")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ItemsGRPCServer) UpdateItemParent(ctx context.Context, in *ItemParent) (*emptypb.Empty, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "car", "move") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	InvalidParams, err := in.Validate()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(InvalidParams) > 0 {
		return nil, wrapFieldViolations(InvalidParams)
	}

	_, err = s.repository.UpdateItemParent(
		ctx, in.GetItemId(), in.GetParentId(), reverseConvertItemParentType(in.GetType()), in.GetCatname(), false,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ItemsGRPCServer) DeleteItemParent(ctx context.Context, in *DeleteItemParentRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "car", "move") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	item, err := s.repository.Item(
		ctx, query.ItemsListOptions{ItemID: in.GetItemId(), Language: EventsDefaultLanguage},
		items.ListFields{NameText: true},
	)
	if err != nil {
		if errors.Is(err, items.ErrItemNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	parent, err := s.repository.Item(
		ctx, query.ItemsListOptions{ItemID: in.GetParentId(), Language: EventsDefaultLanguage},
		items.ListFields{NameText: true},
	)
	if err != nil {
		if errors.Is(err, items.ErrItemNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.repository.RemoveItemParent(ctx, item.ID, parent.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.repository.UpdateInheritance(ctx, item.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.repository.RefreshItemVehicleTypeInheritanceFromParents(ctx, item.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.attrsRepository.UpdateActualValues(ctx, item.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	itemNameText, err := s.formatItemNameText(item, "en")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	parentNameText, err := s.formatItemNameText(parent, "en")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.events.Add(ctx, Event{
		UserID: userID,
		Message: fmt.Sprintf(
			"%s перестал быть родительским автомобилем для %s",
			parentNameText,
			itemNameText,
		),
		Items: []int64{item.ID, parent.ID},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.notifyItemParentSubscribers(ctx, item, parent, userID, "pm/user-%s-removed-item-%s-%s-from-item-%s-%s")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ItemsGRPCServer) notifyItemParentSubscribers(
	ctx context.Context, item, parent items.Item, userID int64, messageID string,
) error {
	falseRef := false

	subscribers, _, err := s.usersRepository.Users(ctx, query.UserListOptions{
		Deleted: &falseRef,
		ItemSubscribe: &query.UserItemSubscribeListOptions{
			ItemIDs: []int64{item.ID, parent.ID},
		},
		ExcludeIDs: []int64{userID},
	}, users.UserFields{})
	if err != nil {
		return err
	}

	author, err := s.usersRepository.User(ctx, query.UserListOptions{ID: userID}, users.UserFields{})
	if err != nil {
		return err
	}

	for _, subscriber := range subscribers {
		uri, err := s.hostManager.URIByLanguage(subscriber.Language)
		if err != nil {
			return err
		}

		itemNameText, err := s.formatItemNameText(item, subscriber.Language)
		if err != nil {
			return err
		}

		parentNameText, err := s.formatItemNameText(parent, subscriber.Language)
		if err != nil {
			return err
		}

		err = s.messagingRepository.CreateMessageFromTemplate(ctx, 0, subscriber.ID, messageID, map[string]interface{}{
			"UserURL":            frontend.UserURL(uri, author.ID, author.Identity),
			"ItemName":           itemNameText,
			"ItemModerURL":       frontend.ItemModerURL(uri, item.ID),
			"ParentItemName":     parentNameText,
			"ParentItemModerURL": frontend.ItemModerURL(uri, parent.ID),
		}, subscriber.Language)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ItemsGRPCServer) MoveItemParent(ctx context.Context, in *MoveItemParentRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "car", "move") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	success, err := s.repository.MoveItemParent(ctx, in.GetItemId(), in.GetParentId(), in.GetDestParentId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if success {
		item, err := s.repository.Item(
			ctx, query.ItemsListOptions{ItemID: in.GetItemId(), Language: EventsDefaultLanguage},
			items.ListFields{NameText: true},
		)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		itemNameText, err := s.formatItemNameText(item, "en")
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		oldParent, err := s.repository.Item(
			ctx, query.ItemsListOptions{ItemID: in.GetParentId(), Language: EventsDefaultLanguage},
			items.ListFields{NameText: true},
		)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		oldParentNameText, err := s.formatItemNameText(oldParent, "en")
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		newParent, err := s.repository.Item(
			ctx, query.ItemsListOptions{ItemID: in.GetDestParentId(), Language: EventsDefaultLanguage},
			items.ListFields{NameText: true},
		)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		newParentNameText, err := s.formatItemNameText(newParent, "en")
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.events.Add(ctx, Event{
			UserID: userID,
			Message: fmt.Sprintf(
				"%s перемещен из %s в %s",
				oldParentNameText,
				itemNameText,
				newParentNameText,
			),
			Items: []int64{item.ID, oldParent.ID, newParent.ID},
		})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.repository.UpdateInheritance(ctx, item.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		err = s.attrsRepository.UpdateActualValues(ctx, item.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *ItemsGRPCServer) RefreshInheritance(
	ctx context.Context, in *RefreshInheritanceRequest,
) (*emptypb.Empty, error) {
	_, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "specifications", "admin") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	err = s.repository.UpdateInheritance(ctx, in.GetItemId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.attrsRepository.UpdateActualValues(ctx, in.GetItemId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ItemsGRPCServer) SetUserItemSubscription(
	ctx context.Context, in *SetUserItemSubscriptionRequest,
) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "car", "edit_meta") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	if in.GetSubscribed() {
		err = s.repository.UserItemSubscribe(ctx, in.GetItemId(), userID)
	} else {
		err = s.repository.UserItemUnsubscribe(ctx, in.GetItemId(), userID)
	}

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ItemsGRPCServer) SetItemEngine(ctx context.Context, in *SetItemEngineRequest) (*emptypb.Empty, error) {
	userID, role, err := s.auth.ValidateGRPC(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !s.enforcer.Enforce(role, "car", "edit_meta") ||
		!s.enforcer.Enforce(role, "specifications", "edit-engine") ||
		!s.enforcer.Enforce(role, "specifications", "edit") {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	itemID := in.GetItemId()

	item, err := s.repository.Item(
		ctx, query.ItemsListOptions{ItemID: itemID, Language: EventsDefaultLanguage},
		items.ListFields{NameText: true},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	changed, err := s.repository.SetItemEngine(ctx, itemID, in.GetEngineItemId(), in.GetEngineInherited())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if changed {
		user, err := s.usersRepository.User(ctx, query.UserListOptions{ID: userID}, users.UserFields{})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		eventItemIDs := []int64{itemID}
		if item.EngineItemID.Valid {
			eventItemIDs = append(eventItemIDs, item.EngineItemID.Int64)
		}

		itemNameText, err := s.formatItemNameText(item, EventsDefaultLanguage)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		switch {
		case in.GetEngineInherited():
			err = s.events.Add(ctx, Event{
				UserID: userID,
				Message: fmt.Sprintf(
					"У автомобиля %s установлено наследование двигателя",
					itemNameText,
				),
				Items: eventItemIDs,
			})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			err = s.notifyItemEngineInherited(ctx, user, item)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		case in.GetEngineItemId() == 0:
			if item.EngineItemID.Valid {
				oldEngine, err := s.repository.Item(
					ctx, query.ItemsListOptions{ItemID: item.EngineItemID.Int64, Language: EventsDefaultLanguage},
					items.ListFields{NameText: true},
				)
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}

				oldEngineNameText, err := s.formatItemNameText(oldEngine, EventsDefaultLanguage)
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}

				err = s.events.Add(ctx, Event{
					UserID: userID,
					Message: fmt.Sprintf(
						"У автомобиля %s убран двигатель (был %s)",
						itemNameText,
						oldEngineNameText,
					),
					Items: append(eventItemIDs, item.EngineItemID.Int64),
				})
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}

				err = s.notifyItemEngineCleared(ctx, user, item)
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}
			}
		default:
			newEngine, err := s.repository.Item(ctx, query.ItemsListOptions{
				ItemID: in.GetEngineItemId(),
				TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDEngine},
			}, items.ListFields{NameText: true})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			newEngineNameText, err := s.formatItemNameText(newEngine, EventsDefaultLanguage)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			err = s.events.Add(ctx, Event{
				UserID: userID,
				Message: fmt.Sprintf(
					"Автомобилю %s назначен двигатель %s",
					itemNameText,
					newEngineNameText,
				),
				Items: []int64{itemID, newEngine.ID},
			})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			err = s.notifyItemEngineUpdated(ctx, user, item, in.GetEngineItemId())
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}

		err = s.attrsRepository.UpdateActualValues(ctx, item.ID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	err = s.repository.UserItemSubscribe(ctx, itemID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *ItemsGRPCServer) notifyItemEngineInherited(ctx context.Context, user *schema.UsersRow, item items.Item) error {
	var oldEngineID int64
	if item.EngineItemID.Valid {
		oldEngineID = item.EngineItemID.Int64
	}

	return s.notifyItemSubscribers(
		ctx, []int64{item.ID, oldEngineID}, user.ID, "pm/user-%s-set-inherited-vehicle-engine-%s-%s",
		func(uri *url.URL, language string) (map[string]interface{}, error) {
			itemNameText, err := s.formatItemNameText(item, language)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"UserURL":      frontend.UserURL(uri, user.ID, user.Identity),
				"ItemName":     itemNameText,
				"ItemModerURL": frontend.ItemModerURL(uri, item.ID),
			}, nil
		},
	)
}

func (s *ItemsGRPCServer) notifyItemEngineCleared(ctx context.Context, user *schema.UsersRow, item items.Item) error {
	var oldEngineID int64
	if item.EngineItemID.Valid {
		oldEngineID = item.EngineItemID.Int64
	}

	return s.notifyItemSubscribers(
		ctx, []int64{item.ID, oldEngineID}, user.ID, "pm/user-%s-canceled-vehicle-engine-%s-%s-%s",
		func(uri *url.URL, language string) (map[string]interface{}, error) {
			itemNameText, err := s.formatItemNameText(item, language)
			if err != nil {
				return nil, err
			}

			oldEngine, err := s.repository.Item(
				ctx, query.ItemsListOptions{ItemID: oldEngineID, Language: language},
				items.ListFields{NameText: true},
			)
			if err != nil {
				return nil, err
			}

			oldEngineNameText, err := s.formatItemNameText(oldEngine, language)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"UserURL":      frontend.UserURL(uri, user.ID, user.Identity),
				"EngineName":   oldEngineNameText,
				"ItemName":     itemNameText,
				"ItemModerURL": frontend.ItemModerURL(uri, item.ID),
			}, nil
		},
	)
}

func (s *ItemsGRPCServer) notifyItemEngineUpdated(
	ctx context.Context, user *schema.UsersRow, item items.Item, newEngineID int64,
) error {
	var oldEngineID int64
	if item.EngineItemID.Valid {
		oldEngineID = item.EngineItemID.Int64
	}

	return s.notifyItemSubscribers(
		ctx, []int64{item.ID, oldEngineID, newEngineID}, user.ID, "pm/user-%s-set-vehicle-engine-%s-%s-%s",
		func(uri *url.URL, language string) (map[string]interface{}, error) {
			itemNameText, err := s.formatItemNameText(item, language)
			if err != nil {
				return nil, err
			}

			newEngine, err := s.repository.Item(
				ctx, query.ItemsListOptions{ItemID: newEngineID, Language: language},
				items.ListFields{NameText: true},
			)
			if err != nil {
				return nil, err
			}

			newEngineNameText, err := s.formatItemNameText(newEngine, language)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"UserURL":      frontend.UserURL(uri, user.ID, user.Identity),
				"EngineName":   newEngineNameText,
				"ItemName":     itemNameText,
				"ItemModerURL": frontend.ItemModerURL(uri, item.ID),
			}, nil
		},
	)
}

func (s *ItemsGRPCServer) notifyItemSubscribers(
	ctx context.Context, itemIDs []int64, excludeUserID int64, messageID string,
	templateData func(*url.URL, string) (map[string]interface{}, error),
) error {
	falseRef := false

	subscribers, _, err := s.usersRepository.Users(ctx, query.UserListOptions{
		Deleted: &falseRef,
		ItemSubscribe: &query.UserItemSubscribeListOptions{
			ItemIDs: itemIDs,
		},
		ExcludeIDs: []int64{excludeUserID},
	}, users.UserFields{})
	if err != nil {
		return err
	}

	for _, subscriber := range subscribers {
		uri, err := s.hostManager.URIByLanguage(subscriber.Language)
		if err != nil {
			return err
		}

		data, err := templateData(uri, subscriber.Language)
		if err != nil {
			return err
		}

		err = s.messagingRepository.CreateMessageFromTemplate(ctx, 0, subscriber.ID, messageID, data, subscriber.Language)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ItemsGRPCServer) GetBrandSections(
	ctx context.Context, in *GetBrandSectionsRequest,
) (*APIBrandSections, error) {
	item, err := s.repository.Item(ctx, query.ItemsListOptions{
		ItemID: in.GetItemId(),
		TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
	}, items.ListFields{})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	sections, err := s.brandSections(ctx, in.GetLanguage(), item.ID, item.Catname.String)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &APIBrandSections{
		Sections: sections,
	}, nil
}

func (s *ItemsGRPCServer) brandSections(
	ctx context.Context, lang string, brandID int64, brandCatname string,
) ([]*APIBrandSection, error) {
	// create groups array
	sections, err := s.carSections(ctx, lang, brandID, brandCatname)
	if err != nil {
		return nil, fmt.Errorf("carSections(): %w", err)
	}

	otherGroups, err := s.otherGroups(ctx, brandID, brandCatname, lang)
	if err != nil {
		return nil, fmt.Errorf("otherGroups(): %w", err)
	}

	return append(
		sections,
		&APIBrandSection{
			Name:       "Other",
			RouterLink: nil,
			Groups:     otherGroups,
		},
	), nil
}

func (s *ItemsGRPCServer) otherGroups(
	ctx context.Context, brandID int64, brandCatname string, lang string,
) ([]*APIBrandSection, error) {
	var groups []*APIBrandSection

	// concepts
	hasConcepts, err := s.repository.Exists(ctx, query.ItemsListOptions{
		ItemParentCacheAncestor: &query.ItemParentCacheListOptions{
			ParentID: brandID,
		},
		IsConcept: true,
	})
	if err != nil {
		return nil, err
	}

	localizer := s.i18n.Localizer(lang)

	if hasConcepts {
		translated, err := localizer.Localize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID: "concepts and prototypes",
			},
		})
		if err != nil {
			return nil, err
		}

		groups = append(groups, &APIBrandSection{
			RouterLink: []string{"/", brandCatname, "concepts"},
			Name:       translated,
		})
	}

	groupTypes := []struct {
		PerspectiveID        int32
		ExcludePerspectiveID []int32
		Catname              string
		Name                 string
	}{
		{
			PerspectiveID: items.PerspectiveIDLogo,
			Catname:       "logotypes",
			Name:          "logotypes",
		},
		{
			PerspectiveID: items.PerspectiveIDMixed,
			Catname:       "mixed",
			Name:          "mixed",
		},
		{
			ExcludePerspectiveID: []int32{items.PerspectiveIDLogo, items.PerspectiveIDMixed},
			Catname:              "other",
			Name:                 "unsorted",
		},
	}

	for _, groupType := range groupTypes {
		picturesCount, err := s.picturesRepository.Count(ctx, &query.PictureListOptions{
			Status: schema.PictureStatusAccepted,
			PictureItem: &query.PictureItemListOptions{
				ItemID:               brandID,
				PerspectiveID:        groupType.PerspectiveID,
				ExcludePerspectiveID: groupType.ExcludePerspectiveID,
			},
		})
		if err != nil {
			return nil, err
		}

		if picturesCount > 0 {
			translated, err := localizer.Localize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID: groupType.Name,
				},
			})
			if err != nil {
				return nil, err
			}

			groups = append(groups, &APIBrandSection{
				RouterLink: []string{"/", brandCatname, groupType.Catname},
				Name:       translated,
				Count:      int32(picturesCount), //nolint: gosec
			})
		}
	}

	return groups, nil
}

type SectionPreset struct {
	Name       string
	CarTypeID  int64
	ItemTypeID []schema.ItemTableItemTypeID
	RouterLink []string
}

func (s *ItemsGRPCServer) carSections(
	ctx context.Context, lang string, brandID int64, brandCatname string,
) ([]*APIBrandSection, error) {
	sectionsPresets := []SectionPreset{
		{
			ItemTypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDVehicle, schema.ItemTableItemTypeIDBrand},
		},
		{
			Name:       "catalogue/section/moto",
			CarTypeID:  items.VehicleTypeIDMoto,
			ItemTypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDVehicle, schema.ItemTableItemTypeIDBrand},
		},
		{
			Name:       "catalogue/section/buses",
			CarTypeID:  items.VehicleTypeIDBus,
			ItemTypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDVehicle, schema.ItemTableItemTypeIDBrand},
		},
		{
			Name:       "catalogue/section/trucks",
			CarTypeID:  items.VehicleTypeIDTruck,
			ItemTypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDVehicle, schema.ItemTableItemTypeIDBrand},
		},
		{
			Name:       "catalogue/section/tractors",
			CarTypeID:  items.VehicleTypeIDTractor,
			ItemTypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDVehicle, schema.ItemTableItemTypeIDBrand},
		},
		{
			Name:       "catalogue/section/engines",
			ItemTypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDEngine},
			RouterLink: []string{"/", brandCatname, "engines"},
		},
	}

	sections := make([]*APIBrandSection, 0, len(sectionsPresets))

	for _, sectionsPreset := range sectionsPresets {
		sectionGroups, err := s.carSectionGroups(
			ctx,
			lang,
			brandID,
			brandCatname,
			sectionsPreset,
		)
		if err != nil {
			return nil, fmt.Errorf("carSectionGroups(): %w", err)
		}

		slices.SortFunc(sectionGroups, func(a, b *APIBrandSection) int {
			return strings.Compare(a.GetName(), b.GetName())
		})

		sections = append(sections, &APIBrandSection{
			Name:       sectionsPreset.Name,
			RouterLink: sectionsPreset.RouterLink,
			Groups:     sectionGroups,
		})
	}

	return sections, nil
}

func (s *ItemsGRPCServer) carSectionGroups(
	ctx context.Context,
	lang string,
	brandID int64,
	brandCatname string,
	section SectionPreset,
) ([]*APIBrandSection, error) {
	var (
		err  error
		rows []items.ItemParent
	)

	if section.CarTypeID > 0 {
		rows, _, err = s.repository.ItemParents(ctx, query.ItemParentListOptions{
			ParentID: brandID,
			ChildItems: &query.ItemsListOptions{
				TypeID:                section.ItemTypeID,
				IsNotConcept:          true,
				VehicleTypeAncestorID: section.CarTypeID,
			},
			Language: lang,
		}, items.ItemParentFields{Name: true}, items.ItemParentOrderByAuto)
		if err != nil {
			return nil, fmt.Errorf("ItemParents(): %w", err)
		}
	} else {
		rows, _, err = s.repository.ItemParents(ctx, query.ItemParentListOptions{
			ParentID: brandID,
			ChildItems: &query.ItemsListOptions{
				TypeID:       section.ItemTypeID,
				IsNotConcept: true,
				ExcludeVehicleTypeAncestorID: []int64{
					items.VehicleTypeIDMoto, items.VehicleTypeIDTractor, items.VehicleTypeIDTruck, items.VehicleTypeIDBus,
				},
			},
			Language: lang,
		}, items.ItemParentFields{Name: true}, items.ItemParentOrderByAuto)
		if err != nil {
			return nil, fmt.Errorf("ItemParents(): %w", err)
		}

		rows2, _, err := s.repository.ItemParents(ctx, query.ItemParentListOptions{
			ParentID: brandID,
			ChildItems: &query.ItemsListOptions{
				TypeID:            section.ItemTypeID,
				IsNotConcept:      true,
				VehicleTypeIsNull: true,
			},
			Language: lang,
		}, items.ItemParentFields{Name: true}, items.ItemParentOrderByAuto)
		if err != nil {
			return nil, fmt.Errorf("ItemParents(): %w", err)
		}

		rows = append(rows, rows2...)
	}

	groups := make([]*APIBrandSection, 0, len(rows))

	for _, row := range rows {
		groups = append(groups, &APIBrandSection{
			RouterLink: []string{"/", brandCatname, row.Catname},
			Name:       row.Name,
		})
	}

	return groups, nil
}

func (s *ItemsGRPCServer) prefetchItems(
	ctx context.Context, ids []int64, lang string, fields items.ListFields,
) (map[int64]*items.Item, error) {
	itemRows, _, err := s.repository.List(ctx, query.ItemsListOptions{
		ItemIDs:  ids,
		Language: lang,
	}, fields, items.OrderByNone, false)
	if err != nil {
		return nil, err
	}

	itemsMap := make(map[int64]*items.Item, len(itemRows))

	for _, itemRow := range itemRows {
		itemsMap[itemRow.ID] = &itemRow
	}

	return itemsMap, nil
}

func (s *ItemsGRPCServer) GetItemParents(
	ctx context.Context, in *GetItemParentsRequest,
) (*GetItemParentsResponse, error) {
	inOptions := in.GetOptions()

	if inOptions.GetItemId() == 0 && inOptions.GetParentId() == 0 {
		return nil, status.Error(codes.PermissionDenied, "PermissionDenied")
	}

	options := query.ItemParentListOptions{
		Limit: in.GetLimit(),
		Page:  in.GetPage(),
	}

	err := mapItemParentListOptions(inOptions, &options)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	order := items.ItemParentOrderByNone

	switch in.GetOrder() {
	case GetItemParentsRequest_NONE:
	case GetItemParentsRequest_CATEGORIES_FIRST:
		order = items.ItemParentOrderByCategoriesFirst
	case GetItemParentsRequest_AUTO:
		order = items.ItemParentOrderByAuto
	}

	fields := convertItemParentFields(in.GetFields())

	rows, pages, err := s.repository.ItemParents(ctx, options, fields, order)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	itemFields := in.GetFields().GetItem()
	itemsMap := make(map[int64]*items.Item, 0)

	if itemFields != nil && len(rows) > 0 {
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ItemID)
		}

		itemsMap, err = s.prefetchItems(ctx, ids, in.GetLanguage(), convertFields(itemFields))
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	parentFields := in.GetFields().GetParent()
	parentsMap := make(map[int64]*items.Item, len(rows))

	if parentFields != nil && len(rows) > 0 {
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ParentID)
		}

		parentsMap, err = s.prefetchItems(ctx, ids, in.GetLanguage(), convertFields(parentFields))
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	res := make([]*ItemParent, 0, len(rows))

	for _, row := range rows {
		resRow := &ItemParent{
			ItemId:   row.ItemID,
			ParentId: row.ParentID,
			Type:     convertItemParentType(row.Type),
			Catname:  row.Catname,
		}

		if itemFields != nil {
			itemRow, ok := itemsMap[row.ItemID]
			if ok && itemRow != nil {
				resRow.Item, err = s.extractor.Extract(ctx, *itemRow, itemFields, in.GetLanguage())
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}
			}
		}

		if parentFields != nil {
			itemRow, ok := parentsMap[row.ParentID]
			if ok && itemRow != nil {
				resRow.Parent, err = s.extractor.Extract(ctx, *itemRow, parentFields, in.GetLanguage())
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}
			}
		}

		duplicateParentFields := in.GetFields().GetDuplicateParent()
		if duplicateParentFields != nil {
			duplicateRow, err := s.repository.Item(ctx, query.ItemsListOptions{
				ExcludeID: row.ParentID,
				ItemParentChild: &query.ItemParentListOptions{
					ItemID: row.ItemID,
					Type:   schema.ItemParentTypeDefault,
				},
				ItemParentCacheAncestor: &query.ItemParentCacheListOptions{
					ParentID:  row.ParentID,
					StockOnly: true,
				},
			}, convertFields(duplicateParentFields))
			if err != nil && !errors.Is(err, items.ErrItemNotFound) {
				return nil, status.Error(codes.Internal, err.Error())
			}

			if err == nil {
				resRow.DuplicateParent, err = s.extractor.Extract(
					ctx, duplicateRow, duplicateParentFields, in.GetLanguage(),
				)
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}
			}
		}

		duplicateChildFields := in.GetFields().GetDuplicateChild()
		if duplicateChildFields != nil {
			duplicateRow, err := s.repository.Item(ctx, query.ItemsListOptions{
				ExcludeID: row.ItemID,
				ItemParentParent: &query.ItemParentListOptions{
					ParentID: row.ParentID,
					Type:     row.Type,
				},
				ItemParentCacheDescendant: &query.ItemParentCacheListOptions{
					ItemID: row.ItemID,
				},
			}, convertFields(duplicateChildFields))
			if err != nil && !errors.Is(err, items.ErrItemNotFound) {
				return nil, status.Error(codes.Internal, err.Error())
			}

			if err == nil {
				resRow.DuplicateChild, err = s.extractor.Extract(
					ctx, duplicateRow, duplicateChildFields, in.GetLanguage(),
				)
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}
			}
		}

		res = append(res, resRow)
	}

	var paginator *Pages
	if pages != nil {
		paginator = &Pages{
			PageCount:        pages.PageCount,
			First:            pages.First,
			Last:             pages.Last,
			Current:          pages.Current,
			FirstPageInRange: pages.FirstPageInRange,
			LastPageInRange:  pages.LastPageInRange,
			PagesInRange:     pages.PagesInRange,
			TotalItemCount:   pages.TotalItemCount,
			Next:             pages.Next,
			Previous:         pages.Previous,
		}
	}

	return &GetItemParentsResponse{
		Items:     res,
		Paginator: paginator,
	}, nil
}
