package goautowp

import (
	"database/sql"
	"net/http"

	"github.com/casbin/casbin"
	"github.com/gin-gonic/gin"
)

// ACL service
type ACL struct {
	db          *sql.DB
	enforcer    *casbin.Enforcer
	oauthConfig OAuthConfig
}

// NewACL constructor
func NewACL(db *sql.DB, enforcer *casbin.Enforcer, oauthConfig OAuthConfig) *ACL {

	return &ACL{
		db:          db,
		enforcer:    enforcer,
		oauthConfig: oauthConfig,
	}
}

// Routes adds routes
func (s *ACL) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/acl/is-allowed", func(c *gin.Context) {

		_, role, err := validateAuthorization(c, s.db, s.oauthConfig)
		if err != nil {
			c.String(http.StatusForbidden, err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"result": s.enforcer.Enforce(role, c.Query("resource"), c.Query("privilege")),
		})
	})
}
