package items

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

var ErrItemNotFound = errors.New("item not found")

const (
	TopBrandsCount      = 150
	NewDays             = 7
	TopPersonsCount     = 5
	TopFactoriesCount   = 8
	TopCategoriesCount  = 15
	TopTwinsBrandsCount = 20
)

const (
	colNameOnly                   = "name_only"
	colSpecName                   = "spec_name"
	colSpecShortName              = "spec_short_name"
	colDescription                = "description"
	colFullText                   = "full_text"
	colItemsCount                 = "items_count"
	colNewItemsCount              = "new_items_count"
	colDescendantsCount           = "descendants_count"
	colNewDescendantsCount        = "new_descendants_count"
	colChildItemsCount            = "child_items_count"
	colNewChildItemsCount         = "new_child_items_count"
	colCurrentPicturesCount       = "current_pictures_count"
	colChildsCount                = "childs_count"
	colDescendantTwinsGroupsCount = "descendant_twins_groups_count"
	colInboxPicturesCount         = "inbox_pictures_count"
)

type TreeItem struct {
	ID       int64
	Name     string
	Childs   []TreeItem
	ItemType ItemType
}

var languagePriority = map[string][]string{
	"xx":    {"en", "it", "fr", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", "he", "xx"},
	"en":    {"en", "it", "fr", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", "he", "xx"},
	"fr":    {"fr", "en", "it", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", "he", "xx"},
	"pt-br": {"pt", "en", "it", "fr", "de", "es", "ru", "be", "uk", "zh", "jp", "he", "xx"},
	"ru":    {"ru", "en", "it", "fr", "de", "es", "pt", "be", "uk", "zh", "jp", "he", "xx"},
	"be":    {"be", "ru", "uk", "en", "it", "fr", "de", "es", "pt", "zh", "jp", "he", "xx"},
	"uk":    {"uk", "ru", "en", "it", "fr", "de", "es", "pt", "be", "zh", "jp", "he", "xx"},
	"zh":    {"zh", "en", "it", "fr", "de", "es", "pt", "ru", "be", "uk", "jp", "he", "xx"},
	"es":    {"es", "en", "it", "fr", "de", "pt", "ru", "be", "uk", "zh", "jp", "he", "xx"},
	"it":    {"it", "en", "fr", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", "he", "xx"},
	"he":    {"he", "en", "it", "fr", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", "xx"},
}

type ItemType int

const (
	VEHICLE   ItemType = 1
	ENGINE    ItemType = 2
	CATEGORY  ItemType = 3
	TWINS     ItemType = 4
	BRAND     ItemType = 5
	FACTORY   ItemType = 6
	MUSEUM    ItemType = 7
	PERSON    ItemType = 8
	COPYRIGHT ItemType = 9
)

const (
	ItemParentTypeDefault = 0
	ItemParentTypeTuning  = 1
	ItemParentTypeSport   = 2
	ItemParentTypeDesign  = 3
)

// Repository Main Object.
type Repository struct {
	db                *goqu.Database
	mostsMinCarsCount int
}

type Item struct {
	ID                         int64
	Catname                    string
	NameOnly                   string
	Body                       string
	ItemsCount                 int32
	NewItemsCount              int32
	ChildItemsCount            int32
	NewChildItemsCount         int32
	DescendantsCount           int32
	NewDescendantsCount        int32
	BeginYear                  int32
	EndYear                    int32
	BeginMonth                 int16
	EndMonth                   int16
	Today                      *bool
	BeginModelYear             int32
	EndModelYear               int32
	BeginModelYearFraction     string
	EndModelYearFraction       string
	SpecID                     int64
	SpecName                   string
	SpecShortName              string
	EngineItemID               int64
	ItemTypeID                 ItemType
	Description                string
	FullText                   string
	IsConcept                  bool
	IsConceptInherit           bool
	CurrentPicturesCount       int32
	ChildsCount                int32
	DescendantTwinsGroupsCount int32
	InboxPicturesCount         int32
	FullName                   string
	LogoID                     int64
	MostsActive                bool
}

type ItemLanguage struct {
	ItemID     int64
	Language   string
	Name       string
	TextID     int64
	FullTextID int64
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
) *Repository {
	return &Repository{
		db:                db,
		mostsMinCarsCount: mostsMinCarsCount,
	}
}

type PicturesOptions struct {
	Status      pictures.Status
	OwnerID     int64
	ItemPicture *ItemPicturesOptions
}

type ItemPicturesOptions struct {
	TypeID        pictures.ItemPictureType
	Pictures      *PicturesOptions
	PerspectiveID int32
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
	ItemsCount                 bool
	NewItemsCount              bool
	ChildItemsCount            bool
	NewChildItemsCount         bool
	DescendantsCount           bool
	NewDescendantsCount        bool
	NameText                   bool
	CurrentPicturesCount       bool
	ChildsCount                bool
	DescendantTwinsGroupsCount bool
	InboxPicturesCount         bool
	FullName                   bool
	Logo                       bool
	MostsActive                bool
	CommentsAttentionsCount    bool
}

type ListOptions struct {
	Alias              string
	Language           string
	Fields             ListFields
	ItemID             int64
	ItemIDExpr         goqu.Expression
	TypeID             []ItemType
	DescendantPictures *ItemPicturesOptions
	PreviewPictures    *ItemPicturesOptions
	Limit              uint32
	Page               uint32
	OrderBy            []exp.OrderedExpression
	SortByName         bool
	ChildItems         *ListOptions
	DescendantItems    *ListOptions
	ParentItems        *ListOptions
	AncestorItems      *ListOptions
	NoParents          bool
	Catname            string
	Name               string
	IsConcept          bool
	EngineItemID       int64
	HasBeginYear       bool
	HasEndYear         bool
	HasBeginMonth      bool
	HasEndMonth        bool
	HasLogo            bool
}

func applyPicture(alias string, sqSelect *goqu.SelectDataset, options *PicturesOptions) *goqu.SelectDataset {
	pAlias := alias + "_p"

	if options.Status != "" || options.ItemPicture != nil || options.OwnerID != 0 {
		sqSelect = sqSelect.Join(
			schema.PictureTable.As(pAlias),
			goqu.On(goqu.T(alias).Col("picture_id").Eq(goqu.T(pAlias).Col("id"))),
		)

		if options.Status != "" {
			sqSelect = sqSelect.Where(goqu.T(pAlias).Col("status").Eq(options.Status))
		}

		if options.OwnerID != 0 {
			sqSelect = sqSelect.Where(goqu.T(pAlias).Col("owner_id").Eq(options.OwnerID))
		}

		if options.ItemPicture != nil {
			sqSelect, _ = applyItemPicture(pAlias, "id", sqSelect, options.ItemPicture)
		}
	}

	return sqSelect
}

func applyItemPicture(
	alias string, itemIDColumn string, sqSelect *goqu.SelectDataset, options *ItemPicturesOptions,
) (*goqu.SelectDataset, string) {
	piAlias := alias + "_pi"

	sqSelect = sqSelect.Join(
		schema.PictureItemTable.As(piAlias),
		goqu.On(goqu.T(alias).Col(itemIDColumn).Eq(goqu.T(piAlias).Col("item_id"))),
	)

	if options != nil {
		if options.TypeID != 0 {
			sqSelect = sqSelect.Where(goqu.T(piAlias).Col("type").Eq(options.TypeID))
		}

		if options.PerspectiveID != 0 {
			sqSelect = sqSelect.Where(goqu.T(piAlias).Col("perspective_id").Eq(options.PerspectiveID))
		}

		if options.Pictures != nil {
			sqSelect = applyPicture(piAlias, sqSelect, options.Pictures)
		}
	}

	return sqSelect, piAlias
}

func (s *Repository) applyItem( //nolint:maintidx
	alias string,
	sqSelect *goqu.SelectDataset,
	fields bool,
	options *ListOptions,
) (*goqu.SelectDataset, error) {
	var err error

	aliasTable := goqu.T(alias)
	aliasIDCol := aliasTable.Col(schema.ItemTableIDColName)

	if options.ItemID > 0 {
		sqSelect = sqSelect.Where(aliasIDCol.Eq(options.ItemID))
	}

	if options.ItemIDExpr != nil {
		sqSelect = sqSelect.Where(aliasIDCol.Eq(options.ItemIDExpr))
	}

	if options.TypeID != nil && len(options.TypeID) > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableItemTypeIDColName).Eq(options.TypeID))
	}

	ipcAlias := alias + "_ipc"

	if options.ChildItems != nil {
		iAlias := alias + "_ic"
		sqSelect = sqSelect.
			Join(
				schema.ItemParentTable.As(ipcAlias),
				goqu.On(aliasIDCol.Eq(goqu.T(ipcAlias).Col("parent_id"))),
			).
			Join(
				schema.ItemTable.As(iAlias),
				goqu.On(goqu.T(ipcAlias).Col("item_id").Eq(goqu.T(iAlias).Col("id"))),
			)
		sqSelect, err = s.applyItem(iAlias, sqSelect, fields, options.ChildItems)

		if err != nil {
			return sqSelect, err
		}
	}

	if options.ParentItems != nil {
		iAlias := alias + "_ip"
		ippAlias := alias + "_ipp"
		sqSelect = sqSelect.
			Join(
				schema.ItemParentTable.As(ippAlias),
				goqu.On(aliasIDCol.Eq(goqu.T(ippAlias).Col("item_id"))),
			).
			Join(
				schema.ItemTable.As(iAlias),
				goqu.On(goqu.T(ippAlias).Col("parent_id").Eq(goqu.T(iAlias).Col("id"))),
			)
		sqSelect, err = s.applyItem(iAlias, sqSelect, fields, options.ParentItems)

		if err != nil {
			return sqSelect, err
		}
	}

	columns := make([]interface{}, 0)

	if options.DescendantItems != nil || options.DescendantPictures != nil || options.Fields.CurrentPicturesCount {
		ipcdAlias := alias + "_ipcd"
		iAlias := alias + "_id"
		sqSelect = sqSelect.Join(
			schema.ItemParentCacheTable.As(ipcdAlias),
			goqu.On(aliasIDCol.Eq(goqu.T(ipcdAlias).Col("parent_id"))),
		)

		if options.DescendantItems != nil {
			sqSelect = sqSelect.
				Join(
					schema.ItemTable.As(iAlias),
					goqu.On(goqu.T(ipcdAlias).Col("item_id").Eq(goqu.T(iAlias).Col("id"))),
				)
			sqSelect, err = s.applyItem(iAlias, sqSelect, fields, options.DescendantItems)

			if err != nil {
				return sqSelect, err
			}
		}

		if options.DescendantPictures != nil || options.Fields.CurrentPicturesCount {
			var piAlias string
			sqSelect, piAlias = applyItemPicture(ipcdAlias, "item_id", sqSelect, options.DescendantPictures)

			if options.Fields.CurrentPicturesCount {
				columns = append(columns, goqu.COUNT(goqu.DISTINCT(goqu.T(piAlias).Col("picture_id"))).As(colCurrentPicturesCount))
			}
		}
	}

	if options.AncestorItems != nil {
		ipcaAlias := alias + "_ipca"
		iAlias := alias + "_ia"
		sqSelect = sqSelect.
			Join(
				schema.ItemParentCacheTable.As(ipcaAlias),
				goqu.On(aliasIDCol.Eq(goqu.T(ipcaAlias).Col("item_id"))),
			).
			Join(
				schema.ItemTable.As(iAlias),
				goqu.On(goqu.T(ipcaAlias).Col("parent_id").Eq(goqu.T(iAlias).Col("id"))),
			)
		sqSelect, err = s.applyItem(iAlias, sqSelect, fields, options.AncestorItems)

		if err != nil {
			return sqSelect, err
		}
	}

	if options.NoParents {
		ipnpAlias := alias + "_ipnp"
		sqSelect = sqSelect.
			LeftJoin(
				schema.ItemParentTable.As(ipnpAlias),
				goqu.On(aliasIDCol.Eq(goqu.T(ipnpAlias).Col("item_id"))),
			).
			Where(goqu.T(ipnpAlias).Col("parent_id").IsNull())
	}

	if len(options.Catname) > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableCatnameColName).Eq(options.Catname))
	}

	if options.IsConcept {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableIsConceptColName))
	}

	if options.EngineItemID > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableEngineItemIDColName).Eq(options.EngineItemID))
	}

	if len(options.Name) > 0 {
		subSelect := sqSelect.ClearSelect().ClearLimit().ClearOffset().ClearOrder().ClearWhere().GroupBy()

		// WHERE EXISTS(SELECT item_id FROM item_language WHERE item.id = item_id AND name ILIKE ?)
		sqSelect = sqSelect.Where(
			goqu.L(
				"EXISTS ?",
				subSelect.
					From(schema.ItemLanguageTable).
					Where(
						aliasIDCol.Eq(schema.ItemLanguageTableItemIDCol),
						schema.ItemLanguageTableNameCol.ILike(options.Name),
					),
			),
		)
	}

	if options.HasBeginYear {
		sqSelect = sqSelect.Where(aliasTable.Col("begin_year"))
	}

	if options.HasEndYear {
		sqSelect = sqSelect.Where(aliasTable.Col("end_year"))
	}

	if options.HasBeginMonth {
		sqSelect = sqSelect.Where(aliasTable.Col("begin_month"))
	}

	if options.HasEndMonth {
		sqSelect = sqSelect.Where(aliasTable.Col("end_month"))
	}

	if options.HasLogo {
		sqSelect = sqSelect.Where(aliasTable.Col("logo_id").IsNotNull())
	}

	if fields {
		if options.Fields.FullName {
			columns = append(columns, aliasTable.Col(schema.ItemTableFullNameColName))
		}

		if options.Fields.Logo {
			columns = append(columns, aliasTable.Col(schema.ItemTableLogoIDColName))
		}

		if options.Fields.NameText || options.Fields.NameHTML {
			isAlias := alias + "_is"

			columns = append(columns,
				aliasTable.Col(schema.ItemTableBeginYearColName), aliasTable.Col(schema.ItemTableEndYearColName),
				aliasTable.Col(schema.ItemTableBeginMonthColName), aliasTable.Col(schema.ItemTableEndMonthColName),
				aliasTable.Col(schema.ItemTableBeginModelYearColName), aliasTable.Col(schema.ItemTableEndModelYearColName),
				aliasTable.Col(schema.ItemTableBeginModelYearFractionColName),
				aliasTable.Col(schema.ItemTableEndModelYearFractionColName),
				aliasTable.Col(schema.ItemTableTodayColName),
				aliasTable.Col(schema.ItemTableBodyColName),
				goqu.T(isAlias).Col("short_name").As(colSpecShortName),
			)

			if options.Fields.NameHTML {
				columns = append(columns, goqu.T(isAlias).Col("name").As(colSpecName))
			}

			sqSelect = sqSelect.
				LeftJoin(
					schema.SpecTable.As(isAlias),
					goqu.On(aliasTable.Col(schema.ItemTableSpecIDColName).Eq(goqu.T(isAlias).Col("id"))),
				)
		}

		if options.Fields.Description {
			ilAlias := alias + "_ild"

			columns = append(columns,
				s.db.Select(schema.TextstorageTextTableTextCol).
					From(schema.ItemLanguageTable.As(ilAlias)).
					Join(
						schema.TextstorageTextTable,
						goqu.On(goqu.T(ilAlias).Col("text_id").Eq(schema.TextstorageTextTableIDCol)),
					).
					Where(
						goqu.T(ilAlias).Col("item_id").Eq(aliasIDCol),
						goqu.Func("length", schema.TextstorageTextTableTextCol).Gt(0),
					).
					Order(goqu.L(ilAlias+".language = ?", options.Language).Desc()).
					Limit(1).
					As(colDescription),
			)
		}

		if options.Fields.FullText {
			ilAlias := alias + "_ilf"
			columns = append(columns,
				s.db.Select(schema.TextstorageTextTableTextCol).
					From(schema.ItemLanguageTable.As(ilAlias)).
					Join(
						schema.TextstorageTextTable,
						goqu.On(goqu.T(ilAlias).Col("full_text_id").Eq(schema.TextstorageTextTableIDCol)),
					).
					Where(
						goqu.T(ilAlias).Col("item_id").Eq(aliasIDCol),
						goqu.Func("length", schema.TextstorageTextTableTextCol).Gt(0),
					).
					Order(goqu.L(ilAlias+".language = ?", options.Language).Desc()).
					Limit(1).
					As(colFullText),
			)
		}

		if options.SortByName || options.Fields.NameOnly || options.Fields.NameText || options.Fields.NameHTML {
			langPriority, ok := languagePriority[options.Language]
			if !ok {
				return sqSelect, fmt.Errorf("language `%s` not found", options.Language)
			}

			s := make([]interface{}, len(langPriority))
			for i, v := range langPriority {
				s[i] = v
			}

			columns = append(columns, goqu.L(`
				IFNULL(
					(SELECT name
					FROM `+schema.ItemLanguageTableName+`
					WHERE item_id = `+alias+`.id AND length(name) > 0
					ORDER BY FIELD(language, `+strings.Repeat(",?", len(s))[1:]+`)
					LIMIT 1),
					`+alias+`.name
				)
			`, s...).As(colNameOnly))
		}

		if options.Fields.ChildItemsCount {
			columns = append(columns, goqu.L("count(distinct "+ipcAlias+".item_id)").As(colChildItemsCount))
		}

		if options.Fields.NewChildItemsCount {
			columns = append(
				columns,
				goqu.L("count(distinct IF("+ipcAlias+".timestamp > DATE_SUB(NOW(), INTERVAL ? DAY), "+
					ipcAlias+".item_id, NULL))", NewDays).
					As(colNewChildItemsCount))
		}

		if options.Fields.ItemsCount {
			columns = append(columns, goqu.L("count(distinct "+alias+".id)").As(colItemsCount))
		}

		if options.Fields.NewItemsCount {
			columns = append(columns, goqu.L(`
				count(distinct if(`+alias+`.add_datetime > date_sub(NOW(), INTERVAL ? DAY), `+alias+`.id, null))
			`, NewDays).As(colNewItemsCount))
		}

		if options.Fields.ChildsCount {
			subSelect := sqSelect.ClearSelect().ClearLimit().ClearOffset().ClearOrder().ClearWhere().GroupBy()

			columns = append(
				columns,
				subSelect.Select(goqu.COUNT(goqu.Star())).
					From(schema.ItemParentTable).
					Where(schema.ItemParentTableParentIDCol.Eq(goqu.T(alias).Col(schema.ItemTableIDColName))).
					As(colChildsCount),
			)
		}

		if options.Fields.DescendantsCount {
			columns = append(columns, goqu.L(`
				(
					SELECT count(distinct product1.id)
					FROM `+schema.ItemTableName+` AS product1
						JOIN `+schema.ItemParentCacheTableName+` ON product1.id = `+schema.ItemParentCacheTableName+`.item_id
					WHERE `+schema.ItemParentCacheTableName+`.parent_id = `+alias+`.id
						AND `+schema.ItemParentCacheTableName+`.item_id <> `+schema.ItemParentCacheTableName+`.parent_id
					LIMIT 1
				) 
			`).As(colDescendantsCount))
		}

		if options.Fields.NewDescendantsCount {
			columns = append(columns, goqu.L(`
				(
					SELECT count(distinct product2.id)
					FROM `+schema.ItemTableName+` AS product2
						JOIN `+schema.ItemParentCacheTableName+` ON product2.id = `+schema.ItemParentCacheTableName+`.item_id
					WHERE `+schema.ItemParentCacheTableName+`.parent_id = `+alias+`.id
						AND `+schema.ItemParentCacheTableName+`.item_id <> `+schema.ItemParentCacheTableName+`.parent_id
						AND product2.add_datetime > DATE_SUB(NOW(), INTERVAL ? DAY)
				) 
			`, NewDays).As(colNewDescendantsCount))
		}

		if options.Fields.DescendantTwinsGroupsCount {
			subSelect, err := s.CountSelect(ListOptions{
				Alias:  alias + "dtgc",
				TypeID: []ItemType{TWINS},
				DescendantItems: &ListOptions{
					AncestorItems: &ListOptions{
						ItemIDExpr: goqu.T(alias).Col(schema.ItemTableIDColName),
					},
				},
			})
			if err != nil {
				return nil, err
			}

			columns = append(
				columns,
				subSelect.As(colDescendantTwinsGroupsCount),
			)
		}

		if options.Fields.InboxPicturesCount {
			subSelect, err := s.CountSelect(ListOptions{
				Alias:  alias + "ipc",
				TypeID: []ItemType{TWINS},
				AncestorItems: &ListOptions{
					ItemIDExpr: goqu.T(alias).Col(schema.ItemTableIDColName),
				},
			})
			if err != nil {
				return nil, err
			}

			columns = append(
				columns,
				subSelect.As(colInboxPicturesCount),
			)
		}

		sqSelect = sqSelect.SelectAppend(columns...)
	}

	return sqSelect, nil
}

func (s *Repository) CountSelect(options ListOptions) (*goqu.SelectDataset, error) {
	var err error

	alias := "i"
	if options.Alias != "" {
		alias = options.Alias
	}

	sqSelect := s.db.Select(goqu.COUNT(goqu.Star())).From(schema.ItemTable.As(alias))

	sqSelect, err = s.applyItem(alias, sqSelect, false, &options)
	if err != nil {
		return nil, err
	}

	return sqSelect, nil
}

func (s *Repository) Count(ctx context.Context, options ListOptions) (int, error) {
	var err error

	sqSelect, err := s.CountSelect(options)
	if err != nil {
		return 0, err
	}

	var count int

	_, err = sqSelect.Executor().ScanValContext(ctx, &count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Repository) CountDistinct(ctx context.Context, options ListOptions) (int, error) {
	var err error

	sqSelect := s.db.Select(goqu.L("COUNT(DISTINCT i.id)")).From(schema.ItemTable.As("i"))

	sqSelect, err = s.applyItem("i", sqSelect, false, &options)
	if err != nil {
		return 0, err
	}

	var count int

	_, err = sqSelect.Executor().ScanValContext(ctx, &count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Repository) Item(ctx context.Context, id int64, language string, fields ListFields) (Item, error) {
	options := ListOptions{
		ItemID:   id,
		Fields:   fields,
		Limit:    1,
		Language: language,
	}

	res, _, err := s.List(ctx, options)
	if err != nil {
		return Item{}, err
	}

	if len(res) == 0 {
		return Item{}, ErrItemNotFound
	}

	return res[0], nil
}

func (s *Repository) List(ctx context.Context, options ListOptions) ([]Item, *util.Pages, error) { //nolint:maintidx
	/*langPriority, ok := languagePriority[options.Language]
	if !ok {
		return nil, fmt.Errorf("language `%s` not found", options.Language)
	}*/
	var err error

	alias := "i"

	sqSelect := s.db.Select(
		goqu.T(alias).Col(schema.ItemTableIDColName),
		goqu.T(alias).Col(schema.ItemTableCatnameColName),
		goqu.T(alias).Col(schema.ItemTableEngineItemIDColName),
		goqu.T(alias).Col(schema.ItemTableItemTypeIDColName),
		goqu.T(alias).Col(schema.ItemTableIsConceptColName),
		goqu.T(alias).Col(schema.ItemTableIsConceptInheritColName),
		goqu.T(alias).Col(schema.ItemTableSpecIDColName),
		goqu.T(alias).Col(schema.ItemTableFullNameColName),
	).From(schema.ItemTable.As(alias)).
		GroupBy(goqu.T(alias).Col(schema.ItemTableIDColName))

	sqSelect, err = s.applyItem("i", sqSelect, true, &options)
	if err != nil {
		return nil, nil, err
	}

	if len(options.OrderBy) > 0 {
		sqSelect = sqSelect.Order(options.OrderBy...)
	}

	var pages *util.Pages

	if options.Limit > 0 {
		paginator := util.Paginator{
			SQLSelect:         sqSelect,
			ItemCountPerPage:  int32(options.Limit),
			CurrentPageNumber: int32(options.Page),
		}

		pages, err = paginator.GetPages(ctx)
		if err != nil {
			return nil, nil, err
		}

		sqSelect, err = paginator.GetCurrentItems(ctx)
		if err != nil {
			return nil, nil, err
		}
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
		var r Item

		var (
			catname                sql.NullString
			engineItemID           sql.NullInt64
			beginYear              sql.NullInt32
			endYear                sql.NullInt32
			beginMonth             sql.NullInt16
			endMonth               sql.NullInt16
			beginModelYear         sql.NullInt32
			endModelYear           sql.NullInt32
			beginModelYearFraction sql.NullString
			endModelYearFraction   sql.NullString
			today                  sql.NullBool
			specID                 sql.NullInt64
			specName               sql.NullString
			specShortName          sql.NullString
			description            sql.NullString
			fullText               sql.NullString
			fullName               sql.NullString
			logoID                 sql.NullInt64
		)

		pointers := make([]interface{}, len(columnNames))

		for i, colName := range columnNames {
			switch colName {
			case schema.ItemTableIDColName:
				pointers[i] = &r.ID
			case colNameOnly:
				pointers[i] = &r.NameOnly
			case schema.ItemTableCatnameColName:
				pointers[i] = &catname
			case schema.ItemTableFullNameColName:
				pointers[i] = &fullName
			case schema.ItemTableEngineItemIDColName:
				pointers[i] = &engineItemID
			case schema.ItemTableItemTypeIDColName:
				pointers[i] = &r.ItemTypeID
			case schema.ItemTableIsConceptColName:
				pointers[i] = &r.IsConcept
			case schema.ItemTableIsConceptInheritColName:
				pointers[i] = &r.IsConceptInherit
			case schema.ItemTableSpecIDColName:
				pointers[i] = &specID
			case colDescription:
				pointers[i] = &description
			case colFullText:
				pointers[i] = &fullText
			case colItemsCount:
				pointers[i] = &r.ItemsCount
			case colNewItemsCount:
				pointers[i] = &r.NewItemsCount
			case colDescendantsCount:
				pointers[i] = &r.DescendantsCount
			case colNewDescendantsCount:
				pointers[i] = &r.NewDescendantsCount
			case colChildItemsCount:
				pointers[i] = &r.ChildItemsCount
			case colNewChildItemsCount:
				pointers[i] = &r.NewChildItemsCount
			case schema.ItemTableBeginYearColName:
				pointers[i] = &beginYear
			case schema.ItemTableEndYearColName:
				pointers[i] = &endYear
			case schema.ItemTableBeginMonthColName:
				pointers[i] = &beginMonth
			case schema.ItemTableEndMonthColName:
				pointers[i] = &endMonth
			case schema.ItemTableBeginModelYearColName:
				pointers[i] = &beginModelYear
			case schema.ItemTableEndModelYearColName:
				pointers[i] = &endModelYear
			case schema.ItemTableBeginModelYearFractionColName:
				pointers[i] = &beginModelYearFraction
			case schema.ItemTableEndModelYearFractionColName:
				pointers[i] = &endModelYearFraction
			case schema.ItemTableTodayColName:
				pointers[i] = &today
			case schema.ItemTableBodyColName:
				pointers[i] = &r.Body
			case colSpecName:
				pointers[i] = &specName
			case colSpecShortName:
				pointers[i] = &specShortName
			case colCurrentPicturesCount:
				pointers[i] = &r.CurrentPicturesCount
			case colChildsCount:
				pointers[i] = &r.ChildsCount
			case colDescendantTwinsGroupsCount:
				pointers[i] = &r.DescendantTwinsGroupsCount
			case colInboxPicturesCount:
				pointers[i] = &r.InboxPicturesCount
			case schema.ItemTableLogoIDColName:
				pointers[i] = &logoID
			default:
				pointers[i] = nil
			}
		}

		err = rows.Scan(pointers...)
		if err != nil {
			return nil, nil, err
		}

		if catname.Valid {
			r.Catname = catname.String
		}

		if engineItemID.Valid {
			r.EngineItemID = engineItemID.Int64
		}

		if beginYear.Valid {
			r.BeginYear = beginYear.Int32
		}

		if endYear.Valid {
			r.EndYear = endYear.Int32
		}

		if beginMonth.Valid {
			r.BeginMonth = beginMonth.Int16
		}

		if endMonth.Valid {
			r.EndMonth = endMonth.Int16
		}

		if beginModelYear.Valid {
			r.BeginModelYear = beginModelYear.Int32
		}

		if endModelYear.Valid {
			r.EndModelYear = endModelYear.Int32
		}

		if beginModelYearFraction.Valid {
			r.BeginModelYearFraction = beginModelYearFraction.String
		}

		if endModelYearFraction.Valid {
			r.EndModelYearFraction = endModelYearFraction.String
		}

		if today.Valid {
			r.Today = util.BoolPtr(today.Bool)
		}

		if specID.Valid {
			r.SpecID = specID.Int64
		}

		if specName.Valid {
			r.SpecName = specName.String
		}

		if specShortName.Valid {
			r.SpecShortName = specShortName.String
		}

		if description.Valid {
			r.Description = description.String
		}

		if fullText.Valid {
			r.FullText = fullText.String
		}

		if fullName.Valid {
			r.FullName = fullName.String
		}

		if logoID.Valid {
			r.LogoID = logoID.Int64
		}

		if options.Fields.MostsActive {
			carsCount, err := s.Count(ctx, ListOptions{
				AncestorItems: &ListOptions{
					ItemID: r.ID,
				},
			})
			if err != nil {
				return nil, nil, err
			}

			r.MostsActive = carsCount >= s.mostsMinCarsCount
		}

		result = append(result, r)
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
			a := result[i].NameOnly
			b := result[j].NameOnly

			switch options.Language {
			case "ru", "uk", "be":
				aIsCyrillic := cyrillic.MatchString(a)
				bIsCyrillic := cyrillic.MatchString(b)

				if aIsCyrillic && !bIsCyrillic {
					return true
				}

				if bIsCyrillic && !aIsCyrillic {
					return false
				}
			case "zh":
				aIsHan := han.MatchString(a)
				bIsHan := han.MatchString(b)

				if aIsHan && !bIsHan {
					return true
				}

				if bIsHan && !aIsHan {
					return false
				}
			}

			return cl.CompareString(a, b) == -1
		})
	}

	return result, pages, nil
}

func (s *Repository) Tree(ctx context.Context, id string) (*TreeItem, error) {
	type row struct {
		ID       int64    `db:"id"`
		Name     string   `db:"name"`
		ItemType ItemType `db:"item_type_id"`
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
		err = s.refreshItemVehicleTypeInheritanceFromParents(ctx, itemID)
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
		err = s.refreshItemVehicleTypeInheritanceFromParents(ctx, itemID)
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
	res, err := s.db.ExecContext(
		ctx,
		`
			INSERT INTO `+schema.VehicleVehicleTypeTableName+` (vehicle_id, vehicle_type_id, inherited)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE inherited = VALUES(inherited)
        `,
		itemID, vehicleTypeID, inherited,
	)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (s *Repository) refreshItemVehicleTypeInheritanceFromParents(ctx context.Context, itemID int64) error {
	typeIds, err := s.getItemVehicleTypeIDs(ctx, itemID, false)
	if err != nil {
		return err
	}

	if len(typeIds) > 0 {
		// do not inherit when own value
		res, err := s.db.ExecContext(
			ctx,
			"DELETE FROM "+schema.VehicleVehicleTypeTableName+" WHERE vehicle_id = ? AND inherited",
			itemID,
		)
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
	rows, err := s.db.QueryContext(
		ctx,
		"SELECT item_id FROM "+schema.ItemParentTableName+" WHERE parent_id = ?",
		itemID,
	)
	if err != nil {
		return err
	}

	defer util.Close(rows)

	for rows.Next() {
		var childID int64

		err = rows.Scan(&childID)
		if err != nil {
			return err
		}

		err = s.refreshItemVehicleTypeInheritanceFromParents(ctx, childID)
		if err != nil {
			return err
		}
	}

	return rows.Err()
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
		var parentID, typeID int64

		err = rows.Scan(&parentID, &typeID)
		if err != nil {
			return nil, err
		}

		isTuning := typeID == ItemParentTypeTuning
		isSport := typeID == ItemParentTypeSport
		isDesign := typeID == ItemParentTypeDesign
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

	var updates int64

	//nolint: sqlclosecheck
	stmt, err := s.db.PrepareContext(ctx, `
   		INSERT INTO `+schema.ItemParentCacheTableName+` (item_id, parent_id, diff, tuning, sport, design)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			diff = VALUES(diff),
			tuning = VALUES(tuning),
			sport = VALUES(sport),
			design = VALUES(design)
	`)
	if err != nil {
		return 0, err
	}
	defer util.Close(stmt)

	for parentID, info := range parentInfos {
		result, err := stmt.ExecContext(ctx, itemID, parentID, info.Diff, info.Tuning, info.Sport, info.Design)
		if err != nil {
			return 0, err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}

		updates += affected
	}

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
		affected, err := s.RebuildCache(ctx, child)
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
		schema.ItemLanguageTableLanguageCol.Neq("xx"),
	)

	rows, err := sqSelect.Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	var result []ItemLanguage

	for rows.Next() {
		var (
			r              ItemLanguage
			nullName       sql.NullString
			nullTextID     sql.NullInt64
			nullFullTextID sql.NullInt64
		)

		err = rows.Scan(&r.ItemID, &r.Language, &nullName, &nullTextID, &nullFullTextID)
		if err != nil {
			return nil, err
		}

		r.Name = ""
		if nullName.Valid {
			r.Name = nullName.String
		}

		r.TextID = 0
		if nullTextID.Valid {
			r.TextID = nullTextID.Int64
		}

		r.FullTextID = 0
		if nullFullTextID.Valid {
			r.FullTextID = nullFullTextID.Int64
		}

		result = append(result, r)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
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
		var r ItemParentLanguage

		err = rows.Scan(&r.ItemID, &r.ParentID, &r.Language, &r.Name)
		if err != nil {
			return nil, err
		}

		result = append(result, r)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Repository) ItemsWithPicturesCount(
	ctx context.Context, nPictures int,
) (int32, error) {
	var result int32

	const countAlias = "c"

	sqSelect := s.db.Select(goqu.COUNT(goqu.Star())).From(
		s.db.Select(schema.ItemTableIDCol, goqu.COUNT(schema.PictureTableIDCol).As(countAlias)).
			From(schema.ItemTable).
			Join(schema.PictureItemTable, goqu.On(schema.ItemTableIDCol.Eq(schema.PictureItemTable.Col("item_id")))).
			Join(schema.PictureTable, goqu.On(schema.PictureItemTable.Col("picture_id").Eq(schema.PictureTableIDCol))).
			GroupBy(schema.ItemTableIDCol).
			Having(goqu.C(countAlias).Gte(nPictures)).
			As("T1"),
	)

	success, err := sqSelect.ScanValContext(ctx, &result)
	if err != nil {
		return 0, err
	}

	if !success {
		return 0, sql.ErrNoRows
	}

	return result, nil
}
