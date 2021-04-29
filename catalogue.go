package goautowp

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/autowp/goautowp/util"
	"github.com/casbin/casbin"
	"github.com/gin-gonic/gin"
)

// Catalogue service
type Catalogue struct {
	db                *sql.DB
	enforcer          *casbin.Enforcer
	fileStorageConfig FileStorageConfig
	oauthConfig       OAuthConfig
}

type perspective struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type perspectiveResult struct {
	Items []perspective `json:"items"`
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

// BrandsIconsResult BrandsIconsResult
type BrandsIconsResult struct {
	Image string `json:"image"`
	CSS   string `json:"css"`
}

// NewCatalogue constructor
func NewCatalogue(db *sql.DB, enforcer *casbin.Enforcer, fileStorageConfig FileStorageConfig, oauthConfig OAuthConfig) (*Catalogue, error) {

	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	rand.Seed(time.Now().Unix())

	return &Catalogue{
		db:                db,
		enforcer:          enforcer,
		fileStorageConfig: fileStorageConfig,
		oauthConfig:       oauthConfig,
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

func (s *Catalogue) getSpecs(parentID int64) ([]*Spec, error) {
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

func (s *Catalogue) getPerspectives() []perspective {
	sqSelect := sq.Select("id, name").From("perspectives").OrderBy("position")

	rows, err := sqSelect.RunWith(s.db).Query()
	if err != nil {
		panic(err.Error())
	}
	defer util.Close(rows)

	var perspectives []perspective
	for rows.Next() {
		var r perspective
		err = rows.Scan(&r.ID, &r.Name)
		if err != nil {
			panic(err)
		}
		perspectives = append(perspectives, r)
	}

	return perspectives
}

// Routes adds routes
func (s *Catalogue) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/perspective", func(c *gin.Context) {
		perspectives := s.getPerspectives()
		c.JSON(http.StatusOK, perspectiveResult{perspectives})
	})

	apiGroup.GET("/brands/icons", func(c *gin.Context) {

		if len(s.fileStorageConfig.S3.Endpoints) <= 0 {
			c.String(http.StatusInternalServerError, "No endpoints provided")
			return
		}

		endpoint := s.fileStorageConfig.S3.Endpoints[rand.Intn(len(s.fileStorageConfig.S3.Endpoints))]

		parsedURL, err := url.Parse(endpoint)

		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		parsedURL.Path = "/" + url.PathEscape(s.fileStorageConfig.Bucket) + "/brands.png"
		imageURL := parsedURL.String()

		parsedURL.Path = "/" + url.PathEscape(s.fileStorageConfig.Bucket) + "/brands.css"
		cssURL := parsedURL.String()

		c.JSON(http.StatusOK, BrandsIconsResult{imageURL, cssURL})
	})

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
