package goautowp

import (
	"context"
	"fmt"
	"sync"

	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
)

// Catalogue service.
type Catalogue struct {
	db *goqu.Database
}

// NewCatalogue constructor.
func NewCatalogue(db *goqu.Database) (*Catalogue, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	return &Catalogue{
		db: db,
	}, nil
}

func (s *Catalogue) getVehicleTypesTree(ctx context.Context, parentID int32) ([]*VehicleType, error) {
	sqSelect := s.db.Select(schema.CarTypesTableIDCol, schema.CarTypesTableNameCol).
		From(schema.CarTypesTable).
		Order(schema.CarTypesTablePositionCol.Asc())

	if parentID != 0 {
		sqSelect = sqSelect.Where(schema.CarTypesTableParentIDCol.Eq(parentID))
	} else {
		sqSelect = sqSelect.Where(schema.CarTypesTableParentIDCol.IsNull())
	}

	rows, err := sqSelect.Executor().QueryContext(ctx)
	defer util.Close(rows)

	if err != nil {
		return nil, err
	}

	result := []*VehicleType{}

	for rows.Next() {
		var r VehicleType
		err = rows.Scan(&r.Id, &r.Name)

		if err != nil {
			return nil, err
		}

		r.Childs, err = s.getVehicleTypesTree(ctx, r.Id)
		if err != nil {
			return nil, err
		}

		result = append(result, &r)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Catalogue) getSpecs(ctx context.Context, parentID int32) ([]*Spec, error) {
	sqSelect := s.db.Select("id", "name", "short_name").From(schema.TableSpec).Order(goqu.C("name").Asc())

	if parentID != 0 {
		sqSelect = sqSelect.Where(goqu.C("parent_id").Eq(parentID))
	} else {
		sqSelect = sqSelect.Where(goqu.C("parent_id").IsNull())
	}

	rows, err := sqSelect.Executor().QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	var specs []*Spec

	for rows.Next() {
		var r Spec

		err = rows.Scan(&r.Id, &r.Name, &r.ShortName)
		if err != nil {
			return nil, err
		}

		childs, err := s.getSpecs(ctx, r.Id)
		if err != nil {
			return nil, err
		}

		r.Childs = childs
		specs = append(specs, &r)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return specs, nil
}

func (s *Catalogue) getPerspectiveGroups(ctx context.Context, pageID int32) ([]*PerspectiveGroup, error) {
	sqSelect := s.db.Select(schema.PerspectivesGroupsTableIDCol, schema.PerspectivesGroupsTableNameCol).
		From(schema.PerspectivesGroupsTable).
		Where(schema.PerspectivesGroupsTablePageIDCol.Eq(pageID)).
		Order(schema.PerspectivesGroupsTablePositionCol.Asc())

	rows, err := sqSelect.Executor().QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	var wg sync.WaitGroup

	var perspectiveGroups []*PerspectiveGroup

	for rows.Next() {
		var r PerspectiveGroup
		err = rows.Scan(&r.Id, &r.Name)

		if err != nil {
			return nil, err
		}

		wg.Add(1)

		go func() {
			perspectives, err := s.getPerspectives(ctx, &r.Id)
			if err != nil {
				return
			}

			r.Perspectives = perspectives

			wg.Done()
		}()

		perspectiveGroups = append(perspectiveGroups, &r)
	}

	wg.Wait()

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return perspectiveGroups, nil
}

func (s *Catalogue) getPerspectivePages(ctx context.Context) ([]*PerspectivePage, error) {
	sqSelect := s.db.Select(schema.PerspectivesPagesTableIDCol, schema.PerspectivesPagesTableNameCol).
		From(schema.PerspectivesPagesTable).
		Order(schema.PerspectivesPagesTableIDCol.Asc())

	rows, err := sqSelect.Executor().QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	var wg sync.WaitGroup

	var perspectivePages []*PerspectivePage

	for rows.Next() {
		var r PerspectivePage

		err = rows.Scan(&r.Id, &r.Name)
		if err != nil {
			return nil, err
		}

		wg.Add(1)

		go func() {
			groups, err := s.getPerspectiveGroups(ctx, r.Id)
			if err != nil {
				return
			}

			r.Groups = groups

			wg.Done()
		}()

		perspectivePages = append(perspectivePages, &r)
	}

	wg.Wait()

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return perspectivePages, nil
}

func (s *Catalogue) getPerspectives(ctx context.Context, groupID *int32) ([]*Perspective, error) {
	sqSelect := s.db.Select(schema.PerspectivesTableIDCol, schema.PerspectivesTableNameCol).
		From(schema.PerspectivesTable)

	if groupID != nil {
		sqSelect = sqSelect.
			Join(
				schema.PerspectivesGroupsPerspectivesTable,
				goqu.On(schema.PerspectivesTableIDCol.Eq(schema.PerspectivesGroupsPerspectivesTablePerspectiveIDCol)),
			).
			Where(schema.PerspectivesGroupsPerspectivesTableGroupIDCol.Eq(*groupID)).
			Order(schema.PerspectivesGroupsPerspectivesTablePositionCol.Asc())
	} else {
		sqSelect = sqSelect.Order(schema.PerspectivesTablePositionCol.Asc())
	}

	rows, err := sqSelect.Executor().QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer util.Close(rows)

	var perspectives []*Perspective

	for rows.Next() {
		var r Perspective

		err = rows.Scan(&r.Id, &r.Name)
		if err != nil {
			return nil, err
		}

		perspectives = append(perspectives, &r)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return perspectives, nil
}

func (s *Catalogue) getBrandVehicleTypes(ctx context.Context, brandID int32) ([]*BrandVehicleType, error) {
	itemParentCacheTable := goqu.T(schema.TableItemParentCache)
	sqSelect := s.db.
		Select(schema.CarTypesTableIDCol, schema.CarTypesTableNameCol, schema.CarTypesTableCatnameCol,
			goqu.COUNT(goqu.DISTINCT(schema.ItemTableIDCol))).
		From(schema.CarTypesTable).
		Join(
			goqu.T(schema.TableVehicleVehicleType),
			goqu.On(schema.CarTypesTableIDCol.Eq(goqu.T(schema.TableVehicleVehicleType).Col("vehicle_type_id"))),
		).
		Join(schema.ItemTable, goqu.On(goqu.T(schema.TableVehicleVehicleType).Col("vehicle_id").Eq(schema.ItemTableIDCol))).
		Join(itemParentCacheTable, goqu.On(schema.ItemTableIDCol.Eq(itemParentCacheTable.Col("item_id")))).
		Where(
			itemParentCacheTable.Col("parent_id").Eq(brandID),
			goqu.Or(schema.ItemTableBeginYearCol, schema.ItemTableBeginModelYearCol),
			schema.ItemTableIsGroupCol.IsFalse(),
		).
		GroupBy(schema.CarTypesTableIDCol).
		Order(schema.CarTypesTablePositionCol.Asc())

	rows, err := sqSelect.Executor().QueryContext(ctx)
	defer util.Close(rows)

	if err != nil {
		return nil, err
	}

	result := []*BrandVehicleType{}

	for rows.Next() {
		var r BrandVehicleType

		err = rows.Scan(&r.Id, &r.Name, &r.Catname, &r.ItemsCount)
		if err != nil {
			return nil, err
		}

		result = append(result, &r)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
