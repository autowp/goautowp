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
	sqSelect := s.db.Select("id", "name").From(schema.TableCarTypes).Order(goqu.I("position").Asc())

	if parentID != 0 {
		sqSelect = sqSelect.Where(goqu.I("parent_id").Eq(parentID))
	} else {
		sqSelect = sqSelect.Where(goqu.I("parent_id").IsNull())
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
	sqSelect := s.db.Select("id", "name", "short_name").From(schema.TableSpec).Order(goqu.I("name").Asc())

	if parentID != 0 {
		sqSelect = sqSelect.Where(goqu.I("parent_id").Eq(parentID))
	} else {
		sqSelect = sqSelect.Where(goqu.I("parent_id").IsNull())
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
	sqSelect := s.db.Select("id", "name").
		From(schema.TablePerspectivesGroups).
		Where(goqu.Ex{"page_id": pageID}).Order(goqu.I("position").Asc())

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
	sqSelect := s.db.Select("id", "name").From(schema.TablePerspectivesPages).Order(goqu.I("id").Asc())

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
	perspectivesTable := goqu.T(schema.TablePerspectives)
	perspectivesGroupPerspectivesTable := goqu.T(schema.TablePerspectivesGroupsPerspectives)

	sqSelect := s.db.Select(perspectivesTable.Col("id"), perspectivesTable.Col("name")).
		From(schema.TablePerspectives)

	if groupID != nil {
		sqSelect = sqSelect.
			Join(
				perspectivesGroupPerspectivesTable,
				goqu.On(perspectivesTable.Col("id").Eq(perspectivesGroupPerspectivesTable.Col("perspective_id"))),
			).
			Where(perspectivesGroupPerspectivesTable.Col("group_id").Eq(*groupID)).
			Order(perspectivesGroupPerspectivesTable.Col("position").Asc())
	} else {
		sqSelect = sqSelect.Order(perspectivesTable.Col("position").Asc())
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
	carTypeTable := goqu.T(schema.TableCarTypes)
	itemTable := goqu.T(schema.TableItem)
	itemParentCacheTable := goqu.T(schema.TableItemParentCache)
	sqSelect := s.db.
		Select(carTypeTable.Col("id"), carTypeTable.Col("name"), carTypeTable.Col("catname"),
			goqu.COUNT(goqu.DISTINCT(itemTable.Col("id")))).
		From(carTypeTable).
		Join(
			goqu.T(schema.TableVehicleVehicleType),
			goqu.On(carTypeTable.Col("id").Eq(goqu.T(schema.TableVehicleVehicleType).Col("vehicle_type_id"))),
		).
		Join(itemTable, goqu.On(goqu.T(schema.TableVehicleVehicleType).Col("vehicle_id").Eq(itemTable.Col("id")))).
		Join(itemParentCacheTable, goqu.On(itemTable.Col("id").Eq(itemParentCacheTable.Col("item_id")))).
		Where(
			itemParentCacheTable.Col("parent_id").Eq(brandID),
			goqu.L("("+schema.TableItem+".begin_year or "+schema.TableItem+".begin_model_year)"),
			goqu.L("not "+schema.TableItem+".is_group"),
		).
		GroupBy(carTypeTable.Col("id")).
		Order(carTypeTable.Col("position").Asc())

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
