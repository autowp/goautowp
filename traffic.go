package goautowp

import (
	"database/sql"
	"fmt"
	"github.com/casbin/casbin"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"net"
	"net/http"
	"time"
)

const banByUserID = 9

// Traffic Traffic
type Traffic struct {
	Monitoring    *Monitoring
	Whitelist     *Whitelist
	Ban           *BanRepository
	autowpDB      *sql.DB
	enforcer      *casbin.Enforcer
	userExtractor *UserExtractor
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

// APITrafficBlacklistPostRequestBody APITrafficBlacklistPostRequestBody
type APITrafficBlacklistPostRequestBody struct {
	IP     net.IP `json:"ip"`
	Period int    `json:"period"`
	Reason string `json:"reason"`
}

type APITrafficWhitelistPostRequestBody struct {
	IP net.IP `json:"ip"`
}

// NewTraffic constructor
func NewTraffic(pool *pgxpool.Pool, autowpDB *sql.DB, enforcer *casbin.Enforcer, ban *BanRepository, userExtractor *UserExtractor) (*Traffic, error) {

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
		Monitoring:    monitoring,
		Whitelist:     whitelist,
		Ban:           ban,
		autowpDB:      autowpDB,
		enforcer:      enforcer,
		userExtractor: userExtractor,
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

		log.Printf("%s %v\n", profile.Reason, ip)

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
		log.Printf("Check IP %v\n", item.IP)
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
