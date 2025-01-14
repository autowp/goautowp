package goautowp

import (
	"context"

	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
)

type ItemParentCacheExtractor struct {
	repository    *items.Repository
	itemExtractor *ItemExtractor
}

func NewItemParentCacheExtractor(repository *items.Repository, itemExtractor *ItemExtractor) *ItemParentCacheExtractor {
	return &ItemParentCacheExtractor{
		repository:    repository,
		itemExtractor: itemExtractor,
	}
}

func (s *ItemParentCacheExtractor) preloadItems( //nolint: dupl
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

func (s *ItemParentCacheExtractor) ExtractRows(
	ctx context.Context, rows []*schema.ItemParentCacheRow, fields *ItemParentCacheFields, lang string,
) ([]*ItemParentCache, error) {
	parentIDs := make([]int64, 0, len(rows))

	for _, row := range rows {
		if row.ParentID != 0 {
			parentIDs = append(parentIDs, row.ParentID)
		}
	}

	itemRequest := fields.GetParentItem()

	parentItemRows, err := s.preloadItems(ctx, itemRequest, parentIDs, lang)
	if err != nil {
		return nil, err
	}

	result := make([]*ItemParentCache, 0, len(rows))

	for _, row := range rows {
		resultRow := &ItemParentCache{
			ItemId:   row.ItemID,
			ParentId: row.ParentID,
		}

		if itemRequest != nil {
			resultRow.ParentItem = parentItemRows[row.ParentID]
		}

		result = append(result, resultRow)
	}

	return result, nil
}
