package goautowp

import (
	"context"
	"errors"
	"strconv"

	"github.com/autowp/goautowp/attrs"
	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/itemofday"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/casbin/casbin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"google.golang.org/genproto/googleapis/type/latlng"
)

type ItemExtractor struct {
	enforcer            *casbin.Enforcer
	nameFormatter       *items.ItemNameFormatter
	imageStorage        *storage.Storage
	commentsRepository  *comments.Repository
	picturesRepository  *pictures.Repository
	itemRepository      *items.Repository
	itemOfDayRepository *itemofday.Repository
	attrs               *attrs.Repository
}

func NewItemExtractor(
	enforcer *casbin.Enforcer,
	imageStorage *storage.Storage,
	commentsRepository *comments.Repository,
	picturesRepository *pictures.Repository,
	itemRepository *items.Repository,
	itemOfDayRepository *itemofday.Repository,
	attrs *attrs.Repository,
) *ItemExtractor {
	return &ItemExtractor{
		enforcer:            enforcer,
		nameFormatter:       &items.ItemNameFormatter{},
		imageStorage:        imageStorage,
		commentsRepository:  commentsRepository,
		picturesRepository:  picturesRepository,
		itemOfDayRepository: itemOfDayRepository,
		itemRepository:      itemRepository,
		attrs:               attrs,
	}
}

func (s *ItemExtractor) Extract(
	ctx context.Context, row items.Item, fields *ItemFields, localizer *i18n.Localizer, lang string,
) (*APIItem, error) {
	if fields == nil {
		fields = &ItemFields{}
	}

	result := &APIItem{
		Id:                         row.ID,
		Catname:                    util.NullStringToString(row.Catname),
		EngineItemId:               util.NullInt64ToScalar(row.EngineItemID),
		EngineInherit:              row.EngineInherit,
		DescendantsCount:           row.DescendantsCount,
		ItemTypeId:                 convertItemTypeID(row.ItemTypeID),
		IsConcept:                  row.IsConcept,
		IsConceptInherit:           row.IsConceptInherit,
		SpecId:                     int64(util.NullInt32ToScalar(row.SpecID)),
		Description:                row.Description,
		FullText:                   row.FullText,
		DescendantPicturesCount:    row.DescendantPicturesCount,
		ChildsCount:                row.ChildsCount,
		DescendantTwinsGroupsCount: row.DescendantTwinsGroupsCount,
		InboxPicturesCount:         row.InboxPicturesCount,
		AcceptedPicturesCount:      row.AcceptedPicturesCount,
		FullName:                   row.FullName,
		MostsActive:                row.MostsActive,
		CommentsAttentionsCount:    row.CommentsAttentionsCount,
		HasChildSpecs:              row.HasChildSpecs,
		HasSpecs:                   row.HasSpecs,
		Produced:                   util.NullInt32ToScalar(row.Produced),
		ProducedExactly:            row.ProducedExactly,
		IsGroup:                    row.IsGroup,
	}

	if fields.GetLogo120() && row.LogoID.Valid {
		logo120, err := s.imageStorage.FormattedImage(ctx, int(row.LogoID.Int64), "logo")
		if err != nil {
			return nil, err
		}

		result.Logo120 = APIImageToGRPC(logo120)
	}

	if fields.GetBrandicon() && row.LogoID.Valid {
		brandicon2, err := s.imageStorage.FormattedImage(ctx, int(row.LogoID.Int64), "brandicon2")
		if err != nil {
			return nil, err
		}

		result.Brandicon = APIImageToGRPC(brandicon2)
	}

	if fields.GetNameOnly() {
		result.NameOnly = row.NameOnly
	}

	if fields.GetNameText() || fields.GetNameHtml() {
		formatterOptions := items.ItemNameFormatterOptions{
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
		}

		if fields.GetNameText() {
			nameText, err := s.nameFormatter.FormatText(formatterOptions, localizer)
			if err != nil {
				return nil, err
			}

			result.NameText = nameText
		}

		if fields.GetNameHtml() {
			nameHTML, err := s.nameFormatter.FormatHTML(formatterOptions, localizer)
			if err != nil {
				return nil, err
			}

			result.NameHtml = nameHTML
		}
	}

	if fields.GetIsCompilesItemOfDay() {
		IsCompiles, err := s.itemOfDayRepository.IsComplies(ctx, row.ID)
		if err != nil {
			return nil, err
		}

		result.IsCompilesItemOfDay = IsCompiles
	}

	if fields.GetAttrZoneId() {
		vehicleTypes, err := s.itemRepository.VehicleTypes(ctx, row.ID, false)
		if err != nil {
			return nil, err
		}

		result.AttrZoneId = s.attrs.ZoneIDByVehicleTypeIDs(row.ItemTypeID, vehicleTypes)
	}

	if fields.GetLocation() {
		location, err := s.itemRepository.ItemLocation(ctx, row.ID)
		if err != nil {
			if !errors.Is(err, items.ErrItemNotFound) {
				return nil, err
			}
		} else {
			result.Location = &latlng.LatLng{
				Latitude:  location.Lat(),
				Longitude: location.Lng(),
			}
		}
	}

	if fields.GetOtherNames() {
		rows, err := s.itemRepository.Names(ctx, row.ID)
		if err != nil {
			return nil, err
		}

		otherNames := make([]string, 0, len(rows))
		for _, name := range rows {
			if row.Name != name && !util.Contains(otherNames, name) {
				otherNames = append(otherNames, name)
			}
		}

		result.OtherNames = otherNames
	}

	if fields.GetDesign() {
		designInfo, err := s.itemRepository.DesignInfo(ctx, row.ID, lang)
		if err != nil {
			return nil, err
		}

		if designInfo != nil {
			result.Design = &Design{
				Name:  designInfo.Name,
				Route: designInfo.Route,
			}
		}
	}

	var err error

	result.SpecsRoute, err = s.extractSpecsRoute(ctx, fields, row)
	if err != nil {
		return nil, err
	}

	if fields.GetChildsCounts() {
		childCounts, err := s.itemRepository.ChildsCounts(ctx, row.ID)
		if err != nil {
			return nil, err
		}

		result.ChildsCounts = convertChildsCounts(childCounts)
	}

	if fields.GetPublicRoutes() {
		result.PublicRoutes, err = s.ItemPublicRoutes(ctx, row)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *ItemExtractor) extractSpecsRoute(ctx context.Context, fields *ItemFields, row items.Item) ([]string, error) {
	if fields.GetSpecsRoute() {
		itemTypeCanHaveSpecs := []schema.ItemTableItemTypeID{
			schema.ItemTableItemTypeIDCategory, schema.ItemTableItemTypeIDEngine, schema.ItemTableItemTypeIDTwins,
			schema.ItemTableItemTypeIDVehicle,
		}
		if util.Contains(itemTypeCanHaveSpecs, row.ItemTypeID) && row.HasSpecs {
			specsRoute, err := s.itemRepository.SpecsRoute(ctx, row.ID)
			if err != nil {
				return nil, err
			}

			return specsRoute, nil
		}
	}

	return nil, nil
}

func (s *ItemExtractor) ItemPublicRoutes(ctx context.Context, item items.Item) ([]*PublicRoute, error) {
	if item.ItemTypeID == schema.ItemTableItemTypeIDFactory {
		return []*PublicRoute{
			{Route: []string{"/factories", strconv.FormatInt(item.ID, decimal)}},
		}, nil
	}

	if item.ItemTypeID == schema.ItemTableItemTypeIDCategory {
		return []*PublicRoute{
			{Route: []string{"/category", util.NullStringToString(item.Catname)}},
		}, nil
	}

	if item.ItemTypeID == schema.ItemTableItemTypeIDTwins {
		return []*PublicRoute{
			{Route: []string{"/twins", "group", strconv.FormatInt(item.ID, decimal)}},
		}, nil
	}

	if item.ItemTypeID == schema.ItemTableItemTypeIDBrand {
		return []*PublicRoute{
			{Route: []string{"/" + util.NullStringToString(item.Catname)}},
		}, nil
	}

	return s.walkUpUntilBrand(ctx, item.ID, []string{})
}

func (s *ItemExtractor) walkUpUntilBrand(ctx context.Context, id int64, path []string) ([]*PublicRoute, error) {
	routes := make([]*PublicRoute, 0)

	parentRows, _, err := s.itemRepository.ItemParents(ctx, query.ItemParentListOptions{
		ItemID: id,
	}, items.ItemParentFields{}, items.ItemParentOrderByNone)
	if err != nil {
		return nil, err
	}

	for _, parentRow := range parentRows {
		brand, err := s.itemRepository.Item(ctx, query.ItemsListOptions{
			TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
			ItemID: parentRow.ParentID,
		}, items.ListFields{})
		if err != nil && !errors.Is(err, items.ErrItemNotFound) {
			return nil, err
		}

		if err == nil {
			routes = append(routes, &PublicRoute{
				Route: append([]string{"/", util.NullStringToString(brand.Catname), parentRow.Catname}, path...),
			})
		}

		nextRoutes, err := s.walkUpUntilBrand(ctx, parentRow.ParentID, append([]string{parentRow.Catname}, path...))
		if err != nil {
			return nil, err
		}

		routes = append(routes, nextRoutes...)
	}

	return routes, nil
}
