package items

import (
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

type AliaseableExpression interface {
	exp.Expression
	exp.Aliaseable
	exp.Orderable
}

type Column interface {
	SelectExpr(alias string, language string) (AliaseableExpression, error)
}

type DescendantsCountColumn struct {
	db *goqu.Database
}

func (s DescendantsCountColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	options := query.ItemParentCacheListOptions{
		ParentIDExpr: goqu.T(alias).Col(schema.ItemTableIDColName),
		ExcludeSelf:  true,
	}

	return goqu.L("?", options.CountSelect(s.db)), nil
}

type NewDescendantsCountColumn struct {
	db *goqu.Database
}

func (s NewDescendantsCountColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	options := query.ItemsListOptions{
		Alias: alias + "product2",
		ItemParentCacheAncestor: &query.ItemParentCacheListOptions{
			ParentIDExpr: goqu.T(alias).Col(schema.ItemTableIDColName),
			ExcludeSelf:  true,
		},
		CreatedInDays: NewDays,
	}

	return goqu.L("?", options.CountDistinctSelect(s.db)), nil
}

type DescendantTwinsGroupsCountColumn struct {
	db *goqu.Database
}

func (s DescendantTwinsGroupsCountColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	options := query.ItemsListOptions{
		Alias:  alias + "dtgc",
		TypeID: []schema.ItemTableItemTypeID{schema.ItemTableItemTypeIDTwins},
		ItemParentCacheDescendant: &query.ItemParentCacheListOptions{
			ItemParentCacheAncestorByItemID: &query.ItemParentCacheListOptions{
				ItemsByParentID: &query.ItemsListOptions{
					ItemIDExpr: goqu.T(alias).Col(schema.ItemTableIDColName),
				},
			},
		},
	}

	return goqu.L("?", options.CountSelect(s.db)), nil
}

type DescendantPicturesCountColumn struct{}

func (s DescendantPicturesCountColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	piTableAlias := query.AppendPictureItemAlias(
		query.AppendItemParentCacheAlias(alias, "d"),
	)

	return goqu.COUNT(goqu.DISTINCT(goqu.T(piTableAlias).Col(schema.PictureItemTablePictureIDColName))), nil
}

type ChildsCountColumn struct {
	db *goqu.Database
}

func (s ChildsCountColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	options := query.ItemParentListOptions{
		ParentIDExpr: goqu.T(alias).Col(schema.ItemTableIDColName),
	}

	return goqu.L("?", options.CountSelect(s.db)), nil
}

type TextstorageRefColumn struct {
	db  *goqu.Database
	col string
}

func (s TextstorageRefColumn) SelectExpr(alias string, language string) (AliaseableExpression, error) {
	ilAlias := alias + "_" + s.col

	orderExpr, err := langPriorityOrderExpr(goqu.T(ilAlias).Col(schema.ItemLanguageTableLanguageColName), language)
	if err != nil {
		return nil, err
	}

	return goqu.L("?", s.db.Select(schema.TextstorageTextTableTextCol).
			From(schema.ItemLanguageTable.As(ilAlias)).
			Join(
				schema.TextstorageTextTable,
				goqu.On(goqu.T(ilAlias).Col(s.col).Eq(schema.TextstorageTextTableIDCol)),
			).
			Where(
				goqu.T(ilAlias).Col(schema.ItemLanguageTableItemIDColName).Eq(goqu.T(alias).Col(schema.ItemTableIDColName)),
				goqu.Func("length", schema.TextstorageTextTableTextCol).Gt(0),
			).
			Order(orderExpr).
			Limit(1)),
		nil
}

type NameOnlyColumn struct {
	db *goqu.Database
}

func (s NameOnlyColumn) SelectExpr(alias string, language string) (AliaseableExpression, error) {
	orderExpr, err := langPriorityOrderExpr(schema.ItemLanguageTableLanguageCol, language)
	if err != nil {
		return nil, err
	}

	return goqu.Func(
			"IFNULL",
			s.db.Select(schema.ItemLanguageTableNameCol).
				From(schema.ItemLanguageTable).
				Where(
					schema.ItemLanguageTableItemIDCol.Eq(goqu.T(alias).Col(schema.ItemTableIDColName)),
					goqu.Func("LENGTH", schema.ItemLanguageTableNameCol).Gt(0),
				).
				Order(orderExpr).
				Limit(1),
			goqu.T(alias).Col(schema.ItemTableNameColName),
		),
		nil
}

type CommentsAttentionsCountColumn struct {
	db *goqu.Database
}

func (s CommentsAttentionsCountColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	opts := query.CommentMessageListOptions{
		Attention:   schema.CommentMessageModeratorAttentionRequired,
		CommentType: schema.CommentMessageTypeIDPictures,
		PictureItems: &query.PictureItemListOptions{
			ItemParentCacheAncestor: &query.ItemParentCacheListOptions{
				ParentIDExpr: goqu.T(alias).Col(schema.ItemTableIDColName),
			},
		},
	}

	return goqu.L("?", opts.CountSelect(s.db)), nil
}

type InboxPicturesCountColumn struct {
	db *goqu.Database
}

func (s InboxPicturesCountColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	opts := query.PictureListOptions{
		Status: schema.PictureStatusInbox,
		PictureItem: &query.PictureItemListOptions{
			ItemParentCacheAncestor: &query.ItemParentCacheListOptions{
				ParentIDExpr: goqu.T(alias).Col(schema.ItemTableIDColName),
			},
		},
	}

	return goqu.L("?", opts.CountSelect(s.db)), nil
}

type MostsActiveColumn struct {
	mostsMinCarsCount int
	db                *goqu.Database
}

func (s MostsActiveColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	opts := query.ItemParentCacheListOptions{
		ItemsByParentID: &query.ItemsListOptions{
			ItemIDExpr: goqu.T(alias).Col(schema.ItemTableIDColName),
		},
	}

	return goqu.L("? >= ?", opts.CountSelect(s.db), s.mostsMinCarsCount), nil
}

type DescendantsParentsCountColumn struct{}

func (s DescendantsParentsCountColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	cAlias := query.AppendItemParentAlias(
		query.AppendItemParentCacheAlias(alias, "d"), "p",
	)

	return goqu.COUNT(goqu.DISTINCT(goqu.T(cAlias).Col(schema.ItemParentTableParentIDColName))), nil
}

type NewDescendantsParentsCountColumn struct{}

func (s NewDescendantsParentsCountColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	cAlias := query.AppendItemAlias(
		query.AppendItemParentAlias(
			query.AppendItemParentCacheAlias(alias, "d"), "p",
		),
		"p",
	)
	cAliasTable := goqu.T(cAlias)

	return goqu.COUNT(goqu.DISTINCT(goqu.Func("IF",
		cAliasTable.Col(schema.ItemTableAddDatetimeColName).Gt(
			goqu.Func("DATE_SUB", goqu.Func("NOW"), goqu.L("INTERVAL ? DAY", NewDays)),
		),
		cAliasTable.Col(schema.ItemTableIDColName),
		nil,
	))), nil
}

type ChildItemsCountColumn struct{}

func (s ChildItemsCountColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	ipcAlias := query.AppendItemParentAlias(alias, "c")
	ipcAliasTable := goqu.T(ipcAlias)

	return goqu.COUNT(goqu.DISTINCT(ipcAliasTable.Col(schema.ItemParentTableItemIDColName))), nil
}

type NewChildItemsCountColumn struct{}

func (s NewChildItemsCountColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	ipcAlias := query.AppendItemParentAlias(alias, "c")
	ipcAliasTable := goqu.T(ipcAlias)

	return goqu.COUNT(goqu.DISTINCT(
		goqu.Func("IF",
			ipcAliasTable.Col(schema.ItemParentTableTimestampColName).Gt(
				goqu.Func("DATE_SUB", goqu.Func("NOW"), goqu.L("INTERVAL ? DAY", NewDays)),
			),
			ipcAliasTable.Col(schema.ItemParentTableItemIDColName),
			nil,
		),
	)), nil
}

type SimpleColumn struct {
	col string
}

func (s SimpleColumn) SelectExpr(alias string, _ string) (AliaseableExpression, error) {
	return goqu.T(alias).Col(s.col), nil
}

type SpecNameColumn struct{}

func (s SpecNameColumn) SelectExpr(_ string, _ string) (AliaseableExpression, error) {
	return schema.SpecTableNameCol, nil
}

type SpecShortNameColumn struct{}

func (s SpecShortNameColumn) SelectExpr(_ string, _ string) (AliaseableExpression, error) {
	return schema.SpecTableShortNameCol, nil
}

type StarCountColumn struct{}

func (s StarCountColumn) SelectExpr(_ string, _ string) (AliaseableExpression, error) {
	return goqu.COUNT(goqu.Star()), nil
}

type ItemParentParentTimestampColumn struct{}

func (s ItemParentParentTimestampColumn) SelectExpr(_ string, _ string) (AliaseableExpression, error) {
	return goqu.MAX(
			goqu.T(query.AppendItemParentAlias(query.ItemAlias, "p")).Col(schema.ItemParentTableTimestampColName),
		),
		nil
}
