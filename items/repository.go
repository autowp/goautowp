package items

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/autowp/goautowp/filter"
	"github.com/autowp/goautowp/query"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/textstorage"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

var (
	ErrItemNotFound                    = errors.New("item not found")
	errLangNotFound                    = errors.New("language not found")
	errFieldsIsRequired                = errors.New("fields is required")
	errFieldRequires                   = errors.New("fields requires")
	errItemParentCycle                 = errors.New("cycle detected")
	errFailedToCreateItemParentCatname = errors.New("failed to create catname")
	errInvalidItemParentCombination    = errors.New("that type of parent is not allowed for this type")
	errGroupRequired                   = errors.New("only groups can have childs")
	errSelfParent                      = errors.New("self parent forbidden")
)

const (
	DefaultLanguageCode = "xx"

	NewDays                       = 7
	ItemLanguageTextMaxLength     = 4096
	ItemLanguageFullTextMaxLength = 65536

	colNameOnly                   = "name_only"
	colSpecName                   = "spec_name"
	colSpecShortName              = "spec_short_name"
	colDescription                = "description"
	colFullText                   = "full_text"
	colDescendantsParentsCount    = "descendants_parents_count"
	colNewDescendantsParentsCount = "new_descendants_parents_count"
	colDescendantsCount           = "descendants_count"
	colNewDescendantsCount        = "new_descendants_count"
	colChildItemsCount            = "child_items_count"
	colNewChildItemsCount         = "new_child_items_count"
	colDescendantPicturesCount    = "descendant_pictures_count"
	colChildsCount                = "childs_count"
	colDescendantTwinsGroupsCount = "descendant_twins_groups_count"
	colInboxPicturesCount         = "inbox_pictures_count"
	colMostsActive                = "mosts_active"
	colCommentsAttentionsCount    = "comments_attentions_count"
	colStarCount                  = "star_count"
	colItemParentParentTimestamp  = "item_parent_parent_timestamp"
)

type OrderBy int

const (
	OrderByNone OrderBy = iota
	OrderByDescendantsCount
	OrderByDescendantPicturesCount
	OrderByAddDatetime
	OrderByName
	OrderByDescendantsParentsCount
	OrderByStarCount
	OrderByItemParentParentTimestamp
)

var catnameBlacklist = []string{"sport", "tuning", "related", "pictures", "specifications"}

type TreeItem struct {
	ID       int64
	Name     string
	Childs   []TreeItem
	ItemType schema.ItemTableItemTypeID
}

var languagePriority = map[string][]string{
	DefaultLanguageCode: {"en", "it", "fr", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", "he", DefaultLanguageCode},
	"en":                {"en", "it", "fr", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", "he", DefaultLanguageCode},
	"fr":                {"fr", "en", "it", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", "he", DefaultLanguageCode},
	"pt-br":             {"pt", "en", "it", "fr", "de", "es", "ru", "be", "uk", "zh", "jp", "he", DefaultLanguageCode},
	"ru":                {"ru", "en", "it", "fr", "de", "es", "pt", "be", "uk", "zh", "jp", "he", DefaultLanguageCode},
	"be":                {"be", "ru", "uk", "en", "it", "fr", "de", "es", "pt", "zh", "jp", "he", DefaultLanguageCode},
	"uk":                {"uk", "ru", "en", "it", "fr", "de", "es", "pt", "be", "zh", "jp", "he", DefaultLanguageCode},
	"zh":                {"zh", "en", "it", "fr", "de", "es", "pt", "ru", "be", "uk", "jp", "he", DefaultLanguageCode},
	"es":                {"es", "en", "it", "fr", "de", "pt", "ru", "be", "uk", "zh", "jp", "he", DefaultLanguageCode},
	"it":                {"it", "en", "fr", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", "he", DefaultLanguageCode},
	"he":                {"he", "en", "it", "fr", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", DefaultLanguageCode},
}

// Repository Main Object.
type Repository struct {
	db                               *goqu.Database
	mostsMinCarsCount                int
	descendantsCountColumn           *DescendantsCountColumn
	newDescendantsCountColumn        *NewDescendantsCountColumn
	descendantTwinsGroupsCountColumn *DescendantTwinsGroupsCountColumn
	descendantPicturesCountColumn    *DescendantPicturesCountColumn
	childsCountColumn                *ChildsCountColumn
	descriptionColumn                *TextstorageRefColumn
	fullTextColumn                   *TextstorageRefColumn
	nameOnlyColumn                   *NameOnlyColumn
	commentsAttentionsCountColumn    *CommentsAttentionsCountColumn
	inboxPicturesCountColumn         *InboxPicturesCountColumn
	mostsActiveColumn                *MostsActiveColumn
	descendantsParentsCountColumn    *DescendantsParentsCountColumn
	newDescendantsParentsCountColumn *NewDescendantsParentsCountColumn
	childItemsCountColumn            *ChildItemsCountColumn
	newChildItemsCountColumn         *NewChildItemsCountColumn
	logoColumn                       *SimpleColumn
	fullNameColumn                   *SimpleColumn
	idColumn                         *SimpleColumn
	catnameColumn                    *SimpleColumn
	engineItemIDColumn               *SimpleColumn
	itemTypeIDColumn                 *SimpleColumn
	isGroupColumn                    *SimpleColumn
	isConceptColumn                  *SimpleColumn
	isConceptInheritColumn           *SimpleColumn
	specIDColumn                     *SimpleColumn
	beginYearColumn                  *SimpleColumn
	endYearColumn                    *SimpleColumn
	beginMonthColumn                 *SimpleColumn
	endMonthColumn                   *SimpleColumn
	beginModelYearColumn             *SimpleColumn
	endModelYearColumn               *SimpleColumn
	beginModelYearFractionColumn     *SimpleColumn
	endModelYearFractionColumn       *SimpleColumn
	todayColumn                      *SimpleColumn
	bodyColumn                       *SimpleColumn
	addDatetimeColumn                *SimpleColumn
	beginOrderCacheColumn            *SimpleColumn
	endOrderCacheColumn              *SimpleColumn
	nameColumn                       *SimpleColumn
	specNameColumn                   *SpecNameColumn
	specShortNameColumn              *SpecShortNameColumn
	starCountColumn                  *StarCountColumn
	itemParentParentTimestampColumn  *ItemParentParentTimestampColumn
	contentLanguages                 []string
	textStorageRepository            *textstorage.Repository
}

type Item struct {
	schema.ItemRow
	NameOnly                   string
	DescendantsParentsCount    int32
	NewDescendantsParentsCount int32
	ChildItemsCount            int32
	NewChildItemsCount         int32
	DescendantsCount           int32
	NewDescendantsCount        int32
	SpecName                   string
	SpecShortName              string
	Description                string
	FullText                   string
	DescendantPicturesCount    int32
	ChildsCount                int32
	DescendantTwinsGroupsCount int32
	InboxPicturesCount         int32
	FullName                   string
	MostsActive                bool
	CommentsAttentionsCount    int32
}

type ItemLanguage struct {
	ItemID     int64
	Language   string
	Name       string
	TextID     int32
	FullTextID int32
}

type ItemParentLanguage struct {
	ItemID   int64
	ParentID int64
	Language string
	Name     string
}

// NewRepository constructor.
func NewRepository(
	db *goqu.Database,
	mostsMinCarsCount int,
	contentLanguages []string,
	textStorageRepository *textstorage.Repository,
) *Repository {
	return &Repository{
		db:                               db,
		mostsMinCarsCount:                mostsMinCarsCount,
		descendantsCountColumn:           &DescendantsCountColumn{db: db},
		newDescendantsCountColumn:        &NewDescendantsCountColumn{db: db},
		descendantTwinsGroupsCountColumn: &DescendantTwinsGroupsCountColumn{db: db},
		descendantPicturesCountColumn:    &DescendantPicturesCountColumn{},
		childsCountColumn:                &ChildsCountColumn{db: db},
		descriptionColumn: &TextstorageRefColumn{
			db:  db,
			col: schema.ItemLanguageTableTextIDColName,
		},
		fullTextColumn: &TextstorageRefColumn{
			db:  db,
			col: schema.ItemLanguageTableFullTextIDColName,
		},
		nameOnlyColumn:                &NameOnlyColumn{db: db},
		commentsAttentionsCountColumn: &CommentsAttentionsCountColumn{db: db},
		inboxPicturesCountColumn:      &InboxPicturesCountColumn{db: db},
		mostsActiveColumn: &MostsActiveColumn{
			db:                db,
			mostsMinCarsCount: mostsMinCarsCount,
		},
		descendantsParentsCountColumn:    &DescendantsParentsCountColumn{},
		newDescendantsParentsCountColumn: &NewDescendantsParentsCountColumn{},
		childItemsCountColumn:            &ChildItemsCountColumn{},
		newChildItemsCountColumn:         &NewChildItemsCountColumn{},
		idColumn:                         &SimpleColumn{col: schema.ItemTableIDColName},
		logoColumn:                       &SimpleColumn{col: schema.ItemTableLogoIDColName},
		fullNameColumn:                   &SimpleColumn{col: schema.ItemTableFullNameColName},
		catnameColumn:                    &SimpleColumn{col: schema.ItemTableCatnameColName},
		engineItemIDColumn:               &SimpleColumn{col: schema.ItemTableEngineItemIDColName},
		itemTypeIDColumn:                 &SimpleColumn{col: schema.ItemTableItemTypeIDColName},
		isGroupColumn:                    &SimpleColumn{col: schema.ItemTableIsGroupColName},
		isConceptColumn:                  &SimpleColumn{col: schema.ItemTableIsConceptColName},
		isConceptInheritColumn:           &SimpleColumn{col: schema.ItemTableIsConceptInheritColName},
		specIDColumn:                     &SimpleColumn{col: schema.ItemTableSpecIDColName},
		beginYearColumn:                  &SimpleColumn{col: schema.ItemTableBeginYearColName},
		endYearColumn:                    &SimpleColumn{col: schema.ItemTableEndYearColName},
		beginMonthColumn:                 &SimpleColumn{col: schema.ItemTableBeginMonthColName},
		endMonthColumn:                   &SimpleColumn{col: schema.ItemTableEndMonthColName},
		beginModelYearColumn:             &SimpleColumn{col: schema.ItemTableBeginModelYearColName},
		endModelYearColumn:               &SimpleColumn{col: schema.ItemTableEndModelYearColName},
		beginModelYearFractionColumn:     &SimpleColumn{col: schema.ItemTableBeginModelYearFractionColName},
		endModelYearFractionColumn:       &SimpleColumn{col: schema.ItemTableEndModelYearFractionColName},
		todayColumn:                      &SimpleColumn{col: schema.ItemTableTodayColName},
		bodyColumn:                       &SimpleColumn{col: schema.ItemTableBodyColName},
		addDatetimeColumn:                &SimpleColumn{col: schema.ItemTableAddDatetimeColName},
		beginOrderCacheColumn:            &SimpleColumn{col: schema.ItemTableBeginOrderCacheColName},
		endOrderCacheColumn:              &SimpleColumn{col: schema.ItemTableEndOrderCacheColName},
		nameColumn:                       &SimpleColumn{col: schema.ItemTableNameColName},
		specNameColumn:                   &SpecNameColumn{},
		specShortNameColumn:              &SpecShortNameColumn{},
		starCountColumn:                  &StarCountColumn{},
		itemParentParentTimestampColumn:  &ItemParentParentTimestampColumn{},
		contentLanguages:                 contentLanguages,
		textStorageRepository:            textStorageRepository,
	}
}

type ListPreviewPicturesPictureFields struct {
	NameText bool
}

type ListPreviewPicturesFields struct {
	Route   bool
	Picture ListPreviewPicturesPictureFields
}

type ListFields struct {
	NameOnly                   bool
	NameHTML                   bool
	NameDefault                bool
	Description                bool
	FullText                   bool
	HasText                    bool
	PreviewPictures            ListPreviewPicturesFields
	TotalPictures              bool
	ChildItemsCount            bool
	NewChildItemsCount         bool
	DescendantsCount           bool
	NewDescendantsCount        bool
	NameText                   bool
	DescendantPicturesCount    bool
	ChildsCount                bool
	DescendantTwinsGroupsCount bool
	InboxPicturesCount         bool
	FullName                   bool
	Logo                       bool
	MostsActive                bool
	CommentsAttentionsCount    bool
	DescendantsParentsCount    bool
	NewDescendantsParentsCount bool
}

func yearsPrefix(begin int32, end int32) string {
	if begin <= 0 && end <= 0 {
		return ""
	}

	if end == begin {
		return strconv.Itoa(int(begin))
	}

	const oneHundred = 100

	var (
		bms = begin / oneHundred
		ems = end / oneHundred
	)

	if bms == ems {
		return fmt.Sprintf("%d–%02d", begin, end%oneHundred)
	}

	if begin <= 0 {
		return fmt.Sprintf("xx–%d", end)
	}

	if end > 0 {
		return fmt.Sprintf("%d–%d", begin, end)
	}

	return fmt.Sprintf("%d–xx", begin)
}

func langPriorityOrderExpr( //nolint: ireturn
	col exp.IdentifierExpression, language string,
) (exp.OrderedExpression, error) {
	langPriority, ok := languagePriority[language]
	if !ok {
		langPriority, ok = languagePriority[DefaultLanguageCode]
	}

	if !ok {
		return nil, fmt.Errorf("%w: `%s`", errLangNotFound, language)
	}

	langs := make([]interface{}, len(langPriority)+1)
	langs[0] = col

	for i, v := range langPriority {
		langs[i+1] = v
	}

	return goqu.Func("FIELD", langs...).Asc(), nil
}

func (s *Repository) columnsByFields(fields ListFields) map[string]Column {
	columns := map[string]Column{
		schema.ItemTableIDColName:               s.idColumn,
		schema.ItemTableCatnameColName:          s.catnameColumn,
		schema.ItemTableEngineItemIDColName:     s.engineItemIDColumn,
		schema.ItemTableItemTypeIDColName:       s.itemTypeIDColumn,
		schema.ItemTableIsConceptColName:        s.isConceptColumn,
		schema.ItemTableIsConceptInheritColName: s.isConceptInheritColumn,
		schema.ItemTableSpecIDColName:           s.specIDColumn,
		schema.ItemTableIsGroupColName:          s.isGroupColumn,
	}

	if fields.FullName {
		columns[schema.ItemTableFullNameColName] = s.fullNameColumn
	}

	if fields.Logo {
		columns[schema.ItemTableLogoIDColName] = s.logoColumn
	}

	if fields.NameText || fields.NameHTML {
		columns[schema.ItemTableBeginYearColName] = s.beginYearColumn
		columns[schema.ItemTableEndYearColName] = s.endYearColumn
		columns[schema.ItemTableBeginMonthColName] = s.beginMonthColumn
		columns[schema.ItemTableEndMonthColName] = s.endMonthColumn
		columns[schema.ItemTableBeginModelYearColName] = s.beginModelYearColumn
		columns[schema.ItemTableEndModelYearColName] = s.endModelYearColumn
		columns[schema.ItemTableBeginModelYearFractionColName] = s.beginModelYearFractionColumn
		columns[schema.ItemTableEndModelYearFractionColName] = s.endModelYearFractionColumn
		columns[schema.ItemTableTodayColName] = s.todayColumn
		columns[schema.ItemTableBodyColName] = s.bodyColumn
		columns[colSpecShortName] = s.specShortNameColumn

		if fields.NameHTML {
			columns[colSpecName] = s.specNameColumn
		}
	}

	if fields.Description {
		columns[colDescription] = s.descriptionColumn
	}

	if fields.FullText {
		columns[colFullText] = s.fullTextColumn
	}

	if fields.NameOnly || fields.NameText || fields.NameHTML {
		columns[colNameOnly] = s.nameOnlyColumn
	}

	if fields.ChildItemsCount {
		columns[colChildItemsCount] = s.childItemsCountColumn
	}

	if fields.NewChildItemsCount {
		columns[colNewChildItemsCount] = s.newChildItemsCountColumn
	}

	if fields.DescendantsParentsCount {
		columns[colDescendantsParentsCount] = s.descendantsParentsCountColumn
	}

	if fields.NewDescendantsParentsCount {
		columns[colNewDescendantsParentsCount] = s.newDescendantsParentsCountColumn
	}

	if fields.ChildsCount {
		columns[colChildsCount] = s.childsCountColumn
	}

	if fields.DescendantsCount {
		columns[colDescendantsCount] = s.descendantsCountColumn
	}

	if fields.NewDescendantsCount {
		columns[colNewDescendantsCount] = s.newDescendantsCountColumn
	}

	if fields.DescendantTwinsGroupsCount {
		columns[colDescendantTwinsGroupsCount] = s.descendantTwinsGroupsCountColumn
	}

	if fields.DescendantPicturesCount {
		columns[colDescendantPicturesCount] = s.descendantPicturesCountColumn
	}

	if fields.MostsActive {
		columns[colMostsActive] = s.mostsActiveColumn
	}

	if fields.InboxPicturesCount {
		columns[colInboxPicturesCount] = s.inboxPicturesCountColumn
	}

	if fields.CommentsAttentionsCount {
		columns[colCommentsAttentionsCount] = s.commentsAttentionsCountColumn
	}

	return columns
}

func (s *Repository) IDsSelect(options query.ItemsListOptions) (*goqu.SelectDataset, error) {
	alias := query.ItemAlias
	if options.Alias != "" {
		alias = options.Alias
	}

	sqSelect := s.db.Select(goqu.I(alias).Col(schema.ItemTableIDColName)).From(schema.ItemTable.As(alias))

	return options.Apply(alias, sqSelect), nil
}

func (s *Repository) IDs(ctx context.Context, options query.ItemsListOptions) ([]int64, error) {
	var err error

	sqSelect, err := s.IDsSelect(options)
	if err != nil {
		return nil, err
	}

	var ids []int64

	err = sqSelect.Executor().ScanValsContext(ctx, &ids)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (s *Repository) Count(ctx context.Context, options query.ItemsListOptions) (int, error) {
	var count int

	success, err := options.CountSelect(s.db).Executor().ScanValContext(ctx, &count)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	return count, nil
}

func (s *Repository) CountDistinct(ctx context.Context, options query.ItemsListOptions) (int, error) {
	var count int

	success, err := options.CountDistinctSelect(s.db).Executor().ScanValContext(ctx, &count)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	return count, nil
}

func (s *Repository) Item(ctx context.Context, options query.ItemsListOptions, fields ListFields) (Item, error) {
	options.Limit = 1

	res, _, err := s.List(ctx, options, fields, OrderByNone, false)
	if err != nil {
		return Item{}, err
	}

	if len(res) == 0 {
		return Item{}, ErrItemNotFound
	}

	return res[0], nil
}

func (s *Repository) isFieldsValid(options query.ItemsListOptions, fields ListFields) error {
	if (fields.ChildItemsCount || fields.NewChildItemsCount) && options.ItemParentChild == nil {
		return fmt.Errorf("%w: ChildItemsCount, NewChildItemsCount requires ItemParentChild", errFieldRequires)
	}

	if fields.DescendantPicturesCount && (options.ItemParentCacheDescendant == nil ||
		options.ItemParentCacheDescendant.PictureItemsByItemID == nil) {
		return fmt.Errorf(
			"%w: DescendantPicturesCount requires ItemParentCacheDescendant.PictureItemsByItemID",
			errFieldRequires,
		)
	}

	if (fields.DescendantsParentsCount || fields.NewDescendantsParentsCount) &&
		(options.ItemParentCacheDescendant == nil || options.ItemParentCacheDescendant.ItemParentByItemID == nil) {
		return fmt.Errorf(
			"%w: (New)DescendantsParentsCount requires ItemParentCacheDescendant.ItemParentByItemID",
			errFieldRequires,
		)
	}

	return nil
}

func (s *Repository) orderBy(alias string, orderBy OrderBy, language string) ([]exp.OrderedExpression, error) {
	type columnOrder struct {
		col Column
		asc bool
	}

	var columns []columnOrder

	switch orderBy {
	case OrderByDescendantsCount:
		columns = []columnOrder{{col: s.descendantsCountColumn, asc: false}}
	case OrderByDescendantPicturesCount:
		columns = []columnOrder{{col: s.descendantPicturesCountColumn, asc: false}}
	case OrderByAddDatetime:
		columns = []columnOrder{{col: s.addDatetimeColumn, asc: false}}
	case OrderByName:
		columns = []columnOrder{
			{col: s.nameColumn, asc: true},
			{col: s.bodyColumn, asc: true},
			{col: s.specIDColumn, asc: true},
			{col: s.beginOrderCacheColumn, asc: true},
			{col: s.endOrderCacheColumn, asc: true},
		}
	case OrderByDescendantsParentsCount:
		columns = []columnOrder{{col: s.descendantsParentsCountColumn, asc: false}}
	case OrderByStarCount:
		columns = []columnOrder{{col: s.starCountColumn, asc: false}}
	case OrderByItemParentParentTimestamp:
		columns = []columnOrder{{col: s.itemParentParentTimestampColumn, asc: false}}
	case OrderByNone:
	}

	orderByExp := make([]exp.OrderedExpression, 0, len(columns))

	for _, column := range columns {
		expr, err := column.col.SelectExpr(alias, language)
		if err != nil {
			return nil, err
		}

		ordExpr := expr.Desc()
		if column.asc {
			ordExpr = expr.Asc()
		}

		orderByExp = append(orderByExp, ordExpr)
	}

	return orderByExp, nil
}

func (s *Repository) wrapperOrderBy(wrapperAlias string, wrappedAlias string, orderBy OrderBy) []exp.OrderedExpression {
	wrapperAliasTable := goqu.T(wrapperAlias)
	wrappedAliasTable := goqu.T(wrappedAlias)

	switch orderBy {
	case OrderByDescendantsCount:
		return []exp.OrderedExpression{wrappedAliasTable.Col(colDescendantsCount).Desc()}
	case OrderByDescendantPicturesCount:
		return []exp.OrderedExpression{wrappedAliasTable.Col(colDescendantPicturesCount).Desc()}
	case OrderByAddDatetime:
		return []exp.OrderedExpression{wrapperAliasTable.Col(schema.ItemTableAddDatetimeColName).Desc()}
	case OrderByName:
		return []exp.OrderedExpression{
			wrapperAliasTable.Col(schema.ItemTableNameColName).Asc(),
			wrapperAliasTable.Col(schema.ItemTableBodyColName).Asc(),
			wrapperAliasTable.Col(schema.ItemTableSpecIDColName).Asc(),
			wrapperAliasTable.Col(schema.ItemTableBeginOrderCacheColName).Asc(),
			wrapperAliasTable.Col(schema.ItemTableEndOrderCacheColName).Asc(),
		}
	case OrderByDescendantsParentsCount:
		return []exp.OrderedExpression{wrappedAliasTable.Col(colDescendantsParentsCount).Desc()}
	case OrderByStarCount:
		return []exp.OrderedExpression{wrappedAliasTable.Col(colStarCount).Desc()}
	case OrderByItemParentParentTimestamp:
		return []exp.OrderedExpression{wrappedAliasTable.Col(colItemParentParentTimestamp).Desc()}
	case OrderByNone:
	}

	return nil
}

func (s *Repository) wrappedOrderBy(alias string, orderBy OrderBy) []exp.OrderedExpression {
	aliasTable := goqu.T(alias)

	var orderByExp []exp.OrderedExpression

	switch orderBy {
	case OrderByDescendantsCount:
		orderByExp = []exp.OrderedExpression{goqu.C(colDescendantsCount).Desc()}
	case OrderByDescendantPicturesCount:
		orderByExp = []exp.OrderedExpression{goqu.C(colDescendantPicturesCount).Desc()}
	case OrderByAddDatetime:
		orderByExp = []exp.OrderedExpression{aliasTable.Col(schema.ItemTableAddDatetimeColName).Desc()}
	case OrderByName:
		orderByExp = []exp.OrderedExpression{
			aliasTable.Col(schema.ItemTableNameColName).Asc(),
			aliasTable.Col(schema.ItemTableBodyColName).Asc(),
			aliasTable.Col(schema.ItemTableSpecIDColName).Asc(),
			aliasTable.Col(schema.ItemTableBeginOrderCacheColName).Asc(),
			aliasTable.Col(schema.ItemTableEndOrderCacheColName).Asc(),
		}
	case OrderByDescendantsParentsCount:
		orderByExp = []exp.OrderedExpression{goqu.C(colDescendantsParentsCount).Desc()}
	case OrderByStarCount:
		orderByExp = []exp.OrderedExpression{goqu.C(colStarCount).Desc()}
	case OrderByItemParentParentTimestamp:
		orderByExp = []exp.OrderedExpression{goqu.C(colItemParentParentTimestamp).Desc()}
	case OrderByNone:
	}

	return orderByExp
}

func (s *Repository) wrappedSelectColumns(orderBy OrderBy) map[string]Column {
	columns := map[string]Column{
		schema.ItemTableIDColName: s.idColumn,
	}

	switch orderBy {
	case OrderByDescendantsCount:
		columns[colDescendantsCount] = s.descendantsCountColumn
	case OrderByDescendantPicturesCount:
		columns[colDescendantPicturesCount] = s.descendantPicturesCountColumn
	case OrderByDescendantsParentsCount:
		columns[colDescendantsParentsCount] = s.descendantsParentsCountColumn
	case OrderByStarCount:
		columns[colStarCount] = s.starCountColumn
	case OrderByItemParentParentTimestamp:
		columns[colItemParentParentTimestamp] = s.itemParentParentTimestampColumn
	case OrderByName, OrderByAddDatetime, OrderByNone:
	}

	return columns
}

func (s *Repository) List( //nolint:maintidx
	ctx context.Context, options query.ItemsListOptions, fields ListFields, orderBy OrderBy,
	pagination bool,
) ([]Item, *util.Pages, error) {
	var err error

	aliasTable := goqu.T(query.ItemAlias)

	err = s.isFieldsValid(options, fields)
	if err != nil {
		return nil, nil, err
	}

	if options.SortByName && !fields.NameOnly {
		return nil, nil, fmt.Errorf("%w: NameOnly for SortByName", errFieldsIsRequired)
	}

	sqSelect := options.Select(s.db).GroupBy(aliasTable.Col(schema.ItemTableIDColName))

	var pages *util.Pages

	outAlias := query.ItemAlias

	if options.Limit > 0 {
		wrappedOrderBy := s.wrappedOrderBy(query.ItemAlias, orderBy)

		if len(wrappedOrderBy) > 0 {
			sqSelect = sqSelect.Order(wrappedOrderBy...)
		}

		paginator := util.Paginator{
			SQLSelect:         sqSelect,
			ItemCountPerPage:  int32(options.Limit), //nolint: gosec
			CurrentPageNumber: int32(options.Page),  //nolint: gosec
		}

		if pagination {
			pages, err = paginator.GetPages(ctx)
			if err != nil {
				return nil, nil, err
			}
		}

		sqSelect, err = paginator.GetCurrentItems(ctx)
		if err != nil {
			return nil, nil, err
		}

		// implements deferred join pattern
		wrappedAlias := "wrapped"
		wrappedIDCol := goqu.T(wrappedAlias).Col(schema.ItemTableIDColName)
		wrappedColumns := s.wrappedSelectColumns(orderBy)
		wrappedColumnsExpr := make([]interface{}, 0, len(wrappedColumns))

		for _, mapItem := range toSortedColumns(wrappedColumns) {
			expr, err := mapItem.col.SelectExpr(query.ItemAlias, options.Language)
			if err != nil {
				return nil, nil, err
			}

			wrappedColumnsExpr = append(wrappedColumnsExpr, expr.As(mapItem.key))
		}

		wrapperColumns := s.columnsByFields(fields)
		wrapperColumnsExpr := make([]interface{}, 0, len(wrapperColumns))

		for _, mapItem := range toSortedColumns(wrapperColumns) {
			var expr AliaseableExpression

			_, isWrapped := wrappedColumns[mapItem.key]
			if isWrapped && mapItem.key != schema.ItemTableIDColName {
				expr = goqu.T(wrappedAlias).Col(mapItem.key)
			} else {
				expr, err = mapItem.col.SelectExpr(schema.ItemTableName, options.Language)
				if err != nil {
					return nil, nil, err
				}
			}

			wrapperColumnsExpr = append(wrapperColumnsExpr, expr.As(mapItem.key))
		}

		options.Alias = schema.ItemTableName

		sqSelect = options.Select(s.db).Select(wrapperColumnsExpr...).
			From(schema.ItemTable).
			Join(sqSelect.Select(wrappedColumnsExpr...).As(wrappedAlias), goqu.On(
				schema.ItemTableIDCol.Eq(wrappedIDCol),
			)).
			GroupBy(schema.ItemTableIDCol)

		wrapperOrderBy := s.wrapperOrderBy(schema.ItemTableName, wrappedAlias, orderBy)

		if len(wrapperOrderBy) > 0 {
			sqSelect = sqSelect.Order(wrapperOrderBy...)
		}

		outAlias = schema.ItemTableName
	} else {
		orderByExpr, err := s.orderBy(query.ItemAlias, orderBy, options.Language)
		if err != nil {
			return nil, nil, err
		}

		if len(orderByExpr) > 0 {
			sqSelect = sqSelect.Order(orderByExpr...)
		}

		columns := s.columnsByFields(fields)

		res := make([]interface{}, 0, len(columns))

		for _, mapItem := range toSortedColumns(columns) {
			expr, err := mapItem.col.SelectExpr(query.ItemAlias, options.Language)
			if err != nil {
				return nil, nil, err
			}

			res = append(res, expr.As(mapItem.key))
		}

		sqSelect = sqSelect.Select(res...)
	}

	outTable := goqu.T(outAlias)

	if fields.NameText || fields.NameHTML {
		sqSelect = sqSelect.LeftJoin(
			schema.SpecTable,
			goqu.On(outTable.Col(schema.ItemTableSpecIDColName).Eq(schema.SpecTableIDCol)),
		)
	}

	rows, err := sqSelect.Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if err != nil {
		return nil, nil, err
	}
	defer util.Close(rows)

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	var result []Item

	for rows.Next() {
		var row Item

		var (
			specName      sql.NullString
			specShortName sql.NullString
			description   sql.NullString
			fullText      sql.NullString
			fullName      sql.NullString
		)

		pointers := make([]interface{}, len(columnNames))

		for i, colName := range columnNames {
			switch colName {
			case schema.ItemTableIDColName:
				pointers[i] = &row.ID
			case colNameOnly:
				pointers[i] = &row.NameOnly
			case schema.ItemTableCatnameColName:
				pointers[i] = &row.Catname
			case schema.ItemTableFullNameColName:
				pointers[i] = &fullName
			case schema.ItemTableEngineItemIDColName:
				pointers[i] = &row.EngineItemID
			case schema.ItemTableItemTypeIDColName:
				pointers[i] = &row.ItemTypeID
			case schema.ItemTableIsGroupColName:
				pointers[i] = &row.IsGroup
			case schema.ItemTableIsConceptColName:
				pointers[i] = &row.IsConcept
			case schema.ItemTableIsConceptInheritColName:
				pointers[i] = &row.IsConceptInherit
			case schema.ItemTableSpecIDColName:
				pointers[i] = &row.SpecID
			case colDescription:
				pointers[i] = &description
			case colFullText:
				pointers[i] = &fullText
			case colDescendantsParentsCount:
				pointers[i] = &row.DescendantsParentsCount
			case colNewDescendantsParentsCount:
				pointers[i] = &row.NewDescendantsParentsCount
			case colDescendantsCount:
				pointers[i] = &row.DescendantsCount
			case colNewDescendantsCount:
				pointers[i] = &row.NewDescendantsCount
			case colChildItemsCount:
				pointers[i] = &row.ChildItemsCount
			case colNewChildItemsCount:
				pointers[i] = &row.NewChildItemsCount
			case schema.ItemTableBeginYearColName:
				pointers[i] = &row.BeginYear
			case schema.ItemTableEndYearColName:
				pointers[i] = &row.EndYear
			case schema.ItemTableBeginMonthColName:
				pointers[i] = &row.BeginMonth
			case schema.ItemTableEndMonthColName:
				pointers[i] = &row.EndMonth
			case schema.ItemTableBeginModelYearColName:
				pointers[i] = &row.BeginModelYear
			case schema.ItemTableEndModelYearColName:
				pointers[i] = &row.EndModelYear
			case schema.ItemTableBeginModelYearFractionColName:
				pointers[i] = &row.BeginModelYearFraction
			case schema.ItemTableEndModelYearFractionColName:
				pointers[i] = &row.EndModelYearFraction
			case schema.ItemTableTodayColName:
				pointers[i] = &row.Today
			case schema.ItemTableBodyColName:
				pointers[i] = &row.Body
			case colSpecName:
				pointers[i] = &specName
			case colSpecShortName:
				pointers[i] = &specShortName
			case colDescendantPicturesCount:
				pointers[i] = &row.DescendantPicturesCount
			case colChildsCount:
				pointers[i] = &row.ChildsCount
			case colDescendantTwinsGroupsCount:
				pointers[i] = &row.DescendantTwinsGroupsCount
			case colInboxPicturesCount:
				pointers[i] = &row.InboxPicturesCount
			case schema.ItemTableLogoIDColName:
				pointers[i] = &row.LogoID
			case colMostsActive:
				pointers[i] = &row.MostsActive
			case colCommentsAttentionsCount:
				pointers[i] = &row.CommentsAttentionsCount
			default:
				pointers[i] = nil
			}
		}

		err = rows.Scan(pointers...)
		if err != nil {
			return nil, nil, err
		}

		if specName.Valid {
			row.SpecName = specName.String
		}

		if specShortName.Valid {
			row.SpecShortName = specShortName.String
		}

		if description.Valid {
			row.Description = description.String
		}

		if fullText.Valid {
			row.FullText = fullText.String
		}

		if fullName.Valid {
			row.FullName = fullName.String
		}

		result = append(result, row)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	if options.SortByName {
		tag := language.English

		switch options.Language {
		case "ru":
			tag = language.Russian
		case "zh":
			tag = language.SimplifiedChinese
		case "fr":
			tag = language.French
		case "es":
			tag = language.Spanish
		case "uk":
			tag = language.Ukrainian
		case "be":
			tag = language.Russian
		case "pt-br":
			tag = language.BrazilianPortuguese
		case "he":
			tag = language.Hebrew
		}

		cyrillic := regexp.MustCompile(`^\p{Cyrillic}`)
		han := regexp.MustCompile(`^\p{Han}`)

		cl := collate.New(tag, collate.IgnoreCase, collate.IgnoreDiacritics)

		sort.SliceStable(result, func(i, j int) bool {
			iName := result[i].NameOnly
			jName := result[j].NameOnly

			switch options.Language {
			case "ru", "uk", "be":
				aIsCyrillic := cyrillic.MatchString(iName)
				bIsCyrillic := cyrillic.MatchString(jName)

				if aIsCyrillic && !bIsCyrillic {
					return true
				}

				if bIsCyrillic && !aIsCyrillic {
					return false
				}
			case "zh":
				aIsHan := han.MatchString(iName)
				bIsHan := han.MatchString(jName)

				if aIsHan && !bIsHan {
					return true
				}

				if bIsHan && !aIsHan {
					return false
				}
			}

			return cl.CompareString(iName, jName) == -1
		})
	}

	return result, pages, nil
}

func (s *Repository) Tree(ctx context.Context, id string) (*TreeItem, error) {
	type row struct {
		ID       int64                      `db:"id"`
		Name     string                     `db:"name"`
		ItemType schema.ItemTableItemTypeID `db:"item_type_id"`
	}

	var item row

	success, err := s.db.Select(schema.ItemTableIDCol, schema.ItemTableNameCol, schema.ItemTableItemTypeIDCol).
		From(schema.ItemTable).
		Where(schema.ItemTableIDCol.Eq(id)).
		ScanStructContext(ctx, item)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, nil //nolint: nilnil
	}

	return &TreeItem{
		ID:       item.ID,
		Name:     item.Name,
		ItemType: item.ItemType,
	}, nil
}

func (s *Repository) AddItemVehicleType(ctx context.Context, itemID int64, vehicleTypeID int64) error {
	changed, err := s.setItemVehicleTypeRow(ctx, itemID, vehicleTypeID, false)
	if err != nil {
		return err
	}

	if changed {
		err = s.RefreshItemVehicleTypeInheritanceFromParents(ctx, itemID)
		if err != nil {
			return err
		}

		err = s.refreshItemVehicleTypeInheritance(ctx, itemID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) RemoveItemVehicleType(ctx context.Context, itemID int64, vehicleTypeID int64) error {
	res, err := s.db.From(schema.VehicleVehicleTypeTable).Delete().
		Where(
			schema.VehicleVehicleTypeTableVehicleIDCol.Eq(itemID),
			schema.VehicleVehicleTypeTableVehicleTypeIDCol.Eq(vehicleTypeID),
			schema.VehicleVehicleTypeTableInheritedCol.IsFalse(),
		).Executor().Exec()
	if err != nil {
		return err
	}

	deleted, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if deleted > 0 {
		err = s.RefreshItemVehicleTypeInheritanceFromParents(ctx, itemID)
		if err != nil {
			return err
		}

		err = s.refreshItemVehicleTypeInheritance(ctx, itemID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) setItemVehicleTypeRow(
	ctx context.Context,
	itemID int64,
	vehicleTypeID int64,
	inherited bool,
) (bool, error) {
	res, err := s.db.Insert(schema.VehicleVehicleTypeTable).Rows(goqu.Record{
		schema.VehicleVehicleTypeTableVehicleIDColName:     itemID,
		schema.VehicleVehicleTypeTableVehicleTypeIDColName: vehicleTypeID,
		schema.VehicleVehicleTypeTableInheritedColName:     inherited,
	}).OnConflict(goqu.DoUpdate(
		schema.VehicleVehicleTypeTableVehicleIDColName+","+schema.VehicleVehicleTypeTableVehicleTypeIDColName,
		goqu.Record{
			schema.VehicleVehicleTypeTableInheritedColName: goqu.Func(
				"VALUES",
				goqu.C(schema.VehicleVehicleTypeTableInheritedColName),
			),
		},
	)).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (s *Repository) RefreshItemVehicleTypeInheritanceFromParents(ctx context.Context, itemID int64) error {
	typeIDs, err := s.getItemVehicleTypeIDs(ctx, itemID, false)
	if err != nil {
		return err
	}

	if len(typeIDs) > 0 {
		// do not inherit when own value
		res, err := s.db.Delete(schema.VehicleVehicleTypeTable).Where(
			schema.VehicleVehicleTypeTableVehicleIDCol.Eq(itemID),
			schema.VehicleVehicleTypeTableInheritedCol.IsTrue(),
		).Executor().ExecContext(ctx)
		if err != nil {
			return err
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}

		if affected > 0 {
			err = s.refreshItemVehicleTypeInheritance(ctx, itemID)
			if err != nil {
				return err
			}
		}

		return nil
	}

	types, err := s.getItemVehicleTypeInheritedIDs(ctx, itemID)
	if err != nil {
		return err
	}

	changed, err := s.setItemVehicleTypeRows(ctx, itemID, types, true)
	if err != nil {
		return err
	}

	if changed {
		err := s.refreshItemVehicleTypeInheritance(ctx, itemID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) refreshItemVehicleTypeInheritance(ctx context.Context, itemID int64) error {
	var ids []int64

	err := s.db.Select(schema.ItemParentTableItemIDCol).
		From(schema.ItemParentTable).
		Where(schema.ItemParentTableParentIDCol.Eq(itemID)).
		ScanValsContext(ctx, &ids)
	if err != nil {
		return err
	}

	for _, childID := range ids {
		err = s.RefreshItemVehicleTypeInheritanceFromParents(ctx, childID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) getItemVehicleTypeIDs(ctx context.Context, itemID int64, inherited bool) ([]int64, error) {
	sqlSelect := s.db.From(schema.VehicleVehicleTypeTable).
		Select(schema.VehicleVehicleTypeTableVehicleTypeIDCol).
		Where(schema.VehicleVehicleTypeTableVehicleIDCol.Eq(itemID))
	if inherited {
		sqlSelect = sqlSelect.Where(schema.VehicleVehicleTypeTableInheritedCol.IsTrue())
	} else {
		sqlSelect = sqlSelect.Where(schema.VehicleVehicleTypeTableInheritedCol.IsFalse())
	}

	res := make([]int64, 0)

	err := sqlSelect.ScanValsContext(ctx, &res)

	return res, err
}

func (s *Repository) getItemVehicleTypeInheritedIDs(ctx context.Context, itemID int64) ([]int64, error) {
	sqlSelect := s.db.From(schema.VehicleVehicleTypeTable).
		Select(schema.VehicleVehicleTypeTableVehicleTypeIDCol).Distinct().
		Join(
			schema.ItemParentTable,
			goqu.On(schema.VehicleVehicleTypeTableVehicleIDCol.Eq(schema.ItemParentTableParentIDCol)),
		).
		Where(schema.ItemParentTableItemIDCol.Eq(itemID))

	res := make([]int64, 0)

	err := sqlSelect.ScanValsContext(ctx, &res)

	return res, err
}

func (s *Repository) setItemVehicleTypeRows(
	ctx context.Context,
	itemID int64,
	types []int64,
	inherited bool,
) (bool, error) {
	changed := false

	for _, t := range types {
		rowChanged, err := s.setItemVehicleTypeRow(ctx, itemID, t, inherited)
		if err != nil {
			return false, err
		}

		if rowChanged {
			changed = true
		}
	}

	sqlDelete := s.db.From(schema.VehicleVehicleTypeTable).Delete().
		Where(schema.VehicleVehicleTypeTableVehicleIDCol.Eq(itemID))

	if len(types) > 0 {
		sqlDelete = sqlDelete.Where(schema.VehicleVehicleTypeTableVehicleTypeIDCol.NotIn(types))
	}

	res, err := sqlDelete.Executor().Exec()
	if err != nil {
		return false, err
	}

	deleted, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	if deleted > 0 {
		changed = true
	}

	return changed, nil
}

type parentInfo struct {
	Diff   int64
	Tuning bool
	Sport  bool
	Design bool
}

func (s *Repository) collectParentInfo(ctx context.Context, id int64, diff int64) (map[int64]parentInfo, error) {
	//nolint: sqlclosecheck
	rows, err := s.db.Select(schema.ItemParentTableParentIDCol, schema.ItemParentTableTypeCol).
		From(schema.ItemParentTable).
		Where(schema.ItemParentTableItemIDCol.Eq(id)).
		Executor().QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	result := make(map[int64]parentInfo, 0)

	for rows.Next() {
		var (
			parentID int64
			typeID   schema.ItemParentType
		)

		err = rows.Scan(&parentID, &typeID)
		if err != nil {
			return nil, err
		}

		isTuning := typeID == schema.ItemParentTypeTuning
		isSport := typeID == schema.ItemParentTypeSport
		isDesign := typeID == schema.ItemParentTypeDesign
		result[parentID] = parentInfo{
			Diff:   diff,
			Tuning: isTuning,
			Sport:  isSport,
			Design: isDesign,
		}

		parentInfos, err := s.collectParentInfo(ctx, parentID, diff+1)
		if err != nil {
			return nil, err
		}

		for pid, info := range parentInfos {
			val, ok := result[pid]

			if !ok || info.Diff < val.Diff {
				val = info
				val.Tuning = result[pid].Tuning || isTuning
				val.Sport = result[pid].Sport || isSport
				val.Design = result[pid].Design || isDesign
				result[pid] = val
			}
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Repository) getChildItemsIDs(ctx context.Context, parentID int64) ([]int64, error) {
	vals := make([]int64, 0)

	err := s.db.Select(schema.ItemParentTableItemIDCol).
		From(schema.ItemParentTable).
		Where(schema.ItemParentTableParentIDCol.Eq(parentID)).
		Executor().ScanValsContext(ctx, &vals)
	if err != nil {
		return nil, err
	}

	return vals, nil
}

func (s *Repository) RebuildCache(ctx context.Context, itemID int64) (int64, error) {
	parentInfos, err := s.collectParentInfo(ctx, itemID, 1)
	if err != nil {
		return 0, err
	}

	parentInfos[itemID] = parentInfo{
		Diff:   0,
		Tuning: false,
		Sport:  false,
		Design: false,
	}

	var (
		updates int64
		records = make([]goqu.Record, len(parentInfos))
		idx     = 0
	)

	for parentID, info := range parentInfos {
		records[idx] = goqu.Record{
			schema.ItemParentCacheTableItemIDColName:   itemID,
			schema.ItemParentCacheTableParentIDColName: parentID,
			schema.ItemParentCacheTableDiffColName:     info.Diff,
			schema.ItemParentCacheTableTuningColName:   info.Tuning,
			schema.ItemParentCacheTableSportColName:    info.Sport,
			schema.ItemParentCacheTableDesignColName:   info.Design,
		}
		idx++
	}

	result, err := s.db.Insert(schema.ItemParentCacheTable).
		Rows(records).
		OnConflict(
			goqu.DoUpdate(schema.ItemParentCacheTableItemIDColName+","+schema.ItemParentCacheTableParentIDColName,
				goqu.Record{
					schema.ItemParentCacheTableDiffColName:   goqu.Func("VALUES", goqu.C(schema.ItemParentCacheTableDiffColName)),
					schema.ItemParentCacheTableTuningColName: goqu.Func("VALUES", goqu.C(schema.ItemParentCacheTableTuningColName)),
					schema.ItemParentCacheTableSportColName:  goqu.Func("VALUES", goqu.C(schema.ItemParentCacheTableSportColName)),
					schema.ItemParentCacheTableDesignColName: goqu.Func("VALUES", goqu.C(schema.ItemParentCacheTableDesignColName)),
				},
			),
		).
		Executor().ExecContext(ctx)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	updates += affected

	keys := make([]int64, len(parentInfos))

	i := 0

	for k := range parentInfos {
		keys[i] = k
		i++
	}

	_, err = s.db.Delete(schema.ItemParentCacheTable).Where(
		schema.ItemParentCacheTableItemIDCol.Eq(itemID),
		schema.ItemParentCacheTableParentIDCol.NotIn(keys),
	).Executor().ExecContext(ctx)
	if err != nil {
		return 0, err
	}

	childs, err := s.getChildItemsIDs(ctx, itemID)
	if err != nil {
		return 0, err
	}

	for _, child := range childs {
		affected, err = s.RebuildCache(ctx, child)
		if err != nil {
			return 0, err
		}

		updates += affected
	}

	return updates, nil
}

func (s *Repository) LanguageList(ctx context.Context, itemID int64) ([]ItemLanguage, error) {
	sqSelect := s.db.Select(schema.ItemLanguageTableItemIDCol, schema.ItemLanguageTableLanguageCol,
		schema.ItemLanguageTableNameCol, schema.ItemLanguageTableTextIDCol, schema.ItemLanguageTableFullTextIDCol).
		From(schema.ItemLanguageTable).Where(
		schema.ItemLanguageTableItemIDCol.Eq(itemID),
		schema.ItemLanguageTableLanguageCol.Neq(DefaultLanguageCode),
	)

	rows, err := sqSelect.Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	var result []ItemLanguage

	for rows.Next() {
		var (
			row            ItemLanguage
			nullName       sql.NullString
			nullTextID     sql.NullInt32
			nullFullTextID sql.NullInt32
		)

		err = rows.Scan(&row.ItemID, &row.Language, &nullName, &nullTextID, &nullFullTextID)
		if err != nil {
			return nil, err
		}

		row.Name = ""
		if nullName.Valid {
			row.Name = nullName.String
		}

		row.TextID = 0
		if nullTextID.Valid {
			row.TextID = nullTextID.Int32
		}

		row.FullTextID = 0
		if nullFullTextID.Valid {
			row.FullTextID = nullFullTextID.Int32
		}

		result = append(result, row)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Repository) UpdateItemLanguage(
	ctx context.Context, itemID int64, lang, name, text, fullText string, userID int64,
) ([]string, error) {
	var row schema.ItemLanguageRow

	success, err := s.db.Select(
		schema.ItemLanguageTableNameCol, schema.ItemLanguageTableTextIDCol, schema.ItemLanguageTableFullTextIDCol,
	).
		From(schema.ItemLanguageTable).
		Where(
			schema.ItemLanguageTableItemIDCol.Eq(itemID),
			schema.ItemLanguageTableLanguageCol.Eq(lang),
		).ScanStructContext(ctx, &row)
	if err != nil {
		return nil, err
	}

	if !success {
		row = schema.ItemLanguageRow{}
	}

	set := goqu.Record{}

	changes := make([]string, 0)

	oldName := ""
	if row.Name.Valid {
		oldName = row.Name.String
	}

	if oldName != name {
		set[schema.ItemLanguageTableNameColName] = name

		changes = append(changes, "moder/vehicle/name")
	}

	textChanged := false

	if row.TextID.Valid {
		oldText, err := s.textStorageRepository.Text(ctx, row.TextID.Int32)
		if err != nil {
			return nil, err
		}

		textChanged = text != oldText

		err = s.textStorageRepository.SetText(ctx, row.TextID.Int32, text, userID)
		if err != nil {
			return nil, err
		}
	} else if len(text) > 0 {
		textChanged = true

		textID, err := s.textStorageRepository.CreateText(ctx, text, userID)
		if err != nil {
			return nil, err
		}

		set[schema.ItemLanguageTableTextIDColName] = textID
	}

	if textChanged {
		changes = append(changes, "moder/item/short-description")
	}

	fullTextChanged := false

	if row.FullTextID.Valid {
		oldFullText, err := s.textStorageRepository.Text(ctx, row.FullTextID.Int32)
		if err != nil {
			return nil, err
		}

		fullTextChanged = fullText != oldFullText

		err = s.textStorageRepository.SetText(ctx, row.FullTextID.Int32, fullText, userID)
		if err != nil {
			return nil, err
		}
	} else if len(fullText) > 0 {
		fullTextChanged = true

		fullTextID, err := s.textStorageRepository.CreateText(ctx, fullText, userID)
		if err != nil {
			return nil, err
		}

		set[schema.ItemLanguageTableFullTextIDColName] = fullTextID
	}

	if fullTextChanged {
		changes = append(changes, "moder/item/full-description")
	}

	if len(set) > 0 {
		onConflict := goqu.Record{}
		for col := range set {
			onConflict[col] = goqu.Func("VALUES", goqu.C(col))
		}

		set[schema.ItemLanguageTableItemIDColName] = itemID
		set[schema.ItemLanguageTableLanguageColName] = lang

		_, err = s.db.Insert(schema.ItemLanguageTable).Rows(set).OnConflict(goqu.DoUpdate(
			schema.ItemLanguageTableItemIDColName+","+schema.ItemLanguageTableLanguageColName,
			onConflict,
		)).Executor().ExecContext(ctx)
		if err != nil {
			return nil, err
		}

		_, err = s.refreshAutoByVehicle(ctx, itemID)
		if err != nil {
			return nil, err
		}
	}

	return changes, nil
}

func (s *Repository) ParentLanguageList(
	ctx context.Context, itemID int64, parentID int64,
) ([]ItemParentLanguage, error) {
	sqSelect := s.db.Select(schema.ItemParentLanguageTableItemIDCol, schema.ItemParentLanguageTableParentIDCol,
		schema.ItemParentLanguageTableLanguageCol, schema.ItemParentLanguageTableNameCol).
		From(schema.ItemParentLanguageTable).
		Where(
			schema.ItemParentLanguageTableItemIDCol.Eq(itemID),
			schema.ItemParentLanguageTableParentIDCol.Eq(parentID),
		)

	rows, err := sqSelect.Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	var result []ItemParentLanguage

	for rows.Next() {
		var row ItemParentLanguage

		err = rows.Scan(&row.ItemID, &row.ParentID, &row.Language, &row.Name)
		if err != nil {
			return nil, err
		}

		result = append(result, row)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Repository) ItemsWithPicturesCount(
	ctx context.Context, nPictures int,
) (int32, error) {
	const countAlias = "c"

	sqSelect := s.db.From(
		s.db.Select(schema.ItemTableIDCol, goqu.COUNT(schema.PictureTableIDCol).As(countAlias)).
			From(schema.ItemTable).
			Join(schema.PictureItemTable, goqu.On(schema.ItemTableIDCol.Eq(schema.PictureItemTable.Col("item_id")))).
			Join(schema.PictureTable, goqu.On(schema.PictureItemTable.Col("picture_id").Eq(schema.PictureTableIDCol))).
			GroupBy(schema.ItemTableIDCol).
			Having(goqu.C(countAlias).Gte(nPictures)).
			As("T1"),
	)

	result, err := sqSelect.CountContext(ctx)
	if err != nil {
		return 0, err
	}

	return int32(result), nil //nolint: gosec
}

func (s *Repository) SetItemParentLanguage(
	ctx context.Context, parentID int64, itemID int64, language string, newName string, forceIsAuto bool,
) error {
	bvlRow := struct {
		IsAuto bool   `db:"is_auto"`
		Name   string `db:"name"`
	}{}

	success, err := s.db.From(schema.ItemParentLanguageTable).Where(
		schema.ItemParentLanguageTableParentIDCol.Eq(parentID),
		schema.ItemParentLanguageTableItemIDCol.Eq(itemID),
		schema.ItemParentLanguageTableLanguageCol.Eq(language),
	).ScanStructContext(ctx, &bvlRow)
	if err != nil {
		return err
	}

	isAuto := true

	if !forceIsAuto {
		name := ""

		if success {
			isAuto = bvlRow.IsAuto
			name = bvlRow.Name
		}

		if name != newName {
			isAuto = false
		}
	}

	if len(newName) == 0 {
		parentRow := schema.ItemRow{}
		itmRow := schema.ItemRow{}

		success, err = s.db.Select(
			schema.ItemTableIDCol, schema.ItemTableNameCol, schema.ItemTableBodyCol, schema.ItemTableSpecIDCol,
			schema.ItemTableBeginYearCol, schema.ItemTableEndYearCol,
			schema.ItemTableBeginModelYearCol, schema.ItemTableEndModelYearCol,
		).
			From(schema.ItemTable).
			Where(schema.ItemTableIDCol.Eq(parentID)).
			ScanStructContext(ctx, &parentRow)
		if err != nil {
			return err
		}

		if !success {
			return ErrItemNotFound
		}

		success, err = s.db.Select(
			schema.ItemTableIDCol, schema.ItemTableNameCol, schema.ItemTableBodyCol, schema.ItemTableSpecIDCol,
			schema.ItemTableBeginYearCol, schema.ItemTableEndYearCol,
			schema.ItemTableBeginModelYearCol, schema.ItemTableEndModelYearCol,
		).
			From(schema.ItemTable).
			Where(schema.ItemTableIDCol.Eq(itemID)).
			ScanStructContext(ctx, &itmRow)
		if err != nil {
			return err
		}

		if !success {
			return ErrItemNotFound
		}

		newName, err = s.extractName(ctx, parentRow, itmRow, language)
		if err != nil {
			return err
		}

		isAuto = true
	}

	if len(newName) > schema.ItemLanguageNameMaxLength {
		newName = newName[:schema.ItemLanguageNameMaxLength]
	}

	_, err = s.db.Insert(schema.ItemParentLanguageTable).Rows(goqu.Record{
		schema.ItemParentLanguageTableItemIDColName:   itemID,
		schema.ItemParentLanguageTableParentIDColName: parentID,
		schema.ItemParentLanguageTableLanguageColName: language,
		schema.ItemParentLanguageTableNameColName:     newName,
		schema.ItemParentLanguageTableIsAutoColName:   isAuto,
	}).OnConflict(
		goqu.DoUpdate(schema.ItemParentLanguageTableNameColName+","+schema.ItemParentLanguageTableIsAutoColName,
			goqu.Record{
				schema.ItemParentLanguageTableNameColName: goqu.Func(
					"VALUES",
					goqu.C(schema.ItemParentLanguageTableNameColName),
				),
				schema.ItemParentLanguageTableIsAutoColName: goqu.Func(
					"VALUES",
					goqu.C(schema.ItemParentLanguageTableIsAutoColName),
				),
			},
		)).Executor().ExecContext(ctx)

	return err
}

func (s *Repository) namePreferLanguage(ctx context.Context, parentID, itemID int64, language string) (string, error) {
	res := ""

	success, err := s.db.Select(schema.ItemParentLanguageTableNameCol).
		From(schema.ItemParentLanguageTable).
		Where(
			schema.ItemParentLanguageTableItemIDCol.Eq(itemID),
			schema.ItemParentLanguageTableParentIDCol.Eq(parentID),
			goqu.Func("LENGTH", schema.ItemParentLanguageTableNameCol).Gt(0),
		).
		Order(goqu.L("? > 0", schema.ItemParentLanguageTableLanguageCol.Eq(language)).Desc()).
		ScanValContext(ctx, &res)
	if err != nil {
		return "", err
	}

	if !success {
		return "", nil
	}

	return res, nil
}

func (s *Repository) extractCatname(ctx context.Context, brandRow, vehicleRow schema.ItemRow) (string, error) {
	var err error

	diffName, err := s.namePreferLanguage(ctx, brandRow.ID, vehicleRow.ID, "en")
	if err != nil {
		return "", err
	}

	if len(diffName) == 0 {
		diffName, err = s.extractName(ctx, brandRow, vehicleRow, "en")
		if err != nil {
			return "", err
		}
	}

	catnameTemplate := filter.SanitizeFilename(diffName)

	i := 0
	allowed := false
	catname := ""

	for !allowed {
		catname = catnameTemplate
		if i > 0 {
			catname = catname + "_" + strconv.Itoa(i)
		}

		allowed, err = s.isAllowedCatname(ctx, vehicleRow.ID, brandRow.ID, catname)
		if err != nil {
			return "", err
		}

		i++
	}

	return catname, nil
}

func (s *Repository) extractName(
	ctx context.Context, parentRow schema.ItemRow, vehicleRow schema.ItemRow, language string,
) (string, error) {
	langName, err := s.getName(ctx, vehicleRow.ID, language)
	if err != nil {
		return "", err
	}

	vehicleName := langName
	if len(langName) == 0 {
		vehicleName = vehicleRow.Name
	}

	aliases, err := s.getAliases(ctx, parentRow.ID)
	if err != nil {
		return "", err
	}

	name := vehicleName

	for _, alias := range aliases {
		patterns := []string{
			"by The " + alias + " Company",
			"by " + alias,
			"di " + alias,
			"par " + alias,
			alias + "-",
			"-" + alias,
		}

		for _, pattern := range patterns {
			re := regexp.MustCompile(regexp.QuoteMeta(pattern))
			name = re.ReplaceAllString(name, "")
		}

		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(alias) + `\b`)
		name = re.ReplaceAllString(name, "")
	}

	re := regexp.MustCompile("[[:space:]]+")
	name = strings.TrimSpace(re.ReplaceAllString(name, " "))

	name = strings.TrimLeft(name, "/")
	if len(name) == 0 && len(vehicleRow.Body) > 0 && vehicleRow.Body != parentRow.Body {
		name = vehicleRow.Body
	}

	vbmy := vehicleRow.BeginModelYear.Int32
	vemy := vehicleRow.EndModelYear.Int32

	if len(name) == 0 && vehicleRow.BeginModelYear.Valid && vbmy > 0 {
		modelYearsDifferent := vbmy != parentRow.BeginModelYear.Int32 || vemy != parentRow.EndModelYear.Int32
		if modelYearsDifferent {
			name = yearsPrefix(vbmy, vemy)
		}
	}

	vby := vehicleRow.BeginYear.Int32
	vey := vehicleRow.EndYear.Int32

	if len(name) == 0 && vehicleRow.BeginYear.Valid && vby > 0 {
		yearsDifferent := vby != parentRow.BeginYear.Int32 || vey != parentRow.EndYear.Int32
		if yearsDifferent {
			name = yearsPrefix(vby, vey)
		}
	}

	if len(name) == 0 && vehicleRow.SpecID.Valid {
		specsDifferent := vehicleRow.SpecID.Int32 != parentRow.SpecID.Int32
		if specsDifferent {
			specShortName := ""

			success, err := s.db.Select(schema.SpecTableShortNameCol).From(schema.SpecTable).
				Where(schema.SpecTableIDCol.Eq(vehicleRow.SpecID.Int32)).ScanValContext(ctx, &specShortName)
			if err != nil {
				return "", err
			}

			if success {
				name = specShortName
			}
		}
	}

	if len(name) == 0 {
		name = vehicleName
	}

	return name, nil
}

func (s *Repository) getAliases(ctx context.Context, itemID int64) ([]string, error) {
	var aliases []string

	err := s.db.Select(schema.BrandAliasTableNameCol).From(schema.BrandAliasTable).
		Where(schema.BrandAliasTableItemIDCol.Eq(itemID)).ScanValsContext(ctx, &aliases)
	if err != nil {
		return nil, err
	}

	langNames, err := s.getNames(ctx, itemID)
	if err != nil {
		return nil, err
	}

	aliases = append(aliases, langNames...)

	sort.Slice(aliases, func(i, j int) bool {
		return len(aliases[i]) > len(aliases[j])
	})

	return aliases, nil
}

func (s *Repository) getName(ctx context.Context, itemID int64, language string) (string, error) {
	langPriority, ok := languagePriority[language]
	if !ok {
		langPriority, ok = languagePriority[DefaultLanguageCode]
	}

	if !ok {
		return "", fmt.Errorf("%w: `%s`", errLangNotFound, language)
	}

	fieldParams := make([]interface{}, len(langPriority)+1)
	fieldParams[0] = language

	for i, v := range langPriority {
		fieldParams[i+1] = v
	}

	result := ""

	success, err := s.db.Select(schema.ItemLanguageTableNameCol).
		From(schema.ItemLanguageTable).
		Where(
			schema.ItemLanguageTableItemIDCol.Eq(itemID),
			goqu.L("? > 0", goqu.Func("length", schema.ItemLanguageTableNameCol)),
		).
		Order(goqu.Func("FIELD", fieldParams...).Asc()).
		Limit(1).
		ScanValContext(ctx, &result)
	if err != nil {
		return "", err
	}

	if !success {
		return "", nil
	}

	return result, nil
}

func (s *Repository) getNames(ctx context.Context, itemID int64) ([]string, error) {
	var result []string

	err := s.db.Select(schema.ItemLanguageTableNameCol).From(schema.ItemLanguageTable).
		Where(
			schema.ItemLanguageTableItemIDCol.Eq(itemID),
			goqu.L("? > 0", goqu.Func("length", schema.ItemLanguageTableNameCol)),
		).ScanValsContext(ctx, &result)

	return result, err
}

type sortedColumnMapItem struct {
	key string
	col Column
}

func toSortedColumns(cols map[string]Column) []sortedColumnMapItem {
	res := make([]sortedColumnMapItem, 0, len(cols))
	for colName, col := range cols {
		res = append(res, sortedColumnMapItem{key: colName, col: col})
	}

	sort.SliceStable(res, func(i, j int) bool {
		return res[i].key < res[j].key
	})

	return res
}

func (s *Repository) isAllowedCombination(itemTypeID, parentItemTypeID schema.ItemTableItemTypeID) bool {
	allowed := map[schema.ItemTableItemTypeID][]schema.ItemTableItemTypeID{
		schema.ItemTableItemTypeIDVehicle: {schema.ItemTableItemTypeIDVehicle},
		schema.ItemTableItemTypeIDEngine:  {schema.ItemTableItemTypeIDEngine},
		schema.ItemTableItemTypeIDCategory: {
			schema.ItemTableItemTypeIDVehicle,
			schema.ItemTableItemTypeIDCategory,
			schema.ItemTableItemTypeIDBrand,
		},
		schema.ItemTableItemTypeIDTwins: {schema.ItemTableItemTypeIDVehicle},
		schema.ItemTableItemTypeIDBrand: {
			schema.ItemTableItemTypeIDBrand,
			schema.ItemTableItemTypeIDVehicle,
			schema.ItemTableItemTypeIDEngine,
		},
		schema.ItemTableItemTypeIDFactory: {
			schema.ItemTableItemTypeIDVehicle,
			schema.ItemTableItemTypeIDEngine,
		},
		schema.ItemTableItemTypeIDPerson:    {},
		schema.ItemTableItemTypeIDCopyright: {},
		schema.ItemTableItemTypeIDMuseum:    {},
	}

	itemTypes, ok := allowed[parentItemTypeID]
	if !ok {
		return false
	}

	return util.Contains(itemTypes, itemTypeID)
}

func (s *Repository) isAllowedCatname(ctx context.Context, itemID, parentID int64, catname string) (bool, error) {
	if len(catname) == 0 {
		return false, nil
	}

	if util.Contains(catnameBlacklist, catname) {
		return false, nil
	}

	var exists bool

	success, err := s.db.Select(goqu.V(true)).From(schema.ItemParentTable).Where(
		schema.ItemParentTableParentIDCol.Eq(parentID),
		schema.ItemParentTableCatnameCol.Eq(catname),
		schema.ItemParentTableItemIDCol.Neq(itemID),
	).ScanValContext(ctx, &exists)
	if err != nil {
		return false, err
	}

	return !(success && exists), nil
}

func (s *Repository) collectAncestorsIDs(ctx context.Context, id int64) ([]int64, error) {
	var (
		toCheck = []int64{id}
		ids     []int64
	)

	for len(toCheck) > 0 {
		ids = append(ids, toCheck...)

		var res []int64

		err := s.db.Select(schema.ItemParentTableParentIDCol).
			From(schema.ItemParentTable).
			Where(schema.ItemParentTableItemIDCol.In(toCheck)).
			ScanValsContext(ctx, &res)
		if err != nil {
			return nil, err
		}

		toCheck = res
	}

	return util.RemoveDuplicate(ids), nil
}

func (s *Repository) setItemParentLanguages(
	ctx context.Context, parentID, itemID int64, values map[string]schema.ItemParentLanguageRow, forceIsAuto bool,
) error {
	for _, lang := range s.contentLanguages {
		name := ""
		if _, ok := values[lang]; ok {
			name = values[lang].Name
		}

		err := s.SetItemParentLanguage(ctx, parentID, itemID, lang, name, forceIsAuto)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Repository) CreateItemParent(
	ctx context.Context, itemID, parentID int64, typeID schema.ItemParentType, catname string,
) (bool, error) {
	if itemID == parentID {
		return false, errSelfParent
	}

	parentRow, err := s.Item(ctx, query.ItemsListOptions{ItemID: parentID}, ListFields{})
	if err != nil {
		return false, err
	}

	itemRow, err := s.Item(ctx, query.ItemsListOptions{ItemID: itemID}, ListFields{})
	if err != nil {
		return false, err
	}

	if !parentRow.IsGroup {
		return false, errGroupRequired
	}

	if !s.isAllowedCombination(itemRow.ItemTypeID, parentRow.ItemTypeID) {
		return false, fmt.Errorf("%w: %d/%d", errInvalidItemParentCombination, itemRow.ItemTypeID, parentRow.ItemTypeID)
	}

	if len(catname) > 0 {
		allowed, err := s.isAllowedCatname(ctx, itemID, parentID, catname)
		if err != nil {
			return false, err
		}

		if !allowed {
			catname = ""
		}
	}

	manualCatname := len(catname) > 0 && catname != "_"

	if !manualCatname {
		catname, err = s.extractCatname(ctx, parentRow.ItemRow, itemRow.ItemRow)
		if err != nil {
			return false, err
		}

		if len(catname) == 0 {
			return false, errFailedToCreateItemParentCatname
		}
	}

	parentIDs, err := s.collectAncestorsIDs(ctx, parentID)
	if err != nil {
		return false, err
	}

	if util.Contains(parentIDs, itemID) {
		return false, errItemParentCycle
	}

	exists := false

	success, err := s.db.Select(goqu.V(true)).From(schema.ItemParentTable).Where(
		schema.ItemParentTableItemIDCol.Eq(itemID),
		schema.ItemParentTableParentIDCol.Eq(parentID),
	).ScanValContext(ctx, &exists)
	if err != nil {
		return false, err
	}

	if success && exists {
		return false, nil
	}

	_, err = s.db.Insert(schema.ItemParentTable).Rows(goqu.Record{
		schema.ItemParentTableParentIDColName:      parentID,
		schema.ItemParentTableItemIDColName:        itemID,
		schema.ItemParentTableTypeColName:          typeID,
		schema.ItemParentTableCatnameColName:       catname,
		schema.ItemParentTableManualCatnameColName: manualCatname,
		schema.ItemParentTableTimestampColName:     goqu.Func("NOW"),
	}).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	values := make(map[string]schema.ItemParentLanguageRow)

	for _, lang := range s.contentLanguages {
		name, err := s.extractName(ctx, parentRow.ItemRow, itemRow.ItemRow, lang)
		if err != nil {
			return false, err
		}

		values[lang] = schema.ItemParentLanguageRow{
			Name: name,
		}
	}

	err = s.setItemParentLanguages(ctx, parentID, itemID, values, true)
	if err != nil {
		return false, err
	}

	_, err = s.RebuildCache(ctx, itemID)

	return err == nil, err
}

func (s *Repository) UpdateItemParent(
	ctx context.Context, itemID, parentID int64, typeID schema.ItemParentType, catname string, forceIsAuto bool,
) (bool, error) {
	var itemParentRow schema.ItemParentRow

	success, err := s.db.Select(
		schema.ItemParentTableCatnameCol,
		schema.ItemParentTableManualCatnameCol,
	).
		From(schema.ItemParentTable).Where(
		schema.ItemParentTableParentIDCol.Eq(parentID),
		schema.ItemParentTableItemIDCol.Eq(itemID),
	).ScanStructContext(ctx, &itemParentRow)
	if err != nil {
		return false, err
	}

	if !success {
		return false, nil
	}

	var isAuto bool

	if forceIsAuto {
		isAuto = true
	} else {
		isAuto = !itemParentRow.ManualCatname
		if itemParentRow.Catname != catname {
			isAuto = false
		}
	}

	if len(catname) == 0 || catname == "_" || util.Contains(catnameBlacklist, catname) {
		parentRow, err := s.Item(ctx, query.ItemsListOptions{ItemID: parentID}, ListFields{NameText: true})
		if err != nil {
			return false, err
		}

		itemRow, err := s.Item(ctx, query.ItemsListOptions{ItemID: itemID}, ListFields{NameText: true})
		if err != nil {
			return false, err
		}

		catname, err = s.extractCatname(ctx, parentRow.ItemRow, itemRow.ItemRow)
		if err != nil {
			return false, err
		}

		isAuto = true
	}

	res, err := s.db.Update(schema.ItemParentTable).Set(goqu.Record{
		schema.ItemParentTableTypeColName:          typeID,
		schema.ItemParentTableCatnameColName:       catname,
		schema.ItemParentTableManualCatnameColName: !isAuto,
	}).Where(
		schema.ItemParentTableParentIDCol.Eq(parentID),
		schema.ItemParentTableItemIDCol.Eq(itemID),
	).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	if affected > 0 {
		_, err = s.RebuildCache(ctx, itemID)

		return true, err
	}

	return false, nil
}

func (s *Repository) RemoveItemParent(ctx context.Context, itemID, parentID int64) error {
	res, err := s.db.Delete(schema.ItemParentTable).Where(
		schema.ItemParentTableItemIDCol.Eq(itemID),
		schema.ItemParentTableParentIDCol.Eq(parentID),
	).Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	affectedItemParentRows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	res, err = s.db.Delete(schema.ItemParentLanguageTable).Where(
		schema.ItemParentLanguageTableItemIDCol.Eq(itemID),
		schema.ItemParentLanguageTableParentIDCol.Eq(parentID),
	).Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	affectedItemParentLanguageRows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affectedItemParentRows > 0 || affectedItemParentLanguageRows > 0 {
		_, err = s.RebuildCache(ctx, itemID)
	}

	return err
}

func (s *Repository) UpdateInheritance(ctx context.Context, itemID int64) error {
	var item schema.ItemRow

	success, err := s.db.Select(
		schema.ItemTableIDCol, schema.ItemTableIsConceptCol, schema.ItemTableIsConceptInheritCol,
		schema.ItemTableEngineInheritCol, schema.ItemTableCarTypeInheritCol, schema.ItemTableCarTypeIDCol,
		schema.ItemTableSpecInheritCol, schema.ItemTableSpecIDCol,
	).
		From(schema.ItemTable).
		Where(schema.ItemTableIDCol.Eq(itemID)).
		ScanStructContext(ctx, &item)
	if err != nil {
		return err
	}

	if !success {
		return ErrItemNotFound
	}

	return s.updateItemInheritance(ctx, item)
}

func (s *Repository) updateItemInheritance(ctx context.Context, car schema.ItemRow) error {
	var parents []schema.ItemRow

	err := s.db.Select(
		schema.ItemTableIsConceptCol, schema.ItemTableEngineItemIDCol, schema.ItemTableCarTypeIDCol,
		schema.ItemTableSpecIDCol,
	).
		From(schema.ItemTable).
		Join(schema.ItemParentTable, goqu.On(schema.ItemTableIDCol.Eq(schema.ItemParentTableParentIDCol))).
		Where(schema.ItemParentTableItemIDCol.Eq(car.ID)).
		ScanStructsContext(ctx, &parents)
	if err != nil {
		return err
	}

	somethingChanged := false

	set := goqu.Record{}

	if car.IsConceptInherit {
		isConcept := false

		for _, parent := range parents {
			if parent.IsConcept {
				isConcept = true

				break
			}
		}

		if car.IsConcept != isConcept {
			set[schema.ItemTableIsConceptColName] = isConcept
			somethingChanged = true
		}
	}

	if car.EngineInherit {
		enginesMap := make(map[int64]int)

		for _, parent := range parents {
			engineID := parent.EngineItemID
			if engineID.Valid {
				enginesMap[engineID.Int64]++
			}
		}

		// select top
		selectedID := util.KeyOfMapMaxValue(enginesMap)

		var oldEngineID int64
		if !car.EngineItemID.Valid {
			oldEngineID = car.EngineItemID.Int64
		}

		if oldEngineID != selectedID {
			set[schema.ItemTableEngineItemIDColName] = sql.NullInt64{
				Int64: selectedID,
				Valid: selectedID > 0,
			}
			somethingChanged = true
		}
	}

	if car.CarTypeInherit {
		carTypesMap := make(map[int64]int)

		for _, parent := range parents {
			typeID := parent.CarTypeID
			if typeID.Valid {
				carTypesMap[typeID.Int64]++
			}
		}

		for id, count := range carTypesMap {
			otherIDs := make([]int64, 0, len(carTypesMap))

			for i := range carTypesMap {
				if id != i {
					otherIDs = append(otherIDs, i)
				}
			}

			isParentOf, err := s.getChildVehicleTypesByWhitelist(ctx, id, otherIDs)
			if err != nil {
				return err
			}

			if len(isParentOf) > 0 {
				for _, childID := range isParentOf {
					carTypesMap[childID] += count
				}

				delete(carTypesMap, id)
			}
		}

		// select top
		selectedID := util.KeyOfMapMaxValue(carTypesMap)

		var oldCarTypeID int64
		if car.CarTypeID.Valid {
			oldCarTypeID = car.CarTypeID.Int64
		}

		if oldCarTypeID != selectedID {
			set[schema.ItemTableCarTypeIDColName] = sql.NullInt64{
				Int64: selectedID,
				Valid: selectedID > 0,
			}
			somethingChanged = true
		}
	}

	if car.SpecInherit {
		specsMap := make(map[int32]int)

		for _, parent := range parents {
			specID := parent.SpecID
			if specID.Valid {
				specsMap[specID.Int32]++
			}
		}

		// select top
		selectedID := util.KeyOfMapMaxValue(specsMap)

		var oldSpecID int32
		if car.SpecID.Valid {
			oldSpecID = car.SpecID.Int32
		}

		if oldSpecID != selectedID {
			set[schema.ItemTableSpecIDColName] = sql.NullInt32{
				Int32: selectedID,
				Valid: selectedID > 0,
			}
			somethingChanged = true
		}
	}

	if somethingChanged || !car.CarTypeInherit {
		if len(set) > 0 {
			_, err = s.db.Update(schema.ItemTable).Set(set).
				Where(schema.ItemTableIDCol.Eq(car.ID)).
				Executor().ExecContext(ctx)
			if err != nil {
				return err
			}
		}

		var childItems []schema.ItemRow

		err = s.db.Select(
			schema.ItemTableIDCol, schema.ItemTableIsConceptCol, schema.ItemTableIsConceptInheritCol,
			schema.ItemTableEngineInheritCol, schema.ItemTableCarTypeInheritCol, schema.ItemTableCarTypeIDCol,
			schema.ItemTableSpecInheritCol, schema.ItemTableSpecIDCol,
		).
			From(schema.ItemTable).
			Join(schema.ItemParentTable, goqu.On(schema.ItemTableIDCol.Eq(schema.ItemParentTableItemIDCol))).
			Where(schema.ItemParentTableParentIDCol.Eq(car.ID)).
			ScanStructsContext(ctx, &childItems)
		if err != nil {
			return err
		}

		for _, child := range childItems {
			err = s.updateItemInheritance(ctx, child)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Repository) getChildVehicleTypesByWhitelist(
	ctx context.Context, parentID int64, whitelist []int64,
) ([]int64, error) {
	res := make([]int64, 0)

	if len(whitelist) == 0 {
		return res, nil
	}

	var ids []int64

	err := s.db.Select(schema.CarTypesParentsTableIDCol).
		From(schema.CarTypesParentsTable).
		Where(
			schema.CarTypesParentsTableIDCol.In(whitelist),
			schema.CarTypesParentsTableParentIDCol.Eq(parentID),
			schema.CarTypesParentsTableIDCol.Neq(schema.CarTypesParentsTableParentIDCol),
		).ScanValsContext(ctx, &ids)
	if err != nil {
		return res, err
	}

	return ids, nil
}

func (s *Repository) MoveItemParent(ctx context.Context, itemID, parentID, newParentID int64) (bool, error) {
	oldParentRow, err := s.Item(ctx, query.ItemsListOptions{ItemID: parentID}, ListFields{})
	if err != nil {
		return false, err
	}

	itemRow, err := s.Item(ctx, query.ItemsListOptions{ItemID: itemID}, ListFields{})
	if err != nil {
		return false, err
	}

	newParentRow, err := s.Item(ctx, query.ItemsListOptions{ItemID: newParentID}, ListFields{})
	if err != nil {
		return false, err
	}

	if oldParentRow.ID == newParentRow.ID {
		return false, nil
	}

	if !oldParentRow.IsGroup {
		return false, errGroupRequired
	}

	if !newParentRow.IsGroup {
		return false, errGroupRequired
	}

	if !s.isAllowedCombination(itemRow.ItemTypeID, newParentRow.ItemTypeID) {
		return false, fmt.Errorf("%w: %d/%d", errInvalidItemParentCombination, itemRow.ItemTypeID, newParentRow.ItemTypeID)
	}

	parentIDs, err := s.collectAncestorsIDs(ctx, newParentRow.ID)
	if err != nil {
		return false, err
	}

	if util.Contains(parentIDs, itemID) {
		return false, errItemParentCycle
	}

	res, err := s.db.Update(schema.ItemParentTable).Set(goqu.Record{
		schema.ItemParentTableParentIDColName: newParentID,
	}).Where(
		schema.ItemParentTableItemIDCol.Eq(itemID),
		schema.ItemParentTableParentIDCol.Eq(parentID),
	).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	if affected <= 0 {
		return false, nil
	}

	_, err = s.db.Update(schema.ItemParentLanguageTable).Set(goqu.Record{
		schema.ItemParentLanguageTableParentIDColName: newParentID,
	}).Where(
		schema.ItemParentLanguageTableItemIDCol.Eq(itemID),
		schema.ItemParentLanguageTableParentIDCol.Eq(parentID),
	).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	_, err = s.RebuildCache(ctx, itemRow.ID)
	if err != nil {
		return false, err
	}

	_, err = s.refreshAuto(ctx, newParentID, itemID)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *Repository) getParentRows(ctx context.Context, itemID int64, stockFirst bool) ([]schema.ItemParentRow, error) {
	sqSelect := s.db.Select(schema.ItemParentTableParentIDCol).
		From(schema.ItemParentTable).
		Where(
			schema.ItemParentTableItemIDCol.Eq(itemID),
		)

	if stockFirst {
		sqSelect = sqSelect.Order(
			goqu.L("?", schema.ItemParentTableTypeCol.Eq(schema.ItemParentTypeDefault)).Desc(),
		)
	}

	var rows []schema.ItemParentRow

	err := sqSelect.ScanStructsContext(ctx, &rows)

	return rows, err
}

func (s *Repository) refreshAutoByVehicle(ctx context.Context, itemID int64) (bool, error) {
	rows, err := s.getParentRows(ctx, itemID, false)
	if err != nil {
		return false, err
	}

	somethingChanged := false

	for _, itemParentRow := range rows {
		changed, err := s.refreshAuto(ctx, itemParentRow.ParentID, itemID)
		if err != nil {
			return false, err
		}

		if changed {
			somethingChanged = true
		}
	}

	return somethingChanged, nil
}

func (s *Repository) refreshAuto(ctx context.Context, parentID, itemID int64) (bool, error) {
	var bvlRows []schema.ItemParentLanguageRow

	err := s.db.Select(
		schema.ItemParentLanguageTableIsAutoColName,
		schema.ItemParentLanguageTableNameCol,
		schema.ItemParentLanguageTableLanguageCol,
	).
		From(schema.ItemParentLanguageTable).Where(
		schema.ItemParentLanguageTableItemIDCol.Eq(itemID),
		schema.ItemParentLanguageTableParentIDCol.Eq(parentID),
	).ScanStructsContext(ctx, &bvlRows)
	if err != nil {
		return false, err
	}

	values := make(map[string]schema.ItemParentLanguageRow)

	for _, bvlRow := range bvlRows {
		row := schema.ItemParentLanguageRow{}
		if !bvlRow.IsAuto {
			row.Name = bvlRow.Name
		}

		values[bvlRow.Language] = row
	}

	err = s.setItemParentLanguages(ctx, parentID, itemID, values, false)
	if err != nil {
		return false, err
	}

	var bvRow schema.ItemParentRow

	success, err := s.db.Select(schema.ItemParentTableManualCatnameCol).
		From(schema.ItemParentTable).
		Where(
			schema.ItemParentTableItemIDCol.Eq(itemID),
			schema.ItemParentTableParentIDCol.Eq(parentID),
		).
		ScanStructContext(ctx, &bvRow)
	if err != nil {
		return false, err
	}

	if !success {
		return false, nil
	}

	if bvRow.ManualCatname {
		return true, nil
	}

	brandRow, err := s.Item(ctx, query.ItemsListOptions{ItemID: parentID}, ListFields{NameText: true})
	if err != nil {
		return false, err
	}

	vehicleRow, err := s.Item(ctx, query.ItemsListOptions{ItemID: itemID}, ListFields{NameText: true})
	if err != nil {
		return false, err
	}

	catname, err := s.extractCatname(ctx, brandRow.ItemRow, vehicleRow.ItemRow)
	if err != nil {
		return false, err
	}

	if len(catname) == 0 {
		return false, nil
	}

	_, err = s.db.Update(schema.ItemParentTable).Set(goqu.Record{
		schema.ItemParentTableCatnameColName: catname,
	}).Where(
		schema.ItemParentTableItemIDCol.Eq(itemID),
		schema.ItemParentTableParentIDCol.Eq(parentID),
	).Executor().ExecContext(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *Repository) ItemParent(ctx context.Context, itemID, parentID int64) (*schema.ItemParentRow, error) {
	listOptions := query.ItemParentListOptions{
		ItemID:   itemID,
		ParentID: parentID,
	}

	var res schema.ItemParentRow

	success, err := listOptions.Apply(schema.ItemParentTableName, s.db.Select().From(schema.ItemParentTable)).
		ScanStructContext(ctx, &res)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, ErrItemNotFound
	}

	return &res, nil
}

func (s *Repository) UserItemSubscribe(ctx context.Context, itemID, userID int64) error {
	_, err := s.db.Insert(schema.UserItemSubscribeTable).Rows(goqu.Record{
		schema.UserItemSubscribeTableUserIDColName: userID,
		schema.UserItemSubscribeTableItemIDColName: itemID,
	}).OnConflict(goqu.DoNothing()).Executor().ExecContext(ctx)

	return err
}

func (s *Repository) UserItemUnsubscribe(ctx context.Context, itemID, userID int64) error {
	_, err := s.db.Delete(schema.UserItemSubscribeTable).Where(
		schema.UserItemSubscribeTableUserIDCol.Eq(userID),
		schema.UserItemSubscribeTableItemIDCol.Eq(itemID),
	).Executor().ExecContext(ctx)

	return err
}
