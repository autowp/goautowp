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
	tableItem            = "item"
	tableItemParent      = "item_parent"
	tableItemParentCache = "item_parent_cache"
	tableItemLanguage    = "item_language"
	colCatname           = "catname"
	colEngineItemID      = "engine_item_id"
	colItemTypeID        = "item_type_id"
	colIsConcept         = "is_concept"
	colIsConceptInherit  = "is_concept_inherit"
	colSpecID            = "spec_id"
	colID                = "id"
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
	db *goqu.Database
}

type Item struct {
	ID                     int64
	Catname                string
	NameOnly               string
	Body                   string
	ItemsCount             int32
	NewItemsCount          int32
	ChildItemsCount        int32
	NewChildItemsCount     int32
	DescendantsCount       int32
	NewDescendantsCount    int32
	BeginYear              int32
	EndYear                int32
	BeginMonth             int16
	EndMonth               int16
	Today                  *bool
	BeginModelYear         int32
	EndModelYear           int32
	BeginModelYearFraction string
	EndModelYearFraction   string
	SpecID                 int64
	SpecName               string
	SpecShortName          string
	EngineItemID           int64
	ItemTypeID             ItemType
	Description            string
	FullText               string
	IsConcept              bool
	IsConceptInherit       bool
	CurrentPicturesCount   int32
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
) *Repository {
	return &Repository{
		db: db,
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
	NameOnly             bool
	NameHTML             bool
	NameDefault          bool
	Description          bool
	FullText             bool
	HasText              bool
	PreviewPictures      ListPreviewPicturesFields
	TotalPictures        bool
	ItemsCount           bool
	NewItemsCount        bool
	ChildItemsCount      bool
	NewChildItemsCount   bool
	DescendantsCount     bool
	NewDescendantsCount  bool
	NameText             bool
	CurrentPicturesCount bool
}

type ListOptions struct {
	Language           string
	Fields             ListFields
	ItemID             int64
	TypeID             []ItemType
	DescendantPictures *ItemPicturesOptions
	PreviewPictures    *ItemPicturesOptions
	Limit              uint32
	OrderBy            []exp.OrderedExpression
	SortByName         bool
	ChildItems         *ListOptions
	DescendantItems    *ListOptions
	ParentItems        *ListOptions
	AncestorItems      *ListOptions
	NoParents          bool
	Catname            string
}

func applyPicture(alias string, sqSelect *goqu.SelectDataset, options *PicturesOptions) *goqu.SelectDataset {
	pAlias := alias + "_p"

	if options.Status != "" || options.ItemPicture != nil || options.OwnerID != 0 {
		sqSelect = sqSelect.Join(
			goqu.I("pictures").As(pAlias),
			goqu.On(goqu.I(alias+".picture_id").Eq(goqu.I(pAlias+".id"))),
		)

		if options.Status != "" {
			sqSelect = sqSelect.Where(goqu.Ex{pAlias + ".status": options.Status})
		}

		if options.OwnerID != 0 {
			sqSelect = sqSelect.Where(goqu.Ex{pAlias + ".owner_id": options.OwnerID})
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
		goqu.I("picture_item").As(piAlias),
		goqu.On(goqu.T(alias).Col(itemIDColumn).Eq(goqu.I(piAlias+".item_id"))),
	)

	if options != nil {
		if options.TypeID != 0 {
			sqSelect = sqSelect.Where(goqu.Ex{piAlias + ".type": options.TypeID})
		}

		if options.PerspectiveID != 0 {
			sqSelect = sqSelect.Where(goqu.Ex{piAlias + ".perspective_id": options.PerspectiveID})
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

	if options.ItemID > 0 {
		sqSelect = sqSelect.Where(goqu.T(alias).Col("id").Eq(options.ItemID))
	}

	if options.TypeID != nil && len(options.TypeID) > 0 {
		sqSelect = sqSelect.Where(goqu.T(alias).Col("item_type_id").Eq(options.TypeID))
	}

	ipcAlias := alias + "_ipc"

	if options.ChildItems != nil {
		iAlias := alias + "_ic"
		sqSelect = sqSelect.
			Join(
				goqu.T(tableItemParent).As(ipcAlias),
				goqu.On(goqu.T(alias).Col(colID).Eq(goqu.T(ipcAlias).Col("parent_id"))),
			).
			Join(
				goqu.T(tableItem).As(iAlias),
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
				goqu.T(tableItemParent).As(ippAlias),
				goqu.On(goqu.T(alias).Col(colID).Eq(goqu.T(ippAlias).Col("item_id"))),
			).
			Join(
				goqu.T(tableItem).As(iAlias),
				goqu.On(goqu.I(ippAlias+".parent_id").Eq(goqu.I(iAlias+".id"))),
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
			goqu.T(tableItemParentCache).As(ipcdAlias),
			goqu.On(goqu.T(alias).Col(colID).Eq(goqu.T(ipcdAlias).Col("parent_id"))),
		)

		if options.DescendantItems != nil {
			sqSelect = sqSelect.
				Join(
					goqu.T(tableItem).As(iAlias),
					goqu.On(goqu.T(ipcdAlias).Col("item_id").Eq(goqu.T(iAlias).Col("id"))),
				)
			sqSelect, err = s.applyItem(iAlias, sqSelect, fields, options.DescendantItems)

			if err != nil {
				return sqSelect, err
			}
		}

		if options.DescendantPictures != nil || options.Fields.CurrentPicturesCount {
			piAlias := ""
			sqSelect, piAlias = applyItemPicture(ipcdAlias, "item_id", sqSelect, options.DescendantPictures)

			if options.Fields.CurrentPicturesCount {
				columns = append(columns, goqu.COUNT(goqu.DISTINCT(goqu.T(piAlias).Col("picture_id"))).As("current_pictures_count"))
			}
		}

	}

	if options.AncestorItems != nil {
		ipcaAlias := alias + "_ipca"
		iAlias := alias + "_ia"
		sqSelect = sqSelect.
			Join(
				goqu.T(tableItemParentCache).As(ipcaAlias),
				goqu.On(goqu.T(alias).Col(colID).Eq(goqu.T(ipcaAlias).Col("item_id"))),
			).
			Join(
				goqu.T(tableItem).As(iAlias),
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
				goqu.T(tableItemParent).As(ipnpAlias),
				goqu.On(goqu.T(alias).Col(colID).Eq(goqu.T(ipnpAlias).Col("item_id"))),
			).
			Where(goqu.T(ipnpAlias).Col("parent_id").IsNull())
	}

	if len(options.Catname) > 0 {
		sqSelect = sqSelect.Where(goqu.T(alias).Col(colCatname).Eq(options.Catname))
	}

	if fields {
		if options.Fields.NameText || options.Fields.NameHTML {
			isAlias := alias + "_is"

			columns = append(columns,
				goqu.C("begin_year"), goqu.C("end_year"),
				goqu.C("begin_month"), goqu.C("end_month"),
				goqu.C("begin_model_year"), goqu.C("end_model_year"),
				goqu.C("begin_model_year_fraction"), goqu.C("end_model_year_fraction"),
				goqu.C("today"),
				goqu.C("body"),
				goqu.T(isAlias).Col("short_name").As("spec_short_name"),
			)

			if options.Fields.NameHTML {
				columns = append(columns, goqu.T(isAlias).Col("name").As("spec_name"))
			}

			sqSelect = sqSelect.
				LeftJoin(
					goqu.T("spec").As(isAlias),
					goqu.On(goqu.T(alias).Col(colSpecID).Eq(goqu.T(isAlias).Col("id"))),
				)
		}

		if options.Fields.Description {
			ilAlias := alias + "_ild"

			columns = append(columns,
				s.db.Select(goqu.T("textstorage_text").Col("text")).
					From(goqu.T(tableItemLanguage).As(ilAlias)).
					Join(
						goqu.T("textstorage_text"),
						goqu.On(goqu.T(ilAlias).Col("text_id").Eq(goqu.T("textstorage_text").Col("id"))),
					).
					Where(
						goqu.T(ilAlias).Col("item_id").Eq(goqu.T(alias).Col(colID)),
						goqu.Func("length", goqu.T("textstorage_text").Col("text")).Gt(0),
					).
					Order(goqu.L(ilAlias+".language = ?", options.Language).Desc()).
					Limit(1).
					As("description"),
			)
		}

		if options.Fields.FullText {
			ilAlias := alias + "_ilf"
			columns = append(columns,
				s.db.Select(goqu.T("textstorage_text").Col("text")).
					From(goqu.T(tableItemLanguage).As(ilAlias)).
					Join(
						goqu.T("textstorage_text"),
						goqu.On(goqu.T(ilAlias).Col("full_text_id").Eq(goqu.T("textstorage_text").Col("id"))),
					).
					Where(
						goqu.T(ilAlias).Col("item_id").Eq(goqu.T(alias).Col(colID)),
						goqu.Func("length", goqu.T("textstorage_text").Col("text")).Gt(0),
					).
					Order(goqu.L(ilAlias+".language = ?", options.Language).Desc()).
					Limit(1).
					As("full_text"),
			)
		}

		if options.Fields.ChildItemsCount {
			columns = append(columns, goqu.L("count(distinct "+ipcAlias+".item_id)").As("child_items_count"))
		}

		if options.Fields.NewChildItemsCount {
			columns = append(
				columns,
				goqu.L("count(distinct IF("+ipcAlias+".timestamp > DATE_SUB(NOW(), INTERVAL ? DAY), "+
					ipcAlias+".item_id, NULL))", NewDays).
					As("new_child_items_count"))
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
					FROM item_language
					WHERE item_id = `+alias+`.id AND length(name) > 0
					ORDER BY FIELD(language, `+strings.Repeat(",?", len(s))[1:]+`)
					LIMIT 1),
					`+alias+`.name
				)
			`, s...).As("name_only"))
		}

		if options.Fields.ItemsCount {
			columns = append(columns, goqu.L("count(distinct "+alias+".id)").As("items_count"))
		}

		if options.Fields.NewItemsCount {
			columns = append(columns, goqu.L(`
				count(distinct if(`+alias+`.add_datetime > date_sub(NOW(), INTERVAL ? DAY), `+alias+`.id, null))
			`, NewDays).As("new_items_count"))
		}

		if options.Fields.DescendantsCount {
			columns = append(columns, goqu.L(`
				(
					SELECT count(distinct product1.id)
					FROM item AS product1
						JOIN `+tableItemParentCache+` ON product1.id = `+tableItemParentCache+`.item_id
					WHERE `+tableItemParentCache+`.parent_id = `+alias+`.id
						AND `+tableItemParentCache+`.item_id <> `+tableItemParentCache+`.parent_id
					LIMIT 1
				) 
			`).As("descendants_count"))
		}

		if options.Fields.NewDescendantsCount {
			columns = append(columns, goqu.L(`
				(
					SELECT count(distinct product2.id)
					FROM item AS product2
						JOIN `+tableItemParentCache+` ON product2.id = `+tableItemParentCache+`.item_id
					WHERE `+tableItemParentCache+`.parent_id = `+alias+`.id
						AND `+tableItemParentCache+`.item_id <> `+tableItemParentCache+`.parent_id
						AND product2.add_datetime > DATE_SUB(NOW(), INTERVAL ? DAY)
				) 
			`, NewDays).As("new_descendants_count"))
		}

		sqSelect = sqSelect.SelectAppend(columns...)
	}

	return sqSelect, nil
}

func (s *Repository) Count(ctx context.Context, options ListOptions) (int, error) {
	var err error

	sqSelect := s.db.Select(goqu.L("COUNT(1)")).From(goqu.T(tableItem).As("i"))

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

func (s *Repository) CountDistinct(ctx context.Context, options ListOptions) (int, error) {
	var err error

	sqSelect := s.db.Select(goqu.L("COUNT(DISTINCT i.id)")).From(goqu.T(tableItem).As("i"))

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

	res, err := s.List(ctx, options)
	if err != nil {
		return Item{}, err
	}

	if len(res) == 0 {
		return Item{}, ErrItemNotFound
	}

	return res[0], nil
}

func (s *Repository) List(ctx context.Context, options ListOptions) ([]Item, error) { //nolint:maintidx
	/*langPriority, ok := languagePriority[options.Language]
	if !ok {
		return nil, fmt.Errorf("language `%s` not found", options.Language)
	}*/
	var err error

	alias := "i"

	sqSelect := s.db.Select(
		goqu.T(alias).Col(colID),
		goqu.T(alias).Col(colCatname),
		goqu.T(alias).Col(colEngineItemID),
		goqu.T(alias).Col(colItemTypeID),
		goqu.T(alias).Col(colIsConcept),
		goqu.T(alias).Col(colIsConceptInherit),
		goqu.T(alias).Col(colSpecID),
	).From(goqu.T(tableItem).As(alias)).
		GroupBy(goqu.T(alias).Col(colID))

	sqSelect, err = s.applyItem("i", sqSelect, true, &options)
	if err != nil {
		return nil, err
	}

	if len(options.OrderBy) > 0 {
		sqSelect = sqSelect.Order(options.OrderBy...)
	}

	if options.Limit > 0 {
		sqSelect = sqSelect.Limit(uint(options.Limit))
	}

	rows, err := sqSelect.Executor().QueryContext(ctx) //nolint:sqlclosecheck
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
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
		)

		pointers := make([]interface{}, len(columnNames))

		for i, colName := range columnNames {
			switch colName {
			case colID:
				pointers[i] = &r.ID
			case "name_only":
				pointers[i] = &r.NameOnly
			case colCatname:
				pointers[i] = &catname
			case colEngineItemID:
				pointers[i] = &engineItemID
			case colItemTypeID:
				pointers[i] = &r.ItemTypeID
			case colIsConcept:
				pointers[i] = &r.IsConcept
			case colIsConceptInherit:
				pointers[i] = &r.IsConceptInherit
			case colSpecID:
				pointers[i] = &specID
			case "description":
				pointers[i] = &description
			case "full_text":
				pointers[i] = &fullText
			case "items_count":
				pointers[i] = &r.ItemsCount
			case "new_items_count":
				pointers[i] = &r.NewItemsCount
			case "descendants_count":
				pointers[i] = &r.DescendantsCount
			case "new_descendants_count":
				pointers[i] = &r.NewDescendantsCount
			case "child_items_count":
				pointers[i] = &r.ChildItemsCount
			case "new_child_items_count":
				pointers[i] = &r.NewChildItemsCount
			case "begin_year":
				pointers[i] = &beginYear
			case "end_year":
				pointers[i] = &endYear
			case "begin_month":
				pointers[i] = &beginMonth
			case "end_month":
				pointers[i] = &endMonth
			case "begin_model_year":
				pointers[i] = &beginModelYear
			case "end_model_year":
				pointers[i] = &endModelYear
			case "begin_model_year_fraction":
				pointers[i] = &beginModelYearFraction
			case "end_model_year_fraction":
				pointers[i] = &endModelYearFraction
			case "today":
				pointers[i] = &today
			case "body":
				pointers[i] = &r.Body
			case "spec_name":
				pointers[i] = &specName
			case "spec_short_name":
				pointers[i] = &specShortName
			case "current_pictures_count":
				pointers[i] = &r.CurrentPicturesCount
			default:
				pointers[i] = nil
			}
		}

		err = rows.Scan(pointers...)
		if err != nil {
			return nil, err
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

		result = append(result, r)
	}

	if err = rows.Err(); err != nil {
		return nil, err
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

	return result, nil
}

func (s *Repository) Tree(ctx context.Context, id string) (*TreeItem, error) {
	type row struct {
		ID       int64    `db:"id"`
		Name     string   `db:"name"`
		ItemType ItemType `db:"item_type_id"`
	}

	var item row

	success, err := s.db.Select(colID, "name", "item_type_id").From(tableItem).
		Where(goqu.I(colID).Eq(id)).ScanStructContext(ctx, item)
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
	res, err := s.db.From("vehicle_vehicle_type").Delete().
		Where(
			goqu.I("vehicle_id").Eq(itemID),
			goqu.I("vehicle_type_id").Eq(vehicleTypeID),
			goqu.L("NOT inherited"),
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
			INSERT INTO vehicle_vehicle_type (vehicle_id, vehicle_type_id, inherited)
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
			"DELETE FROM vehicle_vehicle_type WHERE vehicle_id = ? AND inherited",
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
		"SELECT item_id FROM item_parent WHERE parent_id = ?",
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
	sqlSelect := s.db.From("vehicle_vehicle_type").Select("vehicle_type_id").Where(
		goqu.I("vehicle_id").Eq(itemID),
	)
	if inherited {
		sqlSelect = sqlSelect.Where(goqu.L("inherited"))
	} else {
		sqlSelect = sqlSelect.Where(goqu.L("NOT inherited"))
	}

	res := make([]int64, 0)

	err := sqlSelect.ScanValsContext(ctx, &res)

	return res, err
}

func (s *Repository) getItemVehicleTypeInheritedIDs(ctx context.Context, itemID int64) ([]int64, error) {
	sqlSelect := s.db.From("vehicle_vehicle_type").
		Select("vehicle_type_id").Distinct().
		Join(
			goqu.T(tableItemParent),
			goqu.On(goqu.Ex{"vehicle_vehicle_type.vehicle_id": goqu.T(tableItemParent).Col("parent_id")}),
		).
		Where(goqu.T(tableItemParent).Col("item_id").Eq(itemID))

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

	sqlDelete := s.db.From("vehicle_vehicle_type").Delete().
		Where(goqu.I("vehicle_id").Eq(itemID))

	if len(types) > 0 {
		sqlDelete = sqlDelete.Where(goqu.I("vehicle_type_id").NotIn(types))
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
	rows, err := s.db.Select("parent_id", "type").
		From(tableItemParent).
		Where(goqu.Ex{"item_id": id}).
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

	err := s.db.Select("item_id").
		From(tableItemParent).
		Where(goqu.Ex{"parent_id": parentID}).
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
   		INSERT INTO `+tableItemParentCache+` (item_id, parent_id, diff, tuning, sport, design)
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

	_, err = s.db.Delete(tableItemParentCache).Where(goqu.Ex{
		"item_id":   itemID,
		"parent_id": goqu.Op{"notIn": keys},
	}).Executor().ExecContext(ctx)
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
	sqSelect := s.db.Select("item_id", "language", "name", "text_id", "full_text_id").
		From(goqu.T(tableItemLanguage)).Where(
		goqu.I("item_id").Eq(itemID),
		goqu.I("language").Neq("xx"),
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
	sqSelect := s.db.Select("item_id", "parent_id", "language", "name").
		From(goqu.T("item_parent_language")).Where(
		goqu.I("item_id").Eq(itemID),
		goqu.I("parent_id").Eq(parentID),
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
