package items

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/util"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"regexp"
	"sort"
)

const TopBrandsCount = 150
const NewDays = 7
const TopPersonsCount = 5

var languagePriority = map[string][]string{
	"xx":    {"en", "it", "fr", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", "xx"},
	"en":    {"en", "it", "fr", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", "xx"},
	"fr":    {"fr", "en", "it", "de", "es", "pt", "ru", "be", "uk", "zh", "jp", "xx"},
	"pt-br": {"pt", "en", "it", "fr", "de", "es", "ru", "be", "uk", "zh", "jp", "xx"},
	"ru":    {"ru", "en", "it", "fr", "de", "es", "pt", "be", "uk", "zh", "jp", "xx"},
	"be":    {"be", "ru", "uk", "en", "it", "fr", "de", "es", "pt", "zh", "jp", "xx"},
	"uk":    {"uk", "ru", "en", "it", "fr", "de", "es", "pt", "be", "zh", "jp", "xx"},
	"zh":    {"zh", "en", "it", "fr", "de", "es", "pt", "ru", "be", "uk", "jp", "xx"},
	"es":    {"es", "en", "it", "fr", "de", "pt", "ru", "be", "uk", "zh", "jp", "xx"},
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

// Repository Main Object
type Repository struct {
	db *sql.DB
}

type Item struct {
	ID                  int64
	Catname             string
	Name                string
	DescendantsCount    int32
	NewDescendantsCount int32
}

// NewRepository constructor
func NewRepository(
	autowpDB *sql.DB,
) *Repository {
	return &Repository{
		db: autowpDB,
	}
}

type PicturesOptions struct {
	Status      pictures.Status
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
	Name                bool
	NameHtml            bool
	NameDefault         bool
	Description         bool
	HasText             bool
	PreviewPictures     ListPreviewPicturesFields
	TotalPictures       bool
	DescendantsCount    bool
	NewDescendantsCount bool
}

type ListOptions struct {
	Language           string
	Fields             ListFields
	TypeID             ItemType
	DescendantPictures *ItemPicturesOptions
	PreviewPictures    *ItemPicturesOptions
	Limit              uint64
	OrderBy            string
	SortByName         bool
}

func applyPicture(alias string, sqSelect sq.SelectBuilder, options *PicturesOptions) sq.SelectBuilder {
	joinPicture := false

	pAlias := alias + "_p"

	if options.Status != "" {
		joinPicture = true
		sqSelect = sqSelect.Where(sq.Eq{pAlias + ".status": options.Status})
	}

	if options.ItemPicture != nil {
		joinPicture = true
		sqSelect = applyItemPicture(pAlias, sqSelect, options.ItemPicture)
	}

	if joinPicture {
		sqSelect = sqSelect.Join("pictures AS " + pAlias + " ON " + alias + ".picture_id = " + pAlias + ".id")
	}

	return sqSelect
}

func applyItemPicture(alias string, sqSelect sq.SelectBuilder, options *ItemPicturesOptions) sq.SelectBuilder {
	piAlias := alias + "_pi"

	sqSelect = sqSelect.Join("picture_item AS " + piAlias + " ON " + alias + ".id = " + piAlias + ".item_id")

	if options.TypeID != 0 {
		sqSelect = sqSelect.Where(sq.Eq{piAlias + ".type": options.TypeID})
	}

	if options.PerspectiveID != 0 {
		sqSelect = sqSelect.Where(sq.Eq{piAlias + ".perspective_id": options.PerspectiveID})
	}

	if options.Pictures != nil {
		sqSelect = applyPicture(piAlias, sqSelect, options.Pictures)
	}

	return sqSelect
}

func applyItem(alias string, sqSelect sq.SelectBuilder, fields bool, options *ListOptions) (sq.SelectBuilder, error) {
	if options.TypeID != 0 {
		sqSelect = sqSelect.Where(sq.Eq{alias + ".item_type_id": options.TypeID})
	}

	if options.DescendantPictures != nil {
		sqSelect = applyItemPicture(alias, sqSelect, options.DescendantPictures)
	}

	if fields {
		if options.Fields.Name {

			langPriority, ok := languagePriority[options.Language]
			if !ok {
				return sqSelect, fmt.Errorf("language `%s` not found", options.Language)
			}

			s := make([]interface{}, len(langPriority))
			for i, v := range langPriority {
				s[i] = v
			}

			sqSelect = sqSelect.Column(`
				IFNULL(
					(SELECT name
					FROM item_language
					WHERE item_id = `+alias+`.id AND length(name) > 0
					ORDER BY FIELD(language, `+sq.Placeholders(len(s))+`)
					LIMIT 1),
					`+alias+`.name
				) AS name
			`, s...)
		}

		if options.Fields.DescendantsCount {
			sqSelect = sqSelect.Column(`
				(
					SELECT count(distinct product1.id)
					FROM item AS product1
						JOIN item_parent_cache ON product1.id = item_parent_cache.item_id
					WHERE item_parent_cache.parent_id = ` + alias + `.id
						AND item_parent_cache.item_id <> item_parent_cache.parent_id
					LIMIT 1
				) AS descendants_count
			`)
		}

		if options.Fields.NewDescendantsCount {
			sqSelect = sqSelect.Column(`
				(
					SELECT count(distinct product2.id)
					FROM item AS product2
						JOIN item_parent_cache ON product2.id = item_parent_cache.item_id
					WHERE item_parent_cache.parent_id = `+alias+`.id
						AND item_parent_cache.item_id <> item_parent_cache.parent_id
						AND product2.add_datetime > DATE_SUB(NOW(), INTERVAL ? DAY)
				) AS new_descendants_count
			`, NewDays)
		}
	}

	return sqSelect, nil
}

func (s *Repository) Count(options ListOptions) (int, error) {
	var err error
	sqSelect := sq.Select("COUNT(1)").From("item AS i")

	sqSelect, err = applyItem("i", sqSelect, false, &options)
	if err != nil {
		return 0, err
	}

	var count int
	err = sqSelect.RunWith(s.db).QueryRow().Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Repository) List(options ListOptions) ([]Item, error) {
	/*langPriority, ok := languagePriority[options.Language]
	if !ok {
		return nil, fmt.Errorf("language `%s` not found", options.Language)
	}*/
	var err error
	sqSelect := sq.Select("i.id", "i.catname").From("item AS i").GroupBy("i.id")

	sqSelect, err = applyItem("i", sqSelect, true, &options)
	if err != nil {
		return nil, err
	}

	if len(options.OrderBy) > 0 {
		sqSelect = sqSelect.OrderBy(options.OrderBy)
	}
	if options.Limit > 0 {
		sqSelect = sqSelect.Limit(options.Limit)
	}

	rows, err := sqSelect.RunWith(s.db).Query()
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
		var catname sql.NullString

		pointers := make([]interface{}, len(columnNames))
		for i, colName := range columnNames {
			switch colName {
			case "id":
				pointers[i] = &r.ID
			case "name":
				pointers[i] = &r.Name
			case "catname":
				pointers[i] = &catname
			case "descendants_count":
				pointers[i] = &r.DescendantsCount
			case "new_descendants_count":
				pointers[i] = &r.NewDescendantsCount
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

		result = append(result, r)
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
		}

		cyrillic := regexp.MustCompile(`^\p{Cyrillic}`)
		han := regexp.MustCompile(`^\p{Han}`)

		cl := collate.New(tag, collate.IgnoreCase, collate.IgnoreDiacritics)
		sort.SliceStable(result, func(i, j int) bool {
			a := result[i].Name
			b := result[j].Name

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
