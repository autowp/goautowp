package goautowp

import (
	"database/sql"
	"github.com/casbin/casbin"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"strings"
	"time"
)

// APIIP APIIP
type APIIP struct {
	Address   string          `json:"address"`
	Hostname  *string         `json:"hostname"`
	Blacklist *APIIPBlacklist `json:"blacklist"`
	Rights    *APIIPRights    `json:"rights"`
}

// APIIPBlacklist APIIPBlacklist
type APIIPBlacklist struct {
	Until    time.Time `json:"up_to"`
	ByUserID int       `json:"by_user_id"`
	User     *APIUser  `json:"user"`
	Reason   string    `json:"reason"`
}

// APIIPRights APIIPRights
type APIIPRights struct {
	AddToBlacklist      bool `json:"add_to_blacklist"`
	RemoveFromBlacklist bool `json:"remove_from_blacklist"`
}

// IPController IPController
type IPController struct {
	autowpDB      *sql.DB
	enforcer      *casbin.Enforcer
	oauthConfig   OAuthConfig
	banRepository *BanRepository
	ipExtractor   *IPExtractor
}

// NewIPController constructor
func NewIPController(autowpDB *sql.DB, enforcer *casbin.Enforcer, ipExtractor *IPExtractor, banRepository *BanRepository, oauthConfig OAuthConfig) (*IPController, error) {
	return &IPController{
		autowpDB:      autowpDB,
		enforcer:      enforcer,
		oauthConfig:   oauthConfig,
		banRepository: banRepository,
		ipExtractor:   ipExtractor,
	}, nil
}

func (s *IPController) SetupRouter(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/ip/:ip", func(c *gin.Context) {

		_, role, _ := validateAuthorization(c, s.autowpDB, s.oauthConfig)

		ip := net.ParseIP(c.Param("ip"))
		if ip == nil {
			c.String(http.StatusBadRequest, "Invalid IP")
			return
		}

		fields := strings.Split(c.Query("fields"), ",")
		m := make(map[string]bool)
		for _, e := range fields {
			m[e] = true
		}

		result, err := s.ipExtractor.Extract(ip, m, role)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, result)
	})
}
