package items

import (
	"database/sql"
	"fmt"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/util"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"regexp"
	"sort"
	"strings"
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

const (
	VEHICLE   int = 1
	ENGINE    int = 2
	CATEGORY  int = 3
	TWINS     int = 4
	BRAND     int = 5
	FACTORY   int = 6
	MUSEUM    int = 7
	PERSON    int = 8
	COPYRIGHT int = 9
)

// Repository Main Object
type Repository struct {
	db *sql.DB
}

type TopBrandsListItem struct {
	ID            int32
	Catname       string
	Name          string
	ItemsCount    int32
	NewItemsCount int32
}

type TopBrandsListResult struct {
	Brands []TopBrandsListItem
	Total  int
}

type TopPersonsListItem struct {
	ID   int64
	Name string
}

type TopPersonsListResult struct {
	Items []TopPersonsListItem
	Total int
}

// NewRepository constructor
func NewRepository(
	autowpDB *sql.DB,
) *Repository {
	return &Repository{
		db: autowpDB,
	}
}

func (s *Repository) TopBrandList(lang string) (*TopBrandsListResult, error) {

	langPriority, ok := languagePriority[lang]
	if !ok {
		return nil, fmt.Errorf("language `%s` not found", lang)
	}

	queryArgs := make([]interface{}, 0)
	queryArgs = append(queryArgs, NewDays)
	for _, l := range langPriority {
		queryArgs = append(queryArgs, l)
	}
	queryArgs = append(queryArgs, BRAND)
	queryArgs = append(queryArgs, TopBrandsCount)

	rows, err := s.db.Query(`
		SELECT id, catname, name, (
		    SELECT count(distinct product1.id)
		    FROM item AS product1
		    	JOIN item_parent_cache ON product1.id = item_parent_cache.item_id
			WHERE item_parent_cache.parent_id = item.id
				AND item_parent_cache.item_id <> item_parent_cache.parent_id
		    LIMIT 1
		) AS cars_count, (
			SELECT count(distinct product2.id)
			FROM item AS product2
				JOIN item_parent_cache ON product2.id = item_parent_cache.item_id
			WHERE item_parent_cache.parent_id = item.id
			  	AND item_parent_cache.item_id <> item_parent_cache.parent_id
				AND product2.add_datetime > DATE_SUB(NOW(), INTERVAL ? DAY)
		), (
		    SELECT name
            FROM item_language
            WHERE item_id = item.id AND length(name) > 0
            ORDER BY FIELD(language`+strings.Repeat(", ?", len(langPriority))+`)
            LIMIT 1
		)
		FROM item
		WHERE item_type_id = ?
		GROUP BY item.id
		ORDER BY cars_count DESC
		LIMIT ?
	`, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	var result []TopBrandsListItem
	for rows.Next() {
		var r TopBrandsListItem
		var langName sql.NullString
		var newCount sql.NullInt32
		err = rows.Scan(&r.ID, &r.Catname, &r.Name, &r.ItemsCount, &newCount, &langName)
		if err != nil {
			return nil, err
		}

		if langName.Valid && len(langName.String) > 0 {
			r.Name = langName.String
		}

		r.NewItemsCount = 0
		if newCount.Valid {
			r.NewItemsCount = newCount.Int32
		}

		result = append(result, r)
	}

	tag := language.English
	switch lang {
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

		switch lang {
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

	var total int
	err = s.db.QueryRow("SELECT count(1) FROM item WHERE item_type_id = ?", BRAND).Scan(&total)
	if err != nil {
		return nil, err
	}

	return &TopBrandsListResult{
		Brands: result,
		Total:  total,
	}, nil
}

func (s *Repository) TopPersonsList(lang string, pictureItemType pictures.PictureItemType) (*TopPersonsListResult, error) {

	langPriority, ok := languagePriority[lang]
	if !ok {
		return nil, fmt.Errorf("language `%s` not found", lang)
	}

	queryArgs := make([]interface{}, 0)
	for _, l := range langPriority {
		queryArgs = append(queryArgs, l)
	}
	queryArgs = append(queryArgs, PERSON, pictures.STATUS_ACCEPTED, pictureItemType, TopPersonsCount)

	rows, err := s.db.Query(`
		SELECT item.id, item.name, (
		    SELECT name
            FROM item_language
            WHERE item_id = item.id AND length(name) > 0
            ORDER BY FIELD(language`+strings.Repeat(", ?", len(langPriority))+`)
            LIMIT 1
		)
		FROM item
			INNER JOIN picture_item ON item.id = picture_item.item_id
			INNER JOIN pictures ON picture_item.picture_id = pictures.id
		WHERE item.item_type_id = ? AND pictures.status = ? AND picture_item.type = ?
		GROUP BY item.id
		ORDER BY COUNT(1) DESC
		LIMIT ?
	`, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	var result []TopPersonsListItem
	for rows.Next() {
		var r TopPersonsListItem
		var langName sql.NullString
		err = rows.Scan(&r.ID, &r.Name, &langName)
		if err != nil {
			return nil, err
		}

		if langName.Valid && len(langName.String) > 0 {
			r.Name = langName.String
		}

		result = append(result, r)
	}

	return &TopPersonsListResult{
		Items: result,
	}, nil
}
