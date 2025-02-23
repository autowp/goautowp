package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

const VehicleTypeTableAlias = "vt"

type VehicleTypeListOptions struct {
	Catname  string
	NoParent bool
	ParentID int64
	Childs   *VehicleTypeParentsListOptions
}

func (s *VehicleTypeListOptions) Select(db *goqu.Database, alias string) (*goqu.SelectDataset, error) {
	return s.apply(alias, db.From(schema.CarTypesTable.As(alias)))
}

func (s *VehicleTypeListOptions) apply(alias string, sqSelect *goqu.SelectDataset) (*goqu.SelectDataset, error) {
	aliasTable := goqu.T(alias)

	var err error

	if len(s.Catname) > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.CarTypesTableCatnameColName).Eq(s.Catname))
	}

	if s.NoParent {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.CarTypesTableParentIDColName).IsNull())
	}

	if s.ParentID > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.CarTypesTableParentIDColName).Eq(s.ParentID))
	}

	if s.Childs != nil {
		sqSelect, err = s.Childs.JoinToParentIDAndApply(
			aliasTable.Col(schema.CarTypesTableIDColName), alias+"_"+VehicleTypeParentTableAlias, sqSelect,
		)
		if err != nil {
			return nil, err
		}
	}

	return sqSelect, nil
}
