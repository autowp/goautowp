package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

const (
	pictureItemAlias = "pi"
)

type PictureItemListOptions struct {
	TypeID                  schema.PictureItemType
	PictureID               int64
	ItemID                  int64
	Pictures                *PictureListOptions
	PerspectiveID           int32
	ExcludePerspectiveID    []int32
	ItemParentCacheAncestor *ItemParentCacheListOptions
}

func AppendPictureItemAlias(alias string) string {
	return alias + "_" + pictureItemAlias
}

func (s *PictureItemListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) *goqu.SelectDataset {
	aliasTable := goqu.T(alias)

	if s.TypeID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureItemTableTypeColName).Eq(s.TypeID))
	}

	if s.PictureID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureItemTablePictureIDColName).Eq(s.PictureID))
	}

	if s.ItemID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureItemTableItemIDColName).Eq(s.ItemID))
	}

	if s.PerspectiveID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureItemTablePerspectiveIDColName).Eq(s.PerspectiveID))
	}

	if len(s.ExcludePerspectiveID) > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureItemTablePerspectiveIDColName).NotIn(s.ExcludePerspectiveID))
	}

	if s.Pictures != nil {
		pAlias := AppendPictureAlias(alias)

		sqSelect = sqSelect.Join(
			schema.PictureTable.As(pAlias),
			goqu.On(
				aliasTable.Col(schema.PictureItemTablePictureIDColName).Eq(
					goqu.T(pAlias).Col(schema.PictureTableIDColName),
				),
			),
		)

		sqSelect = s.Pictures.Apply(pAlias, sqSelect)
	}

	if s.ItemParentCacheAncestor != nil {
		ipcaAlias := AppendItemParentCacheAlias(alias, "a")
		sqSelect = sqSelect.
			Join(
				schema.ItemParentCacheTable.As(ipcaAlias),
				goqu.On(aliasTable.Col(schema.PictureItemTableItemIDColName).Eq(
					goqu.T(ipcaAlias).Col(schema.ItemParentCacheTableItemIDColName),
				)),
			)

		sqSelect = s.ItemParentCacheAncestor.Apply(ipcaAlias, sqSelect)
	}

	return sqSelect
}
