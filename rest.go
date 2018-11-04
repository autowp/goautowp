package goautowp

import "github.com/gin-gonic/gin"
import sq "github.com/Masterminds/squirrel"

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

	specs := []spec{}
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

	perspectives := []perspective{}
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

func (s *Service) setupRouter() {
	r := gin.New()
	r.Use(gin.Recovery())

	apiGroup := r.Group("/go-api")
	{
		perspectives := s.getPerspectives()

		apiGroup.GET("/perspective", func(c *gin.Context) {
			c.JSON(200, perspectiveResult{perspectives})
		})

		specs := s.getSpecs(0)

		apiGroup.GET("/spec", func(c *gin.Context) {

			c.JSON(200, specResult{specs})
		})
	}

	s.router = r
}

// GetRouter GetRouter
func (s *Service) GetRouter() *gin.Engine {
	return s.router
}
