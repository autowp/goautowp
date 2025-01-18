package goautowp

import (
	"context"
	"errors"
	"strconv"

	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"google.golang.org/genproto/googleapis/type/latlng"
)

type ItemExtractor struct {
	container *Container
}

func NewItemExtractor(
	container *Container,
) *ItemExtractor {
	return &ItemExtractor{container: container}
}

func (s *ItemExtractor) preloadPictureItems(
	ctx context.Context, request *PictureItemsRequest, ids []int64, lang string, isModer bool, userID int64, role string,
) (map[int64][]*PictureItem, error) {
	if request == nil {
		return nil, nil //nolint: nilnil
	}

	result := make(map[int64][]*PictureItem, len(ids))

	if len(ids) == 0 {
		return result, nil
	}

	options, err := convertPictureItemListOptions(request.GetOptions())
	if err != nil {
		return nil, err
	}

	if options == nil {
		options = &query.PictureItemListOptions{}
	}

	picturesRepository, err := s.container.PicturesRepository()
	if err != nil {
		return nil, err
	}

	var rows []*schema.PictureItemRow

	limit := request.GetLimit()
	if limit > 0 {
		optionsArr := make([]*query.PictureItemListOptions, 0, len(ids))

		for _, id := range ids {
			cOptions := *options
			cOptions.ItemID = id

			optionsArr = append(optionsArr, &cOptions)
		}

		rows, err = picturesRepository.PictureItemsBatch(ctx, optionsArr, limit)
		if err != nil {
			return nil, err
		}
	} else {
		options.ItemIDs = ids

		rows, err = picturesRepository.PictureItems(ctx, options, 0)
		if err != nil {
			return nil, err
		}
	}

	pictureItemExtractor := s.container.PictureItemExtractor()

	extractedRows, err := pictureItemExtractor.ExtractRows(ctx, rows, request.GetFields(), lang, isModer, userID, role)
	if err != nil {
		return nil, err
	}

	for _, row := range extractedRows {
		itemID := row.GetItemId()
		if _, ok := result[itemID]; !ok {
			result[itemID] = make([]*PictureItem, 0)
		}

		result[itemID] = append(result[itemID], row)
	}

	return result, nil
}

func (s *ItemExtractor) ExtractRows(
	ctx context.Context, rows []*items.Item, fields *ItemFields, lang string, isModer bool, userID int64, role string,
) ([]*APIItem, error) {
	if fields == nil {
		fields = &ItemFields{}
	}

	ids := make([]int64, 0, len(rows))

	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	var (
		err    error
		result = make([]*APIItem, 0, len(rows))
	)

	pictureItemRequest := fields.GetPictureItems()

	pictureItemRows, err := s.preloadPictureItems(ctx, pictureItemRequest, ids, lang, isModer, userID, role)
	if err != nil {
		return nil, err
	}

	imageStorage, err := s.container.ImageStorage()
	if err != nil {
		return nil, err
	}

	itemRepository, err := s.container.ItemsRepository()
	if err != nil {
		return nil, err
	}

	itemOfDayRepository, err := s.container.ItemOfDayRepository()
	if err != nil {
		return nil, err
	}

	attrsRepository, err := s.container.AttrsRepository()
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		resultRow := &APIItem{
			Id:                         row.ID,
			Catname:                    util.NullStringToString(row.Catname),
			EngineItemId:               util.NullInt64ToScalar(row.EngineItemID),
			EngineInherit:              row.EngineInherit,
			DescendantsCount:           row.DescendantsCount,
			ItemTypeId:                 extractItemTypeID(row.ItemTypeID),
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
			NameOnly:                   row.NameOnly,
			NameDefault:                row.NameDefault,
		}

		resultRow.NameText, resultRow.NameHtml, err = s.extractNames(fields, row, lang)
		if err != nil {
			return nil, err
		}

		if fields.GetLogo120() && row.LogoID.Valid {
			logo120, err := imageStorage.FormattedImage(ctx, int(row.LogoID.Int64), "logo")
			if err != nil {
				return nil, err
			}

			resultRow.Logo120 = APIImageToGRPC(logo120)
		}

		if fields.GetBrandicon() && row.LogoID.Valid {
			brandicon2, err := imageStorage.FormattedImage(ctx, int(row.LogoID.Int64), "brandicon2")
			if err != nil {
				return nil, err
			}

			resultRow.Brandicon = APIImageToGRPC(brandicon2)
		}

		if fields.GetIsCompilesItemOfDay() {
			IsCompiles, err := itemOfDayRepository.IsComplies(ctx, row.ID)
			if err != nil {
				return nil, err
			}

			resultRow.IsCompilesItemOfDay = IsCompiles
		}

		if fields.GetAttrZoneId() {
			vehicleTypes, err := itemRepository.VehicleTypes(ctx, row.ID, false)
			if err != nil {
				return nil, err
			}

			resultRow.AttrZoneId = attrsRepository.ZoneIDByVehicleTypeIDs(row.ItemTypeID, vehicleTypes)
		}

		resultRow.Location, err = s.extractLocation(ctx, fields, row)
		if err != nil {
			return nil, err
		}

		resultRow.OtherNames, err = s.extractOtherNames(ctx, fields, row)
		if err != nil {
			return nil, err
		}

		resultRow.Design, err = s.extractDesignInfo(ctx, fields, row, lang)
		if err != nil {
			return nil, err
		}

		resultRow.SpecsRoute, err = s.extractSpecsRoute(ctx, fields, row)
		if err != nil {
			return nil, err
		}

		if fields.GetChildsCounts() {
			childCounts, err := itemRepository.ChildsCounts(ctx, row.ID)
			if err != nil {
				return nil, err
			}

			resultRow.ChildsCounts = convertChildsCounts(childCounts)
		}

		if fields.GetPublicRoutes() {
			resultRow.PublicRoutes, err = s.itemPublicRoutes(ctx, row)
			if err != nil {
				return nil, err
			}
		}

		if pictureItemRequest != nil {
			resultRow.PictureItems = &PictureItems{
				Items: pictureItemRows[row.ID],
			}
		}

		result = append(result, resultRow)
	}

	return result, nil
}

func (s *ItemExtractor) Extract(
	ctx context.Context, row *items.Item, fields *ItemFields, lang string, isModer bool, userID int64, role string,
) (*APIItem, error) {
	result, err := s.ExtractRows(ctx, []*items.Item{row}, fields, lang, isModer, userID, role)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, errItemNotFound
	}

	return result[0], nil
}

func (s *ItemExtractor) extractDesignInfo(
	ctx context.Context, fields *ItemFields, row *items.Item, lang string,
) (*Design, error) {
	if !fields.GetDesign() {
		return nil, nil //nolint: nilnil
	}

	itemRepository, err := s.container.ItemsRepository()
	if err != nil {
		return nil, err
	}

	designInfo, err := itemRepository.DesignInfo(ctx, row.ID, lang)
	if err != nil {
		return nil, err
	}

	if designInfo == nil {
		return nil, nil //nolint: nilnil
	}

	return &Design{
		Name:  designInfo.Name,
		Route: designInfo.Route,
	}, nil
}

func (s *ItemExtractor) extractNames(
	fields *ItemFields, row *items.Item, lang string,
) (string, string, error) {
	if !fields.GetNameText() && !fields.GetNameHtml() {
		return "", "", nil
	}

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

	i18nBundle, err := s.container.I18n()
	if err != nil {
		return "", "", err
	}

	itemNameFormatter := items.NewItemNameFormatter(i18nBundle)

	var nameText, nameHTML string

	if fields.GetNameText() {
		nameText, err = itemNameFormatter.FormatText(formatterOptions, lang)
		if err != nil {
			return "", "", err
		}
	}

	if fields.GetNameHtml() {
		nameHTML, err = itemNameFormatter.FormatHTML(formatterOptions, lang)
		if err != nil {
			return "", "", err
		}
	}

	return nameText, nameHTML, nil
}

func (s *ItemExtractor) extractOtherNames(
	ctx context.Context, fields *ItemFields, row *items.Item,
) ([]string, error) {
	if !fields.GetOtherNames() {
		return nil, nil
	}

	itemRepository, err := s.container.ItemsRepository()
	if err != nil {
		return nil, err
	}

	rows, err := itemRepository.Names(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	otherNames := make([]string, 0, len(rows))
	for _, name := range rows {
		if row.NameOnly != name && !util.Contains(otherNames, name) {
			otherNames = append(otherNames, name)
		}
	}

	return otherNames, nil
}

func (s *ItemExtractor) extractLocation(
	ctx context.Context, fields *ItemFields, row *items.Item,
) (*latlng.LatLng, error) {
	if !fields.GetLocation() {
		return nil, nil //nolint: nilnil
	}

	itemRepository, err := s.container.ItemsRepository()
	if err != nil {
		return nil, err
	}

	location, err := itemRepository.ItemLocation(ctx, row.ID)
	if err != nil && !errors.Is(err, items.ErrItemNotFound) {
		if errors.Is(err, items.ErrItemNotFound) {
			return nil, nil //nolint: nilnil
		}

		return nil, err
	}

	return &latlng.LatLng{
		Latitude:  location.Lat(),
		Longitude: location.Lng(),
	}, nil
}

func (s *ItemExtractor) extractSpecsRoute(ctx context.Context, fields *ItemFields, row *items.Item) ([]string, error) {
	if fields.GetSpecsRoute() {
		itemTypeCanHaveSpecs := []schema.ItemTableItemTypeID{
			schema.ItemTableItemTypeIDCategory, schema.ItemTableItemTypeIDEngine, schema.ItemTableItemTypeIDTwins,
			schema.ItemTableItemTypeIDVehicle,
		}
		if util.Contains(itemTypeCanHaveSpecs, row.ItemTypeID) && row.HasSpecs {
			itemRepository, err := s.container.ItemsRepository()
			if err != nil {
				return nil, err
			}

			specsRoute, err := itemRepository.SpecsRoute(ctx, row.ID)
			if err != nil {
				return nil, err
			}

			return specsRoute, nil
		}
	}

	return nil, nil
}

func (s *ItemExtractor) itemPublicRoutes(ctx context.Context, item *items.Item) ([]*PublicRoute, error) {
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

	itemRepository, err := s.container.ItemsRepository()
	if err != nil {
		return nil, err
	}

	parentRows, _, err := itemRepository.ItemParents(ctx, &query.ItemParentListOptions{
		ItemID: id,
	}, items.ItemParentFields{}, items.ItemParentOrderByNone)
	if err != nil {
		return nil, err
	}

	for _, parentRow := range parentRows {
		brand, err := itemRepository.Item(ctx, &query.ItemListOptions{
			TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDBrand},
			ItemID: parentRow.ParentID,
		}, nil)
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
