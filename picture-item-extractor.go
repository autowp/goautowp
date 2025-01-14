package goautowp

import (
	"context"

	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
)

type PictureItemExtractor struct {
	repository               *items.Repository
	itemExtractor            *ItemExtractor
	itemParentCacheExtractor *ItemParentCacheExtractor
}

func NewPictureItemExtractor(
	repository *items.Repository,
	itemExtractor *ItemExtractor,
	itemParentCacheExtractor *ItemParentCacheExtractor,
) *PictureItemExtractor {
	return &PictureItemExtractor{
		repository:               repository,
		itemExtractor:            itemExtractor,
		itemParentCacheExtractor: itemParentCacheExtractor,
	}
}

func (s *PictureItemExtractor) preloadItems( //nolint: dupl
	ctx context.Context, request *ItemsRequest, ids []int64, lang string,
) (map[int64]*APIItem, error) {
	if request == nil {
		return nil, nil //nolint: nilnil
	}

	result := make(map[int64]*APIItem, len(ids))

	if len(ids) == 0 {
		return result, nil
	}

	itemFields := request.GetFields()

	options, err := convertItemListOptions(request.GetOptions())
	if err != nil {
		return nil, err
	}

	if options == nil {
		options = &query.ItemListOptions{}
	}

	options.ItemIDs = ids
	options.Language = lang

	rows, _, err := s.repository.List(
		ctx,
		options,
		convertItemFields(itemFields),
		items.OrderByNone,
		false,
	)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.ID], err = s.itemExtractor.Extract(ctx, row, itemFields, lang)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *PictureItemExtractor) preloadItemParentCache(
	ctx context.Context, request *ItemParentCacheRequest, ids []int64, lang string,
) (map[int64][]*ItemParentCache, error) {
	if request == nil {
		return nil, nil //nolint: nilnil
	}

	result := make(map[int64][]*ItemParentCache, len(ids))

	if len(ids) == 0 {
		return result, nil
	}

	options, err := convertItemParentCacheListOptions(request.GetOptions())
	if err != nil {
		return nil, err
	}

	if options == nil {
		options = &query.ItemParentCacheListOptions{}
	}

	options.ItemIDs = ids

	rows, err := s.repository.ItemParentCaches(ctx, options)
	if err != nil {
		return nil, err
	}

	extractedRows, err := s.itemParentCacheExtractor.ExtractRows(ctx, rows, request.GetFields(), lang)
	if err != nil {
		return nil, err
	}

	for _, row := range extractedRows {
		itemID := row.GetItemId()
		if _, ok := result[itemID]; !ok {
			result[itemID] = make([]*ItemParentCache, 0)
		}

		result[itemID] = append(result[itemID], row)
	}

	return result, nil
}

func (s *PictureItemExtractor) ExtractRows(
	ctx context.Context, rows []*schema.PictureItemRow, fields *PictureItemFields, lang string,
) ([]*PictureItem, error) {
	ids := make([]int64, 0, len(rows))

	for _, row := range rows {
		if row.ItemID != 0 {
			ids = append(ids, row.ItemID)
		}
	}

	itemRequest := fields.GetItem()

	itemRows, err := s.preloadItems(ctx, itemRequest, ids, lang)
	if err != nil {
		return nil, err
	}

	itemParentCacheAncestorRequest := fields.GetItemParentCacheAncestor()

	itemParentCacheAncestorRows, err := s.preloadItemParentCache(ctx, itemParentCacheAncestorRequest, ids, lang)
	if err != nil {
		return nil, err
	}

	result := make([]*PictureItem, 0, len(rows))

	for _, row := range rows {
		resultRow := &PictureItem{
			PictureId:     row.PictureID,
			ItemId:        row.ItemID,
			Type:          extractPictureItemType(row.Type),
			CropLeft:      uint32(util.NullInt32ToScalar(row.CropLeft)),     //nolint:gosec
			CropTop:       uint32(util.NullInt32ToScalar(row.CropTop)),      //nolint:gosec
			CropWidth:     uint32(util.NullInt32ToScalar(row.CropWidth)),    //nolint:gosec
			CropHeight:    uint32(util.NullInt32ToScalar(row.CropHeight)),   //nolint:gosec
			PerspectiveId: int32(util.NullInt64ToScalar(row.PerspectiveID)), //nolint:gosec
		}

		if itemRequest != nil {
			resultRow.Item = itemRows[row.ItemID]
		}

		if itemParentCacheAncestorRequest != nil {
			resultRow.ItemParentCacheAncestors = &ItemParentCaches{
				Items: itemParentCacheAncestorRows[row.ItemID],
			}
		}

		result = append(result, resultRow)
	}

	return result, nil
}

func (s *PictureItemExtractor) Extract(
	ctx context.Context, row *schema.PictureItemRow, fields *PictureItemFields, lang string,
) (*PictureItem, error) {
	result, err := s.ExtractRows(ctx, []*schema.PictureItemRow{row}, fields, lang)

	return result[0], err
}
