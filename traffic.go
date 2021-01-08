package goautowp

import (
	"database/sql"
	"fmt"
	"github.com/autowp/goautowp/util"
	"github.com/casbin/casbin"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

const banByUserID = 9

// Traffic Traffic
type Traffic struct {
	Monitoring  *Monitoring
	Whitelist   *Whitelist
	Ban         *Ban
	autowpDB    *sql.DB
	enforcer    *casbin.Enforcer
	oauthConfig OAuthConfig
}

// AutobanProfile AutobanProfile
type AutobanProfile struct {
	Limit  int
	Reason string
	Group  []string
	Time   time.Duration
}

// AutobanProfiles AutobanProfiles
var AutobanProfiles = []AutobanProfile{
	{
		Limit:  10000,
		Reason: "daily limit",
		Group:  []string{},
		Time:   time.Hour * 10 * 24,
	},
	{
		Limit:  3600,
		Reason: "hourly limit",
		Group:  []string{"hour"},
		Time:   time.Hour * 5 * 24,
	},
	{
		Limit:  1200,
		Reason: "ten min limit",
		Group:  []string{"hour", "tenminute"},
		Time:   time.Hour * 24,
	},
	{
		Limit:  700,
		Reason: "min limit",
		Group:  []string{"hour", "tenminute", "minute"},
		Time:   time.Hour * 12,
	},
}

// TrafficBlacklistPostRequestBody TrafficBlacklistPostRequestBody
type TrafficBlacklistPostRequestBody struct {
	IP     net.IP `json:"ip"`
	Period int    `json:"period"`
	Reason string `json:"reason"`
}

type TrafficWhitelistPostRequestBody struct {
	IP net.IP `json:"ip"`
}

// TopItemBan TopItemBan
type TopItemBan struct {
	Until    time.Time `json:"up_to"`
	ByUserID int       `json:"by_user_id"`
	User     APIUser   `json:"user"`
	Reason   string    `json:"reason"`
}

// TopItem TopItem
type TopItem struct {
	IP          net.IP      `json:"ip"`
	Count       int         `json:"count"`
	Ban         *TopItemBan `json:"ban"`
	InWhitelist bool        `json:"in_whitelist"`
	WhoisUrl    string      `json:"whois_url"`
}

// NewTraffic constructor
func NewTraffic(pool *pgxpool.Pool, autowpDB *sql.DB, enforcer *casbin.Enforcer, oauthConfig OAuthConfig) (*Traffic, error) {

	ban, err := NewBan(pool)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	monitoring, err := NewMonitoring(pool)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	whitelist, err := NewWhitelist(pool)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	s := &Traffic{
		Monitoring:  monitoring,
		Whitelist:   whitelist,
		Ban:         ban,
		autowpDB:    autowpDB,
		enforcer:    enforcer,
		oauthConfig: oauthConfig,
	}

	return s, nil
}

func (s *Traffic) AutoBanByProfile(profile AutobanProfile) error {

	ips, err := s.Monitoring.ListByBanProfile(profile)
	if err != nil {
		return err
	}

	for _, ip := range ips {
		exists, err := s.Whitelist.Exists(ip)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		fmt.Printf("%s %v\n", profile.Reason, ip)

		if err := s.Ban.Add(ip, profile.Time, banByUserID, profile.Reason); err != nil {
			return err
		}
	}

	return nil
}

func (s *Traffic) AutoBan() error {
	for _, profile := range AutobanProfiles {
		if err := s.AutoBanByProfile(profile); err != nil {
			return err
		}
	}

	return nil
}

func (s *Traffic) AutoWhitelist() error {

	items, err := s.Monitoring.ListOfTop(1000)
	if err != nil {
		return err
	}

	for _, item := range items {
		fmt.Printf("Check IP %v\n", item.IP)
		if err := s.AutoWhitelistIP(item.IP); err != nil {
			return err
		}
	}

	return nil
}

func (s *Traffic) AutoWhitelistIP(ip net.IP) error {
	ipText := ip.String()

	fmt.Print(ipText + ": ")

	inWhitelist, err := s.Whitelist.Exists(ip)
	if err != nil {
		return err
	}

	match, desc := s.Whitelist.MatchAuto(ip)

	if !match {
		fmt.Println("")
		return nil
	}

	if inWhitelist {
		fmt.Println("whitelist, skip")
	} else {
		if err := s.Whitelist.Add(ip, desc); err != nil {
			return err
		}
	}

	if err := s.Ban.Remove(ip); err != nil {
		return err
	}

	if err := s.Monitoring.ClearIP(ip); err != nil {
		return err
	}

	fmt.Println(" whitelisted")

	return nil
}

func (s *Traffic) SetupPrivateRouter(r *gin.Engine) {
	r.GET("/ban/:ip", func(c *gin.Context) {
		ip := net.ParseIP(c.Param("ip"))
		if ip == nil {
			c.String(http.StatusBadRequest, "Invalid IP")
			return
		}

		ban, err := s.Ban.Get(ip)
		if err != nil {
			log.Println(err.Error())
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		if ban == nil {
			c.Status(http.StatusNotFound)
			return
		}

		c.JSON(http.StatusOK, ban)
	})
}

func (s *Traffic) SetupPublicRouter(apiGroup *gin.RouterGroup) {
	apiGroup.POST("/traffic/blacklist", func(c *gin.Context) {
		id, role, err := validateAuthorization(c, s.autowpDB, s.oauthConfig)
		if err != nil {
			log.Println(err.Error())
			c.String(http.StatusForbidden, err.Error())
			return
		}

		if res := s.enforcer.Enforce(role, "user", "ban"); !res {
			c.Status(http.StatusForbidden)
			return
		}

		request := TrafficBlacklistPostRequestBody{}
		err = c.BindJSON(&request)

		if err != nil {
			fmt.Println(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		duration := time.Hour * time.Duration(request.Period)

		err = s.Ban.Add(request.IP, duration, id, request.Reason)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.Status(http.StatusCreated)
	})

	apiGroup.DELETE("/traffic/blacklist/:ip", func(c *gin.Context) {
		_, role, err := validateAuthorization(c, s.autowpDB, s.oauthConfig)
		if err != nil {
			c.String(http.StatusForbidden, err.Error())
			return
		}

		if res := s.enforcer.Enforce(role, "user", "ban"); !res {
			c.Status(http.StatusForbidden)
			return
		}

		ip := net.ParseIP(c.Param("ip"))
		if ip == nil {
			c.String(http.StatusBadRequest, "Invalid IP")
			return
		}

		err = s.Ban.Remove(ip)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.Status(http.StatusNoContent)
	})

	apiGroup.GET("/traffic/whitelist", func(c *gin.Context) {
		_, role, err := validateAuthorization(c, s.autowpDB, s.oauthConfig)
		if err != nil {
			c.String(http.StatusForbidden, err.Error())
			return
		}

		if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
			c.Status(http.StatusForbidden)
			return
		}

		list, err := s.Whitelist.List()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items": list,
		})
	})

	apiGroup.POST("/traffic/whitelist", func(c *gin.Context) {
		_, role, err := validateAuthorization(c, s.autowpDB, s.oauthConfig)
		if err != nil {
			c.String(http.StatusForbidden, err.Error())
			return
		}

		if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
			c.Status(http.StatusForbidden)
			return
		}

		request := TrafficWhitelistPostRequestBody{}
		err = c.BindJSON(&request)

		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		err = s.Whitelist.Add(request.IP, "manual click")
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		err = s.Ban.Remove(request.IP)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.Status(http.StatusCreated)
	})

	apiGroup.DELETE("/traffic/whitelist/:ip", func(c *gin.Context) {
		_, role, err := validateAuthorization(c, s.autowpDB, s.oauthConfig)
		if err != nil {
			c.String(http.StatusForbidden, err.Error())
			return
		}

		if res := s.enforcer.Enforce(role, "global", "moderate"); !res {
			c.Status(http.StatusForbidden)
			return
		}

		ip := net.ParseIP(c.Param("ip"))
		if ip == nil {
			c.String(http.StatusBadRequest, "Invalid IP")
			return
		}

		err = s.Whitelist.Remove(ip)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.Status(http.StatusNoContent)
	})

	apiGroup.GET("/traffic", func(c *gin.Context) {
		items, err := s.Monitoring.ListOfTop(50)

		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		result := make([]TopItem, len(items))
		for idx, item := range items {

			ban, err := s.Ban.Get(item.IP)
			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}

			inWhitelist, err := s.Whitelist.Exists(item.IP)
			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}

			var user *DBUser
			var topItemBan *TopItemBan

			if ban != nil {
				user, err = s.getUser(ban.ByUserID)
				if err != nil {
					c.String(http.StatusInternalServerError, err.Error())
					return
				}

				topItemBan = &TopItemBan{
					Until:    ban.Until,
					ByUserID: ban.ByUserID,
					User:     ExtractUser(*user, s.enforcer),
					Reason:   ban.Reason,
				}
			}

			result[idx] = TopItem{
				IP:          item.IP,
				Count:       item.Count,
				Ban:         topItemBan,
				InWhitelist: inWhitelist,
				WhoisUrl:    fmt.Sprintf("http://nic.ru/whois/?query=%s", url.QueryEscape(item.IP.String())),
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"items": result,
		})
	})
}

func (s *Traffic) getUser(id int) (*DBUser, error) {
	rows, err := s.autowpDB.Query(`
		SELECT id, name, deleted, identity, last_online, role
		FROM users
		WHERE id = ?
	`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	defer util.Close(rows)

	if !rows.Next() {
		return nil, nil
	}

	var r DBUser
	err = rows.Scan(&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role)
	if err != nil {
		return nil, err
	}

	return &r, nil
}
