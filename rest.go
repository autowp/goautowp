package goautowp

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/autowp/goautowp/util"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"

	sq "github.com/Masterminds/squirrel"
)

type perspective struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type perspectiveResult struct {
	Items []perspective `json:"items"`
}

type spec struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
	Childs    []spec `json:"childs"`
}

type specResult struct {
	Items []spec `json:"items"`
}

type VehicleType struct {
	ID     int           `json:"id"`
	Name   string        `json:"name"`
	Childs []VehicleType `json:"childs"`
}

type VehicleTypeResult struct {
	Items []VehicleType `json:"items"`
}

type BrandsIconsResult struct {
	Image string `json:"image"`
	CSS   string `json:"css"`
}

func (s *Service) getSpecs(parentID int) []spec {
	sqSelect := sq.Select("id, name, short_name").From("spec").OrderBy("name")

	if parentID != 0 {
		sqSelect = sqSelect.Where(sq.Eq{"parent_id": parentID})
	} else {
		sqSelect = sqSelect.Where(sq.Eq{"parent_id": nil})
	}

	rows, err := sqSelect.RunWith(s.db).Query()
	if err != nil {
		panic(err.Error())
	}

	var specs []spec
	for rows.Next() {
		var r spec
		err = rows.Scan(&r.ID, &r.Name, &r.ShortName)
		if err != nil {
			panic(err)
		}
		r.Childs = s.getSpecs(r.ID)
		specs = append(specs, r)
	}

	return specs
}

func (s *Service) getPerspectives() []perspective {
	sqSelect := sq.Select("id, name").From("perspectives").OrderBy("position")

	rows, err := sqSelect.RunWith(s.db).Query()
	if err != nil {
		panic(err.Error())
	}

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

func (s *Service) getVehicleTypesTree(parentID int) ([]VehicleType, error) {

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

func (s *Service) validateAuthorization(c *gin.Context) (string, error) {
	const bearerSchema = "Bearer"
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) <= len(bearerSchema) {
		return "", fmt.Errorf("authorization header is required")
	}
	tokenString := authHeader[len(bearerSchema)+1:]

	if len(tokenString) <= 0 {
		return "", fmt.Errorf("authorization header is invalid")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, isvalid := token.Method.(*jwt.SigningMethodHMAC); !isvalid {
			return nil, fmt.Errorf("invalid token alg %v", token.Header["alg"])

		}
		return []byte(s.config.OAuth.Secret), nil
	})

	if err != nil {
		return "", err
	}

	claims := token.Claims.(jwt.MapClaims)
	idStr := claims["sub"].(string)

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return "", err
	}

	sqSelect := sq.Select("role").From("users").Where(sq.Eq{"id": id})

	rows, err := sqSelect.RunWith(s.db).Query()
	if err != nil {
		panic(err.Error())
	}

	if !rows.Next() {
		return "", fmt.Errorf("user `%v` not found", id)
	}

	role := ""
	err = rows.Scan(&role)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("user `%v` not found", id)
	}

	if err != nil {
		return "", err
	}

	if role == "" {
		return "", fmt.Errorf("failed role detection for `%v`", id)
	}

	return role, nil
}

func (s *Service) setupRouter() {

	gin.SetMode(s.config.Rest.Mode)

	r := gin.New()
	r.Use(gin.Recovery())

	goapiGroup := r.Group("/go-api")
	{
		perspectives := s.getPerspectives()

		goapiGroup.GET("/perspective", func(c *gin.Context) {
			c.JSON(200, perspectiveResult{perspectives})
		})

		specs := s.getSpecs(0)

		goapiGroup.GET("/spec", func(c *gin.Context) {

			c.JSON(200, specResult{specs})
		})
	}

	apiGroup := r.Group("/api")
	{
		rand.Seed(time.Now().Unix())

		apiGroup.GET("/brands/icons", func(c *gin.Context) {

			if len(s.config.FileStorage.S3.Endpoints) <= 0 {
				c.String(http.StatusInternalServerError, "No enpoints provided")
				return
			}

			endpoint := s.config.FileStorage.S3.Endpoints[rand.Intn(len(s.config.FileStorage.S3.Endpoints))]

			parsedURL, err := url.Parse(endpoint)

			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}

			parsedURL.Path = "/" + url.PathEscape(s.config.FileStorage.Bucket) + "/brands.png"
			imageURL := parsedURL.String()

			parsedURL.Path = "/" + url.PathEscape(s.config.FileStorage.Bucket) + "/brands.css"
			cssURL := parsedURL.String()

			c.JSON(200, BrandsIconsResult{imageURL, cssURL})
		})

		apiGroup.GET("/vehicle-types", func(c *gin.Context) {

			role, err := s.validateAuthorization(c)

			if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
				c.Status(http.StatusForbidden)
				return
			}

			items, err := s.getVehicleTypesTree(0)

			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}

			c.JSON(200, VehicleTypeResult{items})
		})
	}

	s.router = r
}

// GetRouter GetRouter
func (s *Service) GetRouter() *gin.Engine {
	return s.router
}
