package goautowp

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/autowp/goautowp/util"
	"github.com/casbin/casbin"
	"github.com/gin-gonic/gin"
)

// Catalogue service
type Catalogue struct {
	db          *sql.DB
	enforcer    *casbin.Enforcer
	oauthConfig OAuthConfig
}

// VehicleType VehicleType
type VehicleType struct {
	ID     int           `json:"id"`
	Name   string        `json:"name"`
	Childs []VehicleType `json:"childs"`
}

// VehicleTypeResult VehicleTypeResult
type VehicleTypeResult struct {
	Items []VehicleType `json:"items"`
}

// NewCatalogue constructor
func NewCatalogue(db *sql.DB, enforcer *casbin.Enforcer, oauthConfig OAuthConfig) (*Catalogue, error) {

	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	rand.Seed(time.Now().Unix())

	return &Catalogue{
		db:          db,
		enforcer:    enforcer,
		oauthConfig: oauthConfig,
	}, nil
}

func (s *Catalogue) getVehicleTypesTree(parentID int) ([]VehicleType, error) {

	sqSelect := sq.Select("id, name").From("car_types").OrderBy("position")

	if parentID != 0 {
		sqSelect = sqSelect.Where(sq.Eq{"parent_id": parentID})
	} else {
		sqSelect = sqSelect.Where("parent_id is null")
	}

	rows, err := sqSelect.RunWith(s.db).Query()
	defer util.Close(rows)
	if err != nil {
		return nil, err
	}

	result := []VehicleType{}
	for rows.Next() {
		var r VehicleType
		err = rows.Scan(&r.ID, &r.Name)
		if err != nil {
			return nil, err
		}
		r.Childs, err = s.getVehicleTypesTree(r.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}

	return result, nil
}

func (s *Catalogue) getSpecs(parentID int32) ([]*Spec, error) {
	sqSelect := sq.Select("id, name, short_name").From("spec").OrderBy("name")

	if parentID != 0 {
		sqSelect = sqSelect.Where(sq.Eq{"parent_id": parentID})
	} else {
		sqSelect = sqSelect.Where(sq.Eq{"parent_id": nil})
	}

	rows, err := sqSelect.RunWith(s.db).Query()
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
		childs, err := s.getSpecs(r.Id)
		if err != nil {
			return nil, err
		}
		r.Childs = childs
		specs = append(specs, &r)
	}

	return specs, nil
}

func (s *Catalogue) getPerspectiveGroups(pageID int32) ([]*PerspectiveGroup, error) {
	sqSelect := sq.Select("id, name").From("perspectives_groups").Where(sq.Eq{"page_id": pageID}).OrderBy("position")

	rows, err := sqSelect.RunWith(s.db).Query()
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
			perspectives, err := s.getPerspectives(&r.Id)
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

func (s *Catalogue) getPerspectivePages() ([]*PerspectivePage, error) {
	sqSelect := sq.Select("id, name").From("perspectives_pages").OrderBy("id")

	rows, err := sqSelect.RunWith(s.db).Query()
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
			groups, err := s.getPerspectiveGroups(r.Id)
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

func (s *Catalogue) getPerspectives(groupID *int32) ([]*Perspective, error) {
	sqSelect := sq.Select("perspectives.id, perspectives.name").From("perspectives")

	if groupID != nil {
		sqSelect = sqSelect.
			Join("perspectives_groups_perspectives ON perspectives.id = perspectives_groups_perspectives.perspective_id").
			Where(sq.Eq{"perspectives_groups_perspectives.group_id": *groupID}).
			OrderBy("perspectives_groups_perspectives.position")
	} else {
		sqSelect = sqSelect.OrderBy("perspectives.position")
	}

	rows, err := sqSelect.RunWith(s.db).Query()
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

// Routes adds routes
func (s *Catalogue) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/vehicle-types", func(c *gin.Context) {

		_, role, err := validateAuthorization(c, s.db, s.oauthConfig)
		if err != nil {
			c.String(http.StatusForbidden, err.Error())
			return
		}

		if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
			c.Status(http.StatusForbidden)
			return
		}

		items, err := s.getVehicleTypesTree(0)

		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, VehicleTypeResult{items})
	})
}
