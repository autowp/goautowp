package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

const (
	ItemVehicleTypeAlias = "ivt"
)

func AppendItemVehicleTypeAlias(alias string) string {
	return alias + "_" + ItemVehicleTypeAlias
}

type ItemVehicleTypeListOptions struct {
	VehicleTypeID int64
}

func (s *ItemVehicleTypeListOptions) Clone() *ItemVehicleTypeListOptions {
	if s == nil {
		return nil
	}

	clone := *s

	return &clone
}

func (s *ItemVehicleTypeListOptions) JoinToVehicleIDAndApply(
	srcCol exp.IdentifierExpression, alias string, sqSelect *goqu.SelectDataset,
) *goqu.SelectDataset {
	if s == nil {
		return sqSelect
	}

	sqSelect = sqSelect.Join(
		schema.VehicleVehicleTypeTable.As(alias),
		goqu.On(
			srcCol.Eq(goqu.T(alias).Col(schema.VehicleVehicleTypeTableVehicleIDColName)),
		),
	)

	return s.apply(alias, sqSelect)
}

func (s *ItemVehicleTypeListOptions) apply(alias string, sqSelect *goqu.SelectDataset) *goqu.SelectDataset {
	aliasTable := goqu.T(alias)

	if s.VehicleTypeID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.VehicleVehicleTypeTableVehicleTypeIDColName).Eq(s.VehicleTypeID))
	}

	return sqSelect
}
