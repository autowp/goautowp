package query

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

const (
	itemParentNoParentAliasSuffix = "_ipnp"
	ItemAlias                     = "i"
)

func AppendItemAlias(alias string, suffix string) string {
	return alias + "_" + ItemAlias + suffix
}

type ItemListOptions struct {
	Alias                        string
	Language                     string
	ItemID                       int64
	ExcludeID                    int64
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
	SuggestionsTo                int64
}

func ItemParentNoParentAlias(alias string) string {
	return alias + itemParentNoParentAliasSuffix
}

func (s *ItemListOptions) Select(db *goqu.Database, alias string) (*goqu.SelectDataset, error) {
	var err error

	if s.Alias != "" {
		alias = s.Alias
	}

	sqSelect := db.Select().From(schema.ItemTable.As(alias))
	sqSelect, _, err = s.apply(alias, sqSelect)

	return sqSelect, err
}

func (s *ItemListOptions) ExistsSelect(db *goqu.Database, alias string) (*goqu.SelectDataset, error) {
	sqSelect, err := s.Select(db, alias)
	if err != nil {
		return nil, err
	}

	return sqSelect.Select(goqu.V(true)), nil
}

func (s *ItemListOptions) CountSelect(db *goqu.Database, alias string) (*goqu.SelectDataset, error) {
	sqSelect, err := s.Select(db, alias)
	if err != nil {
		return nil, err
	}

	return sqSelect.Select(goqu.COUNT(goqu.Star())), nil
}

func (s *ItemListOptions) CountDistinctSelect(db *goqu.Database, alias string) (*goqu.SelectDataset, error) {
	if s.Alias != "" {
		alias = s.Alias
	}

	sqSelect, err := s.Select(db, alias)
	if err != nil {
		return nil, err
	}

	return sqSelect.Select(
		goqu.COUNT(goqu.DISTINCT(goqu.T(alias).Col(schema.ItemTableIDColName))),
	), nil
}

func (s *ItemListOptions) JoinToIDAndApply(
	srcCol exp.IdentifierExpression, alias string, sqSelect *goqu.SelectDataset,
) (*goqu.SelectDataset, bool, error) {
	if s == nil {
		return sqSelect, false, nil
	}

	return s.apply(
		alias,
		sqSelect.Join(
			schema.ItemTable.As(alias),
			goqu.On(srcCol.Eq(goqu.T(alias).Col(schema.ItemTableIDColName))),
		),
	)
}

func (s *ItemListOptions) apply(alias string, sqSelect *goqu.SelectDataset) (*goqu.SelectDataset, bool, error) {
	var (
		err        error
		groupBy    = false
		aliasTable = goqu.T(alias)
		aliasIDCol = aliasTable.Col(schema.ItemTableIDColName)
	)

	if s.ItemID > 0 {
		sqSelect = sqSelect.Where(aliasIDCol.Eq(s.ItemID))
	}

	if s.ExcludeID != 0 {
		sqSelect = sqSelect.Where(aliasIDCol.Neq(s.ExcludeID))
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

	sqSelect = s.applyName(alias, sqSelect)

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

	var subGroupBy bool

	sqSelect, subGroupBy, err = s.applyJoins(alias, sqSelect)
	if err != nil {
		return nil, false, err
	}

	if subGroupBy {
		groupBy = true
	}

	sqSelect, subGroupBy = s.applySuggestionsTo(alias, sqSelect)
	if subGroupBy {
		groupBy = true
	}

	sqSelect, subGroupBy = s.applyAutocompleteFilter(alias, sqSelect)
	if subGroupBy {
		groupBy = true
	}

	return sqSelect, groupBy, nil
}

func (s *ItemListOptions) applyJoins(
	alias string, sqSelect *goqu.SelectDataset,
) (*goqu.SelectDataset, bool, error) {
	var (
		err        error
		groupBy    bool
		subGroupBy bool
		aliasTable = goqu.T(alias)
		idCol      = aliasTable.Col(schema.ItemTableIDColName)
	)

	sqSelect, subGroupBy = s.applyVehicleTypeAncestorID(alias, sqSelect)
	if subGroupBy {
		groupBy = true
	}

	sqSelect, subGroupBy = s.applyExcludeVehicleTypeAncestorID(alias, sqSelect)
	if subGroupBy {
		groupBy = true
	}

	if s.VehicleTypeIsNull {
		sqSelect = sqSelect.
			LeftJoin(
				schema.VehicleVehicleTypeTable,
				goqu.On(aliasTable.Col(schema.ItemTableIDColName).Eq(schema.VehicleVehicleTypeTableVehicleIDCol)),
			).
			Where(schema.VehicleVehicleTypeTableVehicleIDCol.IsNull())
	}

	if s.ItemParentChild != nil {
		groupBy = true

		sqSelect, _, err = s.ItemParentChild.JoinToParentIDAndApply(idCol, AppendItemParentAlias(alias, "c"), sqSelect)
		if err != nil {
			return nil, false, err
		}
	}

	sqSelect, subGroupBy, err = s.applyItemParentParent(alias, sqSelect)
	if err != nil {
		return nil, false, err
	}

	if subGroupBy {
		groupBy = true
	}

	if s.PictureItems != nil {
		groupBy = true

		sqSelect, err = s.PictureItems.JoinToItemIDAndApply(idCol, AppendPictureItemAlias(alias, ""), sqSelect)
		if err != nil {
			return nil, false, err
		}
	}

	if s.ItemParentCacheDescendant != nil {
		groupBy = true

		sqSelect, err = s.ItemParentCacheDescendant.JoinToParentIDAndApply(
			idCol,
			AppendItemParentCacheAlias(alias, "d"),
			sqSelect,
		)
		if err != nil {
			return nil, false, err
		}
	}

	if s.ItemParentCacheAncestor != nil {
		groupBy = true

		sqSelect, err = s.ItemParentCacheAncestor.JoinToItemIDAndApply(
			idCol,
			AppendItemParentCacheAlias(alias, "a"),
			sqSelect,
		)
		if err != nil {
			return nil, false, err
		}
	}

	if s.NoParents {
		ipnpAlias := ItemParentNoParentAlias(alias)
		sqSelect = sqSelect.
			LeftJoin(
				schema.ItemParentTable.As(ipnpAlias),
				goqu.On(idCol.Eq(goqu.T(ipnpAlias).Col(schema.ItemParentTableItemIDColName))),
			).
			Where(goqu.T(ipnpAlias).Col(schema.ItemParentTableParentIDColName).IsNull())
	}

	sqSelect = s.applyExcludeSelfAndChilds(alias, sqSelect)

	return sqSelect, groupBy, err
}

func (s *ItemListOptions) applyItemParentParent(
	alias string, sqSelect *goqu.SelectDataset,
) (*goqu.SelectDataset, bool, error) {
	var (
		err        error
		groupBy    = false
		aliasTable = goqu.T(alias)
	)

	if s.ItemParentParent != nil {
		groupBy = true

		sqSelect, _, err = s.ItemParentParent.JoinToItemIDAndApply(
			aliasTable.Col(schema.ItemTableIDColName),
			AppendItemParentAlias(alias, "p"),
			sqSelect,
		)
		if err != nil {
			return nil, false, err
		}
	}

	return sqSelect, groupBy, nil
}

func (s *ItemListOptions) applyVehicleTypeAncestorID(
	alias string, sqSelect *goqu.SelectDataset,
) (*goqu.SelectDataset, bool) {
	groupBy := false

	if s.VehicleTypeAncestorID > 0 {
		aliasTable := goqu.T(alias)
		groupBy = true
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

	return sqSelect, groupBy
}

func (s *ItemListOptions) applyName(
	alias string, sqSelect *goqu.SelectDataset,
) *goqu.SelectDataset {
	if len(s.Name) > 0 {
		subSelect := sqSelect.ClearSelect().ClearLimit().ClearOffset().ClearOrder().ClearWhere().GroupBy().FromSelf()

		// WHERE EXISTS(SELECT item_id FROM item_language WHERE item.id = item_id AND name ILIKE ?)
		sqSelect = sqSelect.Where(
			goqu.L(
				"EXISTS ?",
				subSelect.
					From(schema.ItemLanguageTable).
					Where(
						goqu.T(alias).Col(schema.ItemTableIDColName).Eq(schema.ItemLanguageTableItemIDCol),
						schema.ItemLanguageTableNameCol.ILike(s.Name),
					),
			),
		)
	}

	return sqSelect
}

func (s *ItemListOptions) applyExcludeSelfAndChilds(
	alias string, sqSelect *goqu.SelectDataset,
) *goqu.SelectDataset {
	if s.ExcludeSelfAndChilds != 0 {
		aliasTable := goqu.T(alias)
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

	return sqSelect
}

func (s *ItemListOptions) applySuggestionsTo(
	alias string, sqSelect *goqu.SelectDataset,
) (*goqu.SelectDataset, bool) {
	groupBy := false
	if s.SuggestionsTo != 0 {
		groupBy = true
		aliasTable := goqu.T(alias)
		aliasIDCol := aliasTable.Col(schema.ItemTableIDColName)
		subSelect := sqSelect.ClearSelect().ClearLimit().ClearOffset().ClearOrder().ClearWhere().GroupBy().FromSelf()
		ilsAlias := alias + "ils"
		ils2Alias := alias + "ils2"
		ilsAliasTable := goqu.T(ilsAlias)

		sqSelect = sqSelect.
			Join(schema.ItemLanguageTable.As(ilsAlias), goqu.On(
				aliasIDCol.Eq(ilsAliasTable.Col(schema.ItemLanguageTableItemIDColName)),
			)).
			Join(schema.ItemLanguageTable.As(ils2Alias), goqu.On(
				goqu.Func("INSTR", ilsAliasTable.Col(schema.ItemLanguageTableNameColName),
					goqu.T(ils2Alias).Col(schema.ItemLanguageTableNameColName)),
			)).
			Where(
				aliasTable.Col(schema.ItemTableItemTypeIDColName).Eq(schema.ItemTableItemTypeIDBrand),
				ilsAliasTable.Col(schema.ItemLanguageTableItemIDColName).Eq(s.SuggestionsTo),
				aliasIDCol.In(
					subSelect.Select(schema.ItemTableIDCol).
						From(schema.ItemTable).
						Join(schema.ItemParentCacheTable, goqu.On(schema.ItemTableIDCol.Eq(schema.ItemParentCacheTableParentIDCol))).
						Where(
							schema.ItemTableItemTypeIDCol.Eq(schema.ItemTableItemTypeIDBrand),
							schema.ItemParentCacheTableItemIDCol.Eq(s.SuggestionsTo),
						),
				),
			)
	}

	return sqSelect, groupBy
}

func (s *ItemListOptions) applyExcludeVehicleTypeAncestorID(
	alias string, sqSelect *goqu.SelectDataset,
) (*goqu.SelectDataset, bool) {
	if len(s.ExcludeVehicleTypeAncestorID) == 0 {
		return sqSelect, false
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
			Where(schema.VehicleVehicleTypeTableVehicleTypeIDCol.NotIn(subSelect)),
		true
}

func (s *ItemListOptions) applyAutocompleteFilter(
	alias string, sqSelect *goqu.SelectDataset,
) (*goqu.SelectDataset, bool) {
	if s.Autocomplete == "" {
		return sqSelect, false
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
	groupBy := false

	if query != "" {
		groupBy = true
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

	return sqSelect, groupBy
}

func (s *ItemListOptions) applyParentTypesOf(
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
