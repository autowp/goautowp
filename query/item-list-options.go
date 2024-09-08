package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

const (
	itemParentNoParentAliasSuffix = "_ipnp"
	ItemAlias                     = "i"
)

func AppendItemAlias(alias string, suffix string) string {
	return alias + "_" + ItemAlias + suffix
}

type ItemsListOptions struct {
	Alias                     string
	Language                  string
	ItemID                    int64
	ItemIDExpr                goqu.Expression
	TypeID                    []schema.ItemTableItemTypeID
	PictureItems              *PictureItemListOptions
	PreviewPictures           *PictureItemListOptions
	Limit                     uint32
	Page                      uint32
	SortByName                bool
	ItemParentChild           *ItemParentListOptions
	ItemParentParent          *ItemParentListOptions
	ItemParentCacheDescendant *ItemParentCacheListOptions
	ItemParentCacheAncestor   *ItemParentCacheListOptions
	NoParents                 bool
	Catname                   string
	Name                      string
	IsConcept                 bool
	EngineItemID              int64
	HasBeginYear              bool
	HasEndYear                bool
	HasBeginMonth             bool
	HasEndMonth               bool
	HasLogo                   bool
	CreatedInDays             int
}

func ItemParentNoParentAlias(alias string) string {
	return alias + itemParentNoParentAliasSuffix
}

func (s *ItemsListOptions) Select(db *goqu.Database) *goqu.SelectDataset {
	alias := ItemAlias
	if s.Alias != "" {
		alias = s.Alias
	}

	sqSelect := db.Select().From(schema.ItemTable.As(alias))

	return s.Apply(alias, sqSelect)
}

func (s *ItemsListOptions) CountSelect(db *goqu.Database) *goqu.SelectDataset {
	return s.Select(db).Select(goqu.COUNT(goqu.Star()))
}

func (s *ItemsListOptions) CountDistinctSelect(db *goqu.Database) *goqu.SelectDataset {
	alias := ItemAlias
	if s.Alias != "" {
		alias = s.Alias
	}

	return s.Select(db).Select(
		goqu.COUNT(goqu.DISTINCT(goqu.T(alias).Col(schema.ItemTableIDColName))),
	)
}

func (s *ItemsListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) *goqu.SelectDataset {
	aliasTable := goqu.T(alias)
	aliasIDCol := aliasTable.Col(schema.ItemTableIDColName)

	if s.ItemID > 0 {
		sqSelect = sqSelect.Where(aliasIDCol.Eq(s.ItemID))
	}

	if s.ItemIDExpr != nil {
		sqSelect = sqSelect.Where(aliasIDCol.Eq(s.ItemIDExpr))
	}

	if len(s.TypeID) > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableItemTypeIDColName).In(s.TypeID))
	}

	if s.CreatedInDays > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableAddDatetimeColName).Gt(
			goqu.Func("DATE_SUB", goqu.Func("NOW"), goqu.L("INTERVAL ? DAY", s.CreatedInDays)),
		))
	}

	if s.ItemParentChild != nil {
		ipcAlias := AppendItemParentAlias(alias, "c")
		sqSelect = sqSelect.Join(
			schema.ItemParentTable.As(ipcAlias),
			goqu.On(aliasIDCol.Eq(goqu.T(ipcAlias).Col(schema.ItemParentTableParentIDColName))),
		)

		sqSelect = s.ItemParentChild.Apply(ipcAlias, sqSelect)
	}

	if s.ItemParentParent != nil {
		ippAlias := AppendItemParentAlias(alias, "p")
		sqSelect = sqSelect.Join(
			schema.ItemParentTable.As(ippAlias),
			goqu.On(aliasIDCol.Eq(goqu.T(ippAlias).Col(schema.ItemParentTableItemIDColName))),
		)

		sqSelect = s.ItemParentParent.Apply(ippAlias, sqSelect)
	}

	if s.PictureItems != nil {
		piAlias := AppendPictureItemAlias(alias)

		sqSelect = sqSelect.Join(
			schema.PictureItemTable.As(piAlias),
			goqu.On(aliasIDCol.Eq(goqu.T(piAlias).Col(schema.PictureItemTableItemIDColName))),
		)

		sqSelect = s.PictureItems.Apply(piAlias, sqSelect)
	}

	if s.ItemParentCacheDescendant != nil {
		ipcdAlias := AppendItemParentCacheAlias(alias, "d")
		sqSelect = sqSelect.
			Join(
				schema.ItemParentCacheTable.As(ipcdAlias),
				goqu.On(aliasIDCol.Eq(goqu.T(ipcdAlias).Col(schema.ItemParentCacheTableParentIDColName))),
			)

		sqSelect = s.ItemParentCacheDescendant.Apply(ipcdAlias, sqSelect)
	}

	if s.ItemParentCacheAncestor != nil {
		ipcaAlias := AppendItemParentCacheAlias(alias, "a")
		sqSelect = sqSelect.
			Join(
				schema.ItemParentCacheTable.As(ipcaAlias),
				goqu.On(aliasIDCol.Eq(goqu.T(ipcaAlias).Col(schema.ItemParentCacheTableItemIDColName))),
			)

		sqSelect = s.ItemParentCacheAncestor.Apply(ipcaAlias, sqSelect)
	}

	if s.NoParents {
		ipnpAlias := ItemParentNoParentAlias(alias)
		sqSelect = sqSelect.
			LeftJoin(
				schema.ItemParentTable.As(ipnpAlias),
				goqu.On(aliasIDCol.Eq(goqu.T(ipnpAlias).Col(schema.ItemParentTableItemIDColName))),
			).
			Where(goqu.T(ipnpAlias).Col(schema.ItemParentTableParentIDColName).IsNull())
	}

	if len(s.Catname) > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableCatnameColName).Eq(s.Catname))
	}

	if s.IsConcept {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableIsConceptColName))
	}

	if s.EngineItemID > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableEngineItemIDColName).Eq(s.EngineItemID))
	}

	if len(s.Name) > 0 {
		subSelect := sqSelect.ClearSelect().ClearLimit().ClearOffset().ClearOrder().ClearWhere().GroupBy().FromSelf()

		// WHERE EXISTS(SELECT item_id FROM item_language WHERE item.id = item_id AND name ILIKE ?)
		sqSelect = sqSelect.Where(
			goqu.L(
				"EXISTS ?",
				subSelect.
					From(schema.ItemLanguageTable).
					Where(
						aliasIDCol.Eq(schema.ItemLanguageTableItemIDCol),
						schema.ItemLanguageTableNameCol.ILike(s.Name),
					),
			),
		)
	}

	if s.HasBeginYear {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableBeginYearColName))
	}

	if s.HasEndYear {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableEndYearColName))
	}

	if s.HasBeginMonth {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableBeginMonthColName))
	}

	if s.HasEndMonth {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableEndMonthColName))
	}

	if s.HasLogo {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableLogoIDColName).IsNotNull())
	}

	return sqSelect
}
