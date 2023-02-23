package goautowp

import (
	"context"
	"fmt"
	"sync"

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
	sqSelect := s.db.Select("id", "name").From("car_types").Order(goqu.I("position").Asc())

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

	return result, nil
}

func (s *Catalogue) getSpecs(ctx context.Context, parentID int32) ([]*Spec, error) {
	sqSelect := s.db.Select("id", "name", "short_name").From("spec").Order(goqu.I("name").Asc())

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

	return specs, nil
}

func (s *Catalogue) getPerspectiveGroups(ctx context.Context, pageID int32) ([]*PerspectiveGroup, error) {
	sqSelect := s.db.Select("id", "name").
		From("perspectives_groups").
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

	return perspectiveGroups, nil
}

func (s *Catalogue) getPerspectivePages(ctx context.Context) ([]*PerspectivePage, error) {
	sqSelect := s.db.Select("id", "name").From("perspectives_pages").Order(goqu.I("id").Asc())

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

	return perspectivePages, nil
}

func (s *Catalogue) getPerspectives(ctx context.Context, groupID *int32) ([]*Perspective, error) {
	sqSelect := s.db.Select("perspectives.id", "perspectives.name").From("perspectives")

	if groupID != nil {
		sqSelect = sqSelect.
			Join(
				goqu.T("perspectives_groups_perspectives"),
				goqu.On(goqu.I("perspectives.id").Eq("perspectives_groups_perspectives.perspective_id")),
			).
			Where(goqu.Ex{"perspectives_groups_perspectives.group_id": *groupID}).
			Order(goqu.I("perspectives_groups_perspectives.position").Asc())
	} else {
		sqSelect = sqSelect.Order(goqu.I("perspectives.position").Asc())
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

	return perspectives, nil
}

func (s *Catalogue) getBrandVehicleTypes(ctx context.Context, brandID int32) ([]*BrandVehicleType, error) {
	sqSelect := s.db.
		Select("car_types.id", "car_types.name", "car_types.catname", goqu.L("COUNT(DISTINCT item.id)")).
		From("car_types").
		Join(goqu.T("vehicle_vehicle_type"), goqu.On(goqu.I("car_types.id").Eq("vehicle_vehicle_type.vehicle_type_id"))).
		Join(goqu.T("item"), goqu.On(goqu.I("vehicle_vehicle_type.vehicle_id").Eq("item.id"))).
		Join(goqu.T("item_parent_cache"), goqu.On(goqu.I("item.id").Eq("item_parent_cache.item_id"))).
		Where(
			goqu.I("item_parent_cache.parent_id").Eq(brandID),
			goqu.L("(item.begin_year or item.begin_model_year)"),
			goqu.L("not item.is_group"),
		).
		GroupBy("car_types.id").
		Order(goqu.I("car_types.position").Asc())

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

	return result, nil
}
