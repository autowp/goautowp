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

var (
	itemTypeCanHaveSpecs = []schema.ItemTableItemTypeID{
		schema.ItemTableItemTypeIDCategory, schema.ItemTableItemTypeIDEngine, schema.ItemTableItemTypeIDTwins,
		schema.ItemTableItemTypeIDVehicle,
	}
	itemTypeCanHaveRoute = []schema.ItemTableItemTypeID{
		schema.ItemTableItemTypeIDCategory, schema.ItemTableItemTypeIDTwins, schema.ItemTableItemTypeIDBrand,
		schema.ItemTableItemTypeIDEngine, schema.ItemTableItemTypeIDVehicle,
	}

	errPreloadNotImplemented = errors.New("preload item_parent with limit not implemented")
)

type ItemExtractor struct {
	container *Container
}

func NewItemExtractor(
	container *Container,
) *ItemExtractor {
	return &ItemExtractor{container: container}
}

func (s *ItemExtractor) preloadItemParentChilds(
	ctx context.Context, request *ItemParentsRequest, ids []int64, lang string, isModer bool, userID int64, role string,
) (map[int64][]*ItemParent, error) {
	if request == nil {
		return nil, nil //nolint: nilnil
	}

	result := make(map[int64][]*ItemParent, len(ids))

	if len(ids) == 0 {
		return result, nil
	}

	options, err := convertItemParentListOptions(request.GetOptions())
	if err != nil {
		return nil, err
	}

	if options == nil {
		options = &query.ItemParentListOptions{}
	}

	itemsRepository, err := s.container.ItemsRepository()
	if err != nil {
		return nil, err
	}

	fields := convertItemParentFields(request.GetFields())
	orderBy := convertItemParentOrder(request.GetOrder())

	var rows []*items.ItemParent

	limit := request.GetLimit()
	if limit > 0 {
		return nil, errPreloadNotImplemented
	}

	options.ParentIDs = ids

	rows, _, err = itemsRepository.ItemParents(ctx, options, fields, orderBy)
	if err != nil {
		return nil, err
	}

	itemParentExtractor := s.container.ItemParentExtractor()

	extractedRows, err := itemParentExtractor.ExtractRows(ctx, rows, request.GetFields(), lang, isModer, userID, role)
	if err != nil {
		return nil, err
	}

	for _, row := range extractedRows {
		parentID := row.GetParentId()
		if _, ok := result[parentID]; !ok {
			result[parentID] = make([]*ItemParent, 0)
		}

		result[parentID] = append(result[parentID], row)
	}

	return result, nil
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

	order := convertPictureItemsOrder(request.GetOrder())

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

		rows, err = picturesRepository.PictureItems(ctx, options, order, 0)
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

	itemParentChildsRequest := fields.GetItemParentChilds()

	itemParentChildRows, err := s.preloadItemParentChilds(ctx, itemParentChildsRequest, ids, lang, isModer, userID, role)
	if err != nil {
		return nil, err
	}

	itemRepository, err := s.container.ItemsRepository()
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

		resultRow.Logo120, resultRow.Brandicon, err = s.extractLogos(ctx, fields, row)
		if err != nil {
			return nil, err
		}

		resultRow.IsCompilesItemOfDay, err = s.extractIsCompilesItemOfDay(ctx, fields, row)
		if err != nil {
			return nil, err
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

		resultRow.AltNames, err = s.extractAltNames(ctx, fields, row, lang)
		if err != nil {
			return nil, err
		}

		resultRow.Design, err = s.extractDesignInfo(ctx, fields, row, lang)
		if err != nil {
			return nil, err
		}

		resultRow.ChildsCounts, err = s.extractChildsCount(ctx, fields, row)
		if err != nil {
			return nil, err
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

		resultRow.Route, resultRow.SpecsRoute, err = s.extractRoutes(ctx, fields, row)
		if err != nil {
			return nil, err
		}

		if fields.GetHasText() {
			resultRow.HasText, err = itemRepository.HasFullText(ctx, row.ID)
			if err != nil {
				return nil, err
			}
		}

		resultRow.Categories, err = s.extractCategories(ctx, fields, row, lang, isModer, userID, role)
		if err != nil {
			return nil, err
		}

		resultRow.Twins, err = s.extractTwins(ctx, fields, row, lang, isModer, userID, role)
		if err != nil {
			return nil, err
		}

		if fields.GetCanEditSpecs() {
			resultRow.CanEditSpecs = util.Contains(itemTypeCanHaveSpecs, row.ItemTypeID) &&
				s.container.Enforcer().Enforce(role, "specifications", "edit")
		}

		if itemParentChildsRequest != nil {
			resultRow.ItemParentChilds = &ItemParents{
				Items: itemParentChildRows[row.ID],
			}
		}

		resultRow.CommentsCount, err = s.extractCommentsCount(ctx, fields, row)
		if err != nil {
			return nil, err
		}

		resultRow.PreviewPictures, err = s.extractPreviewPictures(ctx, fields, row, lang, isModer, userID, role)
		if err != nil {
			return nil, err
		}

		resultRow.EngineVehicles, err = s.extractEngineVehicles(ctx, fields, row, lang, isModer, userID, role)
		if err != nil {
			return nil, err
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

func (s *ItemExtractor) extractChildsCount(
	ctx context.Context, fields *ItemFields, row *items.Item,
) ([]*ChildsCount, error) {
	if !fields.GetChildsCounts() {
		return nil, nil
	}

	itemRepository, err := s.container.ItemsRepository()
	if err != nil {
		return nil, err
	}

	childCounts, err := itemRepository.ChildsCounts(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	return convertChildsCounts(childCounts), nil
}

func (s *ItemExtractor) extractEngineVehicles(
	ctx context.Context, fields *ItemFields, row *items.Item, lang string, isModer bool, userID int64, role string,
) ([]*APIItem, error) {
	evs := fields.GetEngineVehicles()
	if evs == nil {
		return nil, nil
	}

	if row.ItemTypeID != schema.ItemTableItemTypeIDEngine {
		return nil, nil
	}

	itemRepository, err := s.container.ItemsRepository()
	if err != nil {
		return nil, err
	}

	itemExtractor := s.container.ItemExtractor()

	itemFields := evs.GetFields()
	if itemFields == nil {
		itemFields = &ItemFields{}
	}

	listOptions := evs.GetOptions()
	if listOptions == nil {
		listOptions = &ItemListOptions{}
	}

	listOptions.EngineId = row.ID

	repoListOptions, err := convertItemListOptions(listOptions)
	if err != nil {
		return nil, err
	}

	rows, _, err := itemRepository.List(ctx, repoListOptions, convertItemFields(itemFields), items.OrderByNone, false)
	if err != nil {
		return nil, err
	}

	return itemExtractor.ExtractRows(ctx, rows, itemFields, lang, isModer, userID, role)
}

func (s *ItemExtractor) extractPreviewPictures(
	ctx context.Context, fields *ItemFields, row *items.Item, lang string, isModer bool, userID int64, role string,
) (*PreviewPictures, error) {
	pps := fields.GetPreviewPictures()
	if pps == nil {
		return nil, nil //nolint: nilnil
	}

	pictureRepository, err := s.container.PicturesRepository()
	if err != nil {
		return nil, err
	}

	cFetcher := NewPerspectivePictureFetcher(pictureRepository)

	result, err := cFetcher.Fetch(ctx, row.ItemRow, PerspectivePictureFetcherOptions{
		OnlyExactlyPictures:   false,
		PerspectivePageID:     pps.GetPerspectivePageId(),
		PictureItemTypeID:     convertPictureItemType(pps.GetTypeId()),
		PerspectiveID:         pps.GetPerspectiveId(),
		ContainsPerspectiveID: pps.GetContainsPerspectiveId(),
	})
	if err != nil {
		return nil, err
	}

	// if pps.GetRoute() {
	//	for idx, picture := range result.Pictures {
	//		if picture != nil {
	//			pictures.Pictures[idx].Route = []string{"/picture", picture.Row.Identity}
	//		}
	//	}
	// }

	pictureExtractor := s.container.PictureExtractor()

	pictureFields := pps.GetPicture()
	if pictureFields == nil {
		pictureFields = &PictureFields{}
	}

	extractedPics := make([]*Picture, 0, len(result.Pictures))

	for idx, pic := range result.Pictures {
		pictureFields.ThumbLarge = result.LargeFormat && idx == 0
		pictureFields.ThumbMedium = !pictureFields.ThumbLarge

		var extractedPic *Picture
		if pic != nil && pic.Row != nil {
			extractedPic, err = pictureExtractor.Extract(ctx, pic.Row, pictureFields, lang, isModer, userID, role)
			if err != nil {
				return nil, err
			}
		}

		extractedPics = append(extractedPics, extractedPic)
	}

	return &PreviewPictures{
		LargeFormat:   result.LargeFormat,
		Pictures:      extractedPics,
		TotalPictures: result.TotalPictures,
	}, nil
}

func (s *ItemExtractor) extractIsCompilesItemOfDay(
	ctx context.Context, fields *ItemFields, row *items.Item,
) (bool, error) {
	if !fields.GetIsCompilesItemOfDay() {
		return false, nil
	}

	itemOfDayRepository, err := s.container.ItemOfDayRepository()
	if err != nil {
		return false, err
	}

	return itemOfDayRepository.IsComplies(ctx, row.ID)
}

func (s *ItemExtractor) extractCommentsCount(ctx context.Context, fields *ItemFields, row *items.Item) (int32, error) {
	if !fields.GetCommentsCount() {
		return 0, nil
	}

	commentsRepo, err := s.container.CommentsRepository()
	if err != nil {
		return 0, err
	}

	return commentsRepo.TopicStat(ctx, schema.CommentMessageTypeIDItems, row.ID)
}

func (s *ItemExtractor) extractLogos(
	ctx context.Context, fields *ItemFields, row *items.Item,
) (*APIImage, *APIImage, error) {
	if !row.LogoID.Valid {
		return nil, nil, nil
	}

	var (
		logo120   *APIImage
		brandicon *APIImage
	)

	imageStorage, err := s.container.ImageStorage()
	if err != nil {
		return nil, nil, err
	}

	if fields.GetLogo120() {
		img, err := imageStorage.FormattedImage(ctx, int(row.LogoID.Int64), "logo")
		if err != nil {
			return nil, nil, err
		}

		logo120 = APIImageToGRPC(img)
	}

	if fields.GetBrandicon() {
		img, err := imageStorage.FormattedImage(ctx, int(row.LogoID.Int64), "brandicon2")
		if err != nil {
			return nil, nil, err
		}

		brandicon = APIImageToGRPC(img)
	}

	return logo120, brandicon, nil
}

func (s *ItemExtractor) extractConnectedItems(
	ctx context.Context, request *ItemsRequest, opts *query.ItemListOptions, lang string, isModer bool, userID int64,
	role string,
) ([]*APIItem, error) {
	itemRepository, err := s.container.ItemsRepository()
	if err != nil {
		return nil, err
	}

	var order items.OrderBy

	order, opts.SortByName = convertItemOrder(request.GetOrder())

	opts.Language = lang

	rows, _, err := itemRepository.List(ctx, opts, convertItemFields(request.GetFields()), order, false)
	if err != nil {
		return nil, err
	}

	return s.ExtractRows(ctx, rows, request.GetFields(), lang, isModer, userID, role)
}

func (s *ItemExtractor) extractTwins(
	ctx context.Context, fields *ItemFields, row *items.Item, lang string, isModer bool, userID int64, role string,
) ([]*APIItem, error) {
	twinsRequest := fields.GetTwins()
	if twinsRequest == nil {
		return nil, nil
	}

	opts := &query.ItemListOptions{
		ItemParentCacheDescendant: &query.ItemParentCacheListOptions{ItemID: row.ID},
		TypeID:                    []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDTwins},
	}

	return s.extractConnectedItems(ctx, twinsRequest, opts, lang, isModer, userID, role)
}

func (s *ItemExtractor) extractCategories(
	ctx context.Context, fields *ItemFields, row *items.Item, lang string, isModer bool, userID int64, role string,
) ([]*APIItem, error) {
	categoriesRequest := fields.GetCategories()
	if categoriesRequest == nil {
		return nil, nil
	}

	opts := &query.ItemListOptions{
		ItemParentChild: &query.ItemParentListOptions{
			ChildItems: &query.ItemListOptions{
				TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDVehicle, schema.ItemTableItemTypeIDEngine},
			},
			ItemParentCacheAncestorByChildID: &query.ItemParentCacheListOptions{ItemID: row.ID},
		},
		TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDCategory},
	}

	return s.extractConnectedItems(ctx, categoriesRequest, opts, lang, isModer, userID, role)
}

func (s *ItemExtractor) extractRoutes(
	ctx context.Context, fields *ItemFields, row *items.Item,
) ([]string, []string, error) {
	extractRoute := fields.GetRoute() && util.Contains(itemTypeCanHaveRoute, row.ItemTypeID)
	extractSpecsRoute := fields.GetSpecsRoute() && util.Contains(itemTypeCanHaveSpecs, row.ItemTypeID) && row.HasSpecs

	var (
		route      []string
		specsRoute []string
	)

	if extractSpecsRoute || extractRoute {
		itemRepository, err := s.container.ItemsRepository()
		if err != nil {
			return nil, nil, err
		}

		cataloguePaths, err := itemRepository.CataloguePath(ctx, row.ID, items.CataloguePathOptions{
			BreakOnFirst: true,
			ToBrand:      true,
			StockFirst:   true,
		})
		if err != nil {
			return nil, nil, err
		}

		if extractRoute {
			switch row.ItemTypeID {
			case schema.ItemTableItemTypeIDCategory:
				route = []string{"/category", util.NullStringToString(row.Catname)}
			case schema.ItemTableItemTypeIDTwins:
				route = []string{"/twins/group", strconv.FormatInt(row.ID, 10)}

			case schema.ItemTableItemTypeIDBrand:
				route = []string{"/", util.NullStringToString(row.Catname)}

			case schema.ItemTableItemTypeIDEngine,
				schema.ItemTableItemTypeIDVehicle:
				for _, cPath := range cataloguePaths {
					route = append([]string{"/", cPath.BrandCatname, cPath.CarCatname}, cPath.Path...)

					break
				}
			case schema.ItemTableItemTypeIDFactory,
				schema.ItemTableItemTypeIDMuseum,
				schema.ItemTableItemTypeIDCopyright,
				schema.ItemTableItemTypeIDPerson:
			}
		}

		if extractSpecsRoute {
			for _, path := range cataloguePaths {
				res := append([]string{"/", path.BrandCatname, path.CarCatname}, path.Path...)
				res = append(res, "specifications")

				specsRoute = res

				break
			}
		}
	}

	return route, specsRoute, nil
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

func (s *ItemExtractor) extractAltNames(
	ctx context.Context, fields *ItemFields, row *items.Item, lang string,
) ([]*AltName, error) {
	if !fields.GetAltNames() {
		return nil, nil
	}

	// alt names
	altNames := make(map[string][]string)
	altNames2 := make(map[string][]string)

	itemRepository, err := s.container.ItemsRepository()
	if err != nil {
		return nil, err
	}

	langNames, err := itemRepository.Names(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	currentLangName := ""

	for clang, langName := range langNames {
		if clang == items.DefaultLanguageCode {
			continue
		}

		name := langName
		if _, ok := altNames[name]; !ok {
			altNames[langName] = make([]string, 0)
		}

		altNames[name] = append(altNames[name], clang)

		if lang == clang {
			currentLangName = name
		}
	}

	for name, codes := range altNames {
		if name != currentLangName {
			altNames2[name] = codes
		}
	}

	if len(currentLangName) > 0 {
		delete(altNames2, currentLangName)
	}

	res := make([]*AltName, 0, len(altNames2))
	for name, languages := range altNames2 {
		res = append(res, &AltName{
			Languages: languages,
			Name:      name,
		})
	}

	return res, nil
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
