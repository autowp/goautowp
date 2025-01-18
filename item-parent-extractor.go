package goautowp

import (
	"context"
	"errors"

	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
)

type ItemParentExtractor struct {
	container *Container
}

func NewItemParentExtractor(container *Container) *ItemParentExtractor {
	return &ItemParentExtractor{container: container}
}

func (s *ItemParentExtractor) prefetchItems(
	ctx context.Context, ids []int64, lang string, fields *items.ListFields,
) (map[int64]*items.Item, error) {
	itemsRepository, err := s.container.ItemsRepository()
	if err != nil {
		return nil, err
	}

	itemRows, _, err := itemsRepository.List(ctx, &query.ItemListOptions{
		ItemIDs:  ids,
		Language: lang,
	}, fields, items.OrderByNone, false)
	if err != nil {
		return nil, err
	}

	itemsMap := make(map[int64]*items.Item, len(itemRows))

	for _, itemRow := range itemRows {
		itemsMap[itemRow.ID] = itemRow
	}

	return itemsMap, nil
}

func (s *ItemParentExtractor) ExtractRows(
	ctx context.Context, rows []*items.ItemParent, fields *ItemParentFields, lang string, isModer bool,
	userID int64, role string,
) ([]*ItemParent, error) {
	var err error

	itemFields := fields.GetItem()
	itemsMap := make(map[int64]*items.Item, 0)

	if itemFields != nil && len(rows) > 0 {
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ItemID)
		}

		itemsMap, err = s.prefetchItems(ctx, ids, lang, convertItemFields(itemFields))
		if err != nil {
			return nil, err
		}
	}

	parentFields := fields.GetParent()
	parentsMap := make(map[int64]*items.Item, len(rows))

	if parentFields != nil && len(rows) > 0 {
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ParentID)
		}

		parentsMap, err = s.prefetchItems(ctx, ids, lang, convertItemFields(parentFields))
		if err != nil {
			return nil, err
		}
	}

	res := make([]*ItemParent, 0, len(rows))

	itemExtractor := s.container.ItemExtractor()

	itemRepository, err := s.container.ItemsRepository()
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		resRow := &ItemParent{
			ItemId:   row.ItemID,
			ParentId: row.ParentID,
			Type:     extractItemParentType(row.Type),
			Catname:  row.Catname,
		}

		if itemFields != nil {
			itemRow, ok := itemsMap[row.ItemID]
			if ok && itemRow != nil {
				resRow.Item, err = itemExtractor.Extract(ctx, itemRow, itemFields, lang, isModer, userID, role)
				if err != nil {
					return nil, err
				}
			}
		}

		if parentFields != nil {
			itemRow, ok := parentsMap[row.ParentID]
			if ok && itemRow != nil {
				resRow.Parent, err = itemExtractor.Extract(ctx, itemRow, parentFields, lang, isModer, userID, role)
				if err != nil {
					return nil, err
				}
			}
		}

		duplicateParentFields := fields.GetDuplicateParent()
		if duplicateParentFields != nil {
			duplicateRow, err := itemRepository.Item(ctx, &query.ItemListOptions{
				ExcludeID: row.ParentID,
				ItemParentChild: &query.ItemParentListOptions{
					ItemID: row.ItemID,
					Type:   schema.ItemParentTypeDefault,
				},
				ItemParentCacheAncestor: &query.ItemParentCacheListOptions{
					ParentID:  row.ParentID,
					StockOnly: true,
				},
			}, convertItemFields(duplicateParentFields))
			if err != nil && !errors.Is(err, items.ErrItemNotFound) {
				return nil, err
			}

			if err == nil {
				resRow.DuplicateParent, err = itemExtractor.Extract(
					ctx, duplicateRow, duplicateParentFields, lang, isModer, userID, role,
				)
				if err != nil {
					return nil, err
				}
			}
		}

		duplicateChildFields := fields.GetDuplicateChild()
		if duplicateChildFields != nil {
			duplicateRow, err := itemRepository.Item(ctx, &query.ItemListOptions{
				ExcludeID: row.ItemID,
				ItemParentParent: &query.ItemParentListOptions{
					ParentID: row.ParentID,
					Type:     row.Type,
				},
				ItemParentCacheDescendant: &query.ItemParentCacheListOptions{
					ItemID: row.ItemID,
				},
			}, convertItemFields(duplicateChildFields))
			if err != nil && !errors.Is(err, items.ErrItemNotFound) {
				return nil, err
			}

			if err == nil {
				resRow.DuplicateChild, err = itemExtractor.Extract(
					ctx, duplicateRow, duplicateChildFields, lang, isModer, userID, role,
				)
				if err != nil {
					return nil, err
				}
			}
		}

		res = append(res, resRow)
	}

	return res, nil
}
