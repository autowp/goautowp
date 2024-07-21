package goautowp

import (
	"context"
	"database/sql"
	"errors"

	"github.com/autowp/goautowp/attrs"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/index"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/textstorage"
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

const itemLinkNameMaxLength = 255

const typicalPicturesInList = 4

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
	}
}

func (s *ItemsGRPCServer) GetTopBrandsList(
	ctx context.Context,
	in *GetTopBrandsListRequest,
) (*APITopBrandsList, error) {
	if s == nil {
		return nil, status.Error(codes.Internal, "self not initialized")
	}

	cache, err := s.index.BrandsCache(ctx, in.GetLanguage())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	brands := make([]*APITopBrandsListItem, len(cache.Items))
	for idx, brand := range cache.Items {
		brands[idx] = &APITopBrandsListItem{
			Id:            brand.ID,
			Catname:       brand.Catname,
			Name:          brand.NameOnly,
			ItemsCount:    brand.DescendantsCount,
			NewItemsCount: brand.NewDescendantsCount,
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

	switch in.GetPictureItemType() { //nolint:exhaustive
	case PictureItemType_PICTURE_CONTENT:
		pictureItemType = pictures.ItemPictureContent
	case PictureItemType_PICTURE_AUTHOR:
		pictureItemType = pictures.ItemPictureAuthor
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
			Catname:  itm.Catname,
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
			Catname:  twin.Catname,
			Name:     twin.NameOnly,
			Count:    twin.ItemsCount,
			NewCount: twin.NewItemsCount,
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
	twinsData, _, err := s.repository.List(ctx, items.ListOptions{
		Language: in.GetLanguage(),
		Fields: items.ListFields{
			NameOnly: true,
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
	}, false)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	is := make([]*APITwinsBrandsListItem, len(twinsData))
	for idx, brand := range twinsData {
		is[idx] = &APITwinsBrandsListItem{
			Id:       brand.ID,
			Catname:  brand.Catname,
			Name:     brand.NameOnly,
			Count:    brand.ItemsCount,
			NewCount: brand.NewItemsCount,
		}
	}

	return &APITwinsBrandsList{
		Items: is,
	}, nil
}

func mapPicturesRequest(request *PicturesRequest, dest *items.PicturesOptions) {
	dest.OwnerID = request.GetOwnerId()

	switch request.GetStatus() {
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

	if request.GetItemPicture() != nil {
		dest.ItemPicture = &items.ItemPicturesOptions{}
		mapItemPicturesRequest(request.GetItemPicture(), dest.ItemPicture)
	}
}

func mapItemsRequest(in *ListItemsRequest, options *items.ListOptions) error {
	options.NoParents = in.GetNoParent()
	options.Catname = in.GetCatname()
	options.IsConcept = in.GetIsConcept()
	options.Name = in.GetName()
	options.ItemID = in.GetId()
	options.EngineItemID = in.GetEngineId()

	if in.GetAncestorId() != 0 {
		options.AncestorItems = &items.ListOptions{
			ItemID: in.GetAncestorId(),
		}
	}

	if in.GetOrder() == ListItemsRequest_NAME_NAT {
		options.SortByName = true
	}

	itemTypeID := reverseConvertItemTypeID(in.GetTypeId())
	if itemTypeID != 0 {
		options.TypeID = []items.ItemType{itemTypeID}
	}

	if in.GetDescendant() != nil {
		options.DescendantItems = &items.ListOptions{}

		err := mapItemsRequest(in.GetDescendant(), options.DescendantItems)
		if err != nil {
			return err
		}
	}

	if in.GetParent() != nil {
		options.ParentItems = &items.ListOptions{}

		err := mapItemsRequest(in.GetParent(), options.ParentItems)
		if err != nil {
			return err
		}
	}

	if in.GetDescendantPictures() != nil {
		options.DescendantPictures = &items.ItemPicturesOptions{}
		mapItemPicturesRequest(in.GetDescendantPictures(), options.DescendantPictures)
	}

	if in.GetPreviewPictures() != nil {
		options.PreviewPictures = &items.ItemPicturesOptions{}
		mapItemPicturesRequest(in.GetPreviewPictures(), options.PreviewPictures)
	}

	return nil
}

func mapItemPicturesRequest(request *ItemPicturesRequest, dest *items.ItemPicturesOptions) {
	if request.GetPictures() != nil {
		dest.Pictures = &items.PicturesOptions{}
		mapPicturesRequest(request.GetPictures(), dest.Pictures)
	}

	switch request.GetTypeId() {
	case ItemPictureType_ITEM_PICTURE_UNKNOWN:
	case ItemPictureType_ITEM_PICTURE_CONTENT:
		dest.TypeID = pictures.ItemPictureContent
	case ItemPictureType_ITEM_PICTURE_AUTHOR:
		dest.TypeID = pictures.ItemPictureAuthor
	case ItemPictureType_ITEM_PICTURE_COPYRIGHTS:
		dest.TypeID = pictures.ItemPictureCopyrights
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
		NameOnly:                   fields.GetNameOnly(),
		NameHTML:                   fields.GetNameHtml(),
		NameText:                   fields.GetNameText(),
		NameDefault:                fields.GetNameDefault(),
		Description:                fields.GetDescription(),
		HasText:                    fields.GetHasText(),
		PreviewPictures:            previewPictures,
		TotalPictures:              fields.GetTotalPictures(),
		DescendantsCount:           fields.GetDescendantsCount(),
		CurrentPicturesCount:       fields.GetCurrentPicturesCount(),
		ChildsCount:                fields.GetChildsCount(),
		DescendantTwinsGroupsCount: fields.GetDescendantTwinsGroupsCount(),
		InboxPicturesCount:         fields.GetInboxPicturesCount(),
		FullName:                   fields.GetFullName(),
		Logo:                       fields.GetLogo120(),
		MostsActive:                fields.GetMostsActive(),
		CommentsAttentionsCount:    fields.GetCommentsAttentionsCount(),
	}

	return result
}

func (s *ItemsGRPCServer) Item(ctx context.Context, in *ItemRequest) (*APIItem, error) {
	fields := convertFields(in.GetFields())

	res, err := s.repository.Item(ctx, in.GetId(), in.GetLanguage(), fields)
	if err != nil {
		if errors.Is(err, items.ErrItemNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	localizer := s.i18n.Localizer(in.GetLanguage())

	return s.extractor.Extract(ctx, res, in.GetFields(), localizer)
}

func (s *ItemsGRPCServer) List(ctx context.Context, in *ListItemsRequest) (*APIItemList, error) {
	options := items.ListOptions{
		Language: in.GetLanguage(),
		Limit:    in.GetLimit(),
		Page:     in.GetPage(),
		Fields:   convertFields(in.GetFields()),
		OrderBy: []exp.OrderedExpression{
			goqu.T("i").Col("name").Asc(),
			goqu.T("i").Col("body").Asc(),
			goqu.T("i").Col("spec_id").Asc(),
			goqu.T("i").Col("begin_order_cache").Asc(),
			goqu.T("i").Col("end_order_cache").Asc(),
		},
	}

	err := mapItemsRequest(in, &options)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	res, pages, err := s.repository.List(ctx, options, true)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	localizer := s.i18n.Localizer(in.GetLanguage())

	is := make([]*APIItem, len(res))
	for idx, i := range res {
		is[idx], err = s.extractor.Extract(ctx, i, in.GetFields(), localizer)
		if err != nil {
			return nil, err
		}
	}

	var paginator *Pages
	if pages != nil {
		paginator = &Pages{
			PageCount:        pages.PageCount,
			First:            pages.First,
			Current:          pages.Current,
			FirstPageInRange: pages.FirstPageInRange,
			LastPageInRange:  pages.LastPageInRange,
			PagesInRange:     pages.PagesInRange,
			TotalItemCount:   pages.TotalItemCount,
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
		return nil, status.Error(codes.NotFound, err.Error())
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
			&validation.InArray{Haystack: []string{
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

	var found bool

	success, err := s.db.Select(goqu.L("1")).From(schema.ItemTable).Where(
		schema.ItemTableIDCol.Eq(in.GetItemId()),
		schema.ItemTableItemTypeIDCol.In([]items.ItemType{items.VEHICLE, items.TWINS}),
	).ScanValContext(ctx, &found)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !success {
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

	totalBrands, err := s.repository.Count(ctx, items.ListOptions{
		TypeID: []items.ItemType{items.BRAND},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	totalCars, err := s.repository.Count(ctx, items.ListOptions{
		TypeID: []items.ItemType{items.VEHICLE},
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

	itemsWithBeginYear, err := s.repository.Count(ctx, items.ListOptions{
		HasBeginYear: true,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	itemsWithBeginAndEndYears, err := s.repository.Count(ctx, items.ListOptions{
		HasBeginYear: true,
		HasEndYear:   true,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	itemsWithBeginAndEndYearsAndMonths, err := s.repository.Count(ctx, items.ListOptions{
		HasBeginYear:  true,
		HasEndYear:    true,
		HasBeginMonth: true,
		HasEndMonth:   true,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	brandsWithLogo, err := s.repository.Count(ctx, items.ListOptions{
		HasLogo: true,
		TypeID:  []items.ItemType{items.BRAND},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	totalPictures, err := s.picturesRepository.Count(ctx, pictures.ListOptions{})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	picturesWithCopyrights, err := s.picturesRepository.Count(ctx, pictures.ListOptions{
		HasCopyrights: true,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &StatsResponse{
		Values: []*StatsValue{
			{
				Name:  "moder/statistics/photos-with-copyrights",
				Total: int32(totalPictures),
				Value: int32(picturesWithCopyrights),
			},
			{
				Name:  "moder/statistics/vehicles-with-4-or-more-photos",
				Total: int32(totalCars),
				Value: carsWith4OrMorePictures,
			},
			{
				Name:  "moder/statistics/specifications-values",
				Total: int32(totalCars) * totalCarAttrs,
				Value: carAttrsValues,
			},
			{
				Name:  "moder/statistics/brand-logos",
				Total: int32(totalBrands),
				Value: int32(brandsWithLogo),
			},
			{
				Name:  "moder/statistics/from-years",
				Total: int32(totalCars),
				Value: int32(itemsWithBeginYear),
			},
			{
				Name:  "moder/statistics/from-and-to-years",
				Total: int32(totalCars),
				Value: int32(itemsWithBeginAndEndYears),
			},
			{
				Name:  "moder/statistics/from-and-to-years-and-months",
				Total: int32(totalCars),
				Value: int32(itemsWithBeginAndEndYearsAndMonths),
			},
		},
	}, nil
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
			&validation.StringLength{Min: 0, Max: items.ItemLanguageNameMaxLength},
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
