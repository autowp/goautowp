package query

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
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
	Alias                        string
	Language                     string
	ItemID                       int64
	ItemIDs                      []int64
	ItemIDExpr                   goqu.Expression
	TypeID                       []schema.ItemTableItemTypeID
	PictureItems                 *PictureItemListOptions
	PreviewPictures              *PictureItemListOptions
	Limit                        uint32
	Page                         uint32
	SortByName                   bool
	ItemParentChild              *ItemParentListOptions
	ItemParentParent             *ItemParentListOptions
	ItemParentCacheDescendant    *ItemParentCacheListOptions
	ItemParentCacheAncestor      *ItemParentCacheListOptions
	NoParents                    bool
	Catname                      string
	Name                         string
	IsConcept                    bool
	IsNotConcept                 bool
	EngineItemID                 int64
	HasBeginYear                 bool
	HasEndYear                   bool
	HasBeginMonth                bool
	HasEndMonth                  bool
	HasLogo                      bool
	CreatedInDays                int
	VehicleTypeAncestorID        int64
	ExcludeVehicleTypeAncestorID []int64
	VehicleTypeIsNull            bool
	ParentTypesOf                schema.ItemTableItemTypeID
	IsGroup                      bool
	ExcludeSelfAndChilds         int64
	Autocomplete                 string
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

func (s *ItemsListOptions) ExistsSelect(db *goqu.Database) *goqu.SelectDataset {
	return s.Select(db).Select(goqu.V(true))
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

	if len(s.ItemIDs) > 0 {
		sqSelect = sqSelect.Where(aliasIDCol.In(s.ItemIDs))
	}

	if s.ItemIDExpr != nil {
		sqSelect = sqSelect.Where(aliasIDCol.Eq(s.ItemIDExpr))
	}

	if len(s.TypeID) > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableItemTypeIDColName).In(s.TypeID))
	}

	sqSelect = s.applyParentTypesOf(alias, sqSelect)

	if s.CreatedInDays > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableAddDatetimeColName).Gt(
			goqu.Func("DATE_SUB", goqu.Func("NOW"), goqu.L("INTERVAL ? DAY", s.CreatedInDays)),
		))
	}

	if s.VehicleTypeAncestorID > 0 {
		sqSelect = sqSelect.
			Join(
				schema.VehicleVehicleTypeTable,
				goqu.On(aliasTable.Col(schema.ItemTableIDColName).Eq(schema.VehicleVehicleTypeTableVehicleIDCol)),
			).
			Join(
				schema.CarTypesParentsTable,
				goqu.On(schema.VehicleVehicleTypeTableVehicleTypeIDCol.Eq(schema.CarTypesParentsTableIDCol)),
			).
			Where(schema.CarTypesParentsTableParentIDCol.Eq(s.VehicleTypeAncestorID))
	}

	sqSelect = s.applyExcludeVehicleTypeAncestorID(alias, sqSelect)

	if s.VehicleTypeIsNull {
		sqSelect = sqSelect.
			LeftJoin(
				schema.VehicleVehicleTypeTable,
				goqu.On(aliasTable.Col(schema.ItemTableIDColName).Eq(schema.VehicleVehicleTypeTableVehicleIDCol)),
			).
			Where(schema.VehicleVehicleTypeTableVehicleIDCol.IsNull())
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
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableIsConceptColName).IsTrue())
	}

	if s.IsNotConcept {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableIsConceptColName).IsFalse())
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

	if s.IsGroup {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableIsGroupColName).IsTrue())
	}

	if s.ExcludeSelfAndChilds != 0 {
		esacAlias := "esac"
		esacAliasTable := goqu.T(esacAlias)
		esacAliasTableItemIDCol := esacAliasTable.Col(schema.ItemParentCacheTableItemIDColName)
		sqSelect = sqSelect.
			LeftJoin(schema.ItemParentCacheTable.As(esacAlias), goqu.On(
				aliasTable.Col(schema.ItemTableIDColName).Eq(esacAliasTableItemIDCol),
				esacAliasTable.Col(schema.ItemParentCacheTableParentIDColName).Eq(s.ExcludeSelfAndChilds),
			)).
			Where(esacAliasTableItemIDCol.IsNull())
	}

	sqSelect = s.applyAutocompleteFilter(alias, sqSelect)

	return sqSelect
}

func (s *ItemsListOptions) applyExcludeVehicleTypeAncestorID(
	alias string, sqSelect *goqu.SelectDataset,
) *goqu.SelectDataset {
	if len(s.ExcludeVehicleTypeAncestorID) == 0 {
		return sqSelect
	}

	aliasTable := goqu.T(alias)
	subSelect := sqSelect.ClearSelect().ClearLimit().ClearOffset().ClearOrder().ClearWhere().GroupBy().FromSelf()
	subSelect = subSelect.Select(schema.CarTypesParentsTableIDCol).
		From(schema.CarTypesParentsTable).
		Where(schema.CarTypesParentsTableParentIDCol.In(s.ExcludeVehicleTypeAncestorID))

	return sqSelect.
		Join(
			schema.VehicleVehicleTypeTable,
			goqu.On(aliasTable.Col(schema.ItemTableIDColName).Eq(schema.VehicleVehicleTypeTableVehicleIDCol)),
		).
		Join(
			schema.CarTypesParentsTable,
			goqu.On(schema.VehicleVehicleTypeTableVehicleTypeIDCol.Eq(schema.CarTypesParentsTableIDCol)),
		).
		Where(schema.VehicleVehicleTypeTableVehicleTypeIDCol.NotIn(subSelect))
}

func (s *ItemsListOptions) applyAutocompleteFilter(
	alias string, sqSelect *goqu.SelectDataset,
) *goqu.SelectDataset {
	if s.Autocomplete == "" {
		return sqSelect
	}

	query := s.Autocomplete

	var (
		beginYear      int
		endYear        int
		today          = false
		body           string
		beginModelYear int
		endModelYear   int
	)

	var err error

	regex := regexp.MustCompile(
		`^(([0-9]{4})([-–]([^[:space:]]{2,4}))?[[:space:]]+)?(.*?)( \((.+)\))?( '([0-9]{4})(–(.+))?)?$`,
	)
	match := regex.FindStringSubmatch(query)

	if match != nil {
		query = strings.TrimSpace(match[5])
		body = strings.TrimSpace(match[7])

		beginYearStr := match[9]
		if beginYearStr != "" {
			beginYear, err = strconv.Atoi(beginYearStr)
			if err != nil {
				beginYear = 0
			}
		}

		endYearStr := match[11]

		beginModelYearStr := match[2]
		if beginModelYearStr != "" {
			beginModelYear, err = strconv.Atoi(beginModelYearStr)
			if err != nil {
				beginModelYear = 0
			}
		}

		endModelYearStr := match[4]

		if endYearStr == "н.в." {
			today = true
		} else {
			eyLength := len(endYearStr)
			if eyLength > 0 {
				endYear, err = strconv.Atoi(endYearStr)
				if err != nil {
					endYear = 0
				}

				if eyLength == 2 {
					endYear = beginYear - beginYear%100 + endYear
				}
			}
		}

		if endModelYearStr == "н.в." {
			today = true
		} else {
			eyLength := len(endModelYearStr)
			if eyLength > 0 {
				endModelYear, err = strconv.Atoi(endModelYearStr)
				if err != nil {
					endModelYear = 0
				}

				if eyLength == 2 {
					endModelYear = beginModelYear - beginModelYear%100 + endModelYear
				}
			}
		}
	}

	aliasTable := goqu.T(alias)

	if query != "" {
		ilAlias := alias + "il"
		ilAliasTable := goqu.T(ilAlias)
		sqSelect = sqSelect.
			Join(schema.ItemLanguageTable.As(ilAlias), goqu.On(
				aliasTable.Col(schema.ItemTableIDColName).Eq(ilAliasTable.Col(schema.ItemLanguageTableItemIDColName)),
			)).
			Where(ilAliasTable.Col(schema.ItemLanguageTableNameColName).ILike(query + "%"))
	}

	if beginYear > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableBeginYearColName).Eq(beginYear))
	}

	if today {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableTodayColName).IsTrue())
	} else if endYear > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableEndYearColName).Eq(endYear))
	}

	if body != "" {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableBodyColName).ILike(body + "%"))
	}

	if beginModelYear > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableBeginModelYearColName).Eq(beginModelYear))
	}

	if endModelYear > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableEndModelYearColName).Eq(endModelYear))
	}

	return sqSelect
}

func (s *ItemsListOptions) applyParentTypesOf(
	alias string, sqSelect *goqu.SelectDataset,
) *goqu.SelectDataset {
	aliasTable := goqu.T(alias)

	if s.ParentTypesOf != 0 {
		parentTypes := make([]schema.ItemTableItemTypeID, 0)

		for parentType, childTypes := range schema.AllowedTypeCombinations {
			if util.Contains(childTypes, s.ParentTypesOf) {
				parentTypes = append(parentTypes, parentType)
			}
		}

		if len(parentTypes) > 0 {
			sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemTableItemTypeIDColName).In(parentTypes))
		} else {
			sqSelect = sqSelect.Where(goqu.V(false))
		}
	}

	return sqSelect
}
