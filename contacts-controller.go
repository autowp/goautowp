package goautowp

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
)

// ContactsController Main Object
type ContactsController struct {
	repository     *ContactsRepository
	autowpDB       *sql.DB
	oauthConfig    OAuthConfig
	userRepository *UserRepository
	userExtractor  *UserExtractor
}

// NewBanController constructor
func NewContactsController(repository *ContactsRepository, userRepository *UserRepository, userExtractor *UserExtractor, autowpDB *sql.DB, oauthConfig OAuthConfig) (*ContactsController, error) {

	if repository == nil {
		return nil, fmt.Errorf("ContactsRepository is nil")
	}

	s := &ContactsController{
		autowpDB:       autowpDB,
		repository:     repository,
		oauthConfig:    oauthConfig,
		userRepository: userRepository,
		userExtractor:  userExtractor,
	}

	return s, nil
}

func (s *ContactsController) SetupRouter(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/contacts", func(c *gin.Context) {
		id, _, err := validateAuthorization(c, s.autowpDB, s.oauthConfig)
		if err != nil {
			c.String(http.StatusForbidden, err.Error())
			return
		}

		if id <= 0 {
			c.Status(http.StatusForbidden)
			return
		}

		fields := strings.Split(c.Query("fields"), ",")
		m := make(map[string]bool)
		for _, e := range fields {
			m[e] = true
		}

		userRows, err := s.userRepository.GetUsers(GetUsersOptions{
			InContacts: id,
			Order:      []string{"users.deleted", "users.name"},
			Fields:     m,
		})
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		items := make([]*APIUser, len(userRows))
		for idx, userRow := range userRows {
			items[idx], err = s.userExtractor.Extract(&userRow, m)
			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"items": items,
		})
	})

	apiGroup.GET("/contacts/:id", func(c *gin.Context) {
		id, _, err := validateAuthorization(c, s.autowpDB, s.oauthConfig)
		if err != nil {
			c.String(http.StatusForbidden, err.Error())
			return
		}

		contactID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		if contactID == id {
			c.Status(http.StatusNotFound)
			return
		}

		exists, err := s.repository.isExists(id, contactID)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		if !exists {
			c.Status(http.StatusNotFound)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"contact_user_id": contactID,
		})
	})

	apiGroup.PUT("/contacts/:id", func(c *gin.Context) {
		id, _, err := validateAuthorization(c, s.autowpDB, s.oauthConfig)
		if err != nil {
			c.String(http.StatusForbidden, err.Error())
			return
		}

		contactID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		if contactID == id {
			c.Status(http.StatusNotFound)
			return
		}

		deleted := false
		user, err := s.userRepository.GetUser(GetUsersOptions{ID: contactID, Deleted: &deleted})
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		if user == nil {
			c.Status(http.StatusNotFound)
			return
		}

		err = s.repository.create(id, contactID)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status": true,
		})
	})

	apiGroup.DELETE("/contacts/:id", func(c *gin.Context) {
		id, _, err := validateAuthorization(c, s.autowpDB, s.oauthConfig)
		if err != nil {
			c.String(http.StatusForbidden, err.Error())
			return
		}

		contactID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		err = s.repository.delete(id, contactID)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.Status(http.StatusNoContent)
	})
}
