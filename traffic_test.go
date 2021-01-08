package goautowp

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/casbin/casbin"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const adminAuthorizationHeader = "Bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJkZWZhdWx0Iiwic3ViIjoiMyJ9.tI-wPZ4BSqmpsZN0-SgWXaokzvB8T-uYWLR9OQurxPFNoPC56U3op1gSE5n2H02GYfDGig0Eyp6U0NbDpsQaAg"

func createTrafficService(t *testing.T) *Traffic {
	config := LoadConfig()

	pool, err := pgxpool.Connect(context.Background(), config.TrafficDSN)
	require.NoError(t, err)

	autowpDB, err := sql.Open("mysql", config.AutowpDSN)
	require.NoError(t, err)

	enforcer := casbin.NewEnforcer("model.conf", "policy.csv")

	s, err := NewTraffic(pool, autowpDB, enforcer, config.OAuth)
	require.NoError(t, err)

	return s
}

func TestAutoWhitelist(t *testing.T) {

	s := createTrafficService(t)

	ip := net.IPv4(66, 249, 73, 139) // google

	err := s.Ban.Add(ip, time.Hour, 9, "test")
	require.NoError(t, err)

	exists, err := s.Ban.Exists(ip)
	require.NoError(t, err)
	require.True(t, exists)

	err = s.Monitoring.Add(ip, time.Now())
	require.NoError(t, err)

	exists, err = s.Monitoring.ExistsIP(ip)
	require.NoError(t, err)
	require.True(t, exists)

	err = s.AutoWhitelist()
	require.NoError(t, err)

	exists, err = s.Ban.Exists(ip)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = s.Monitoring.ExistsIP(ip)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = s.Whitelist.Exists(ip)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestAutoBanByProfile(t *testing.T) {

	s := createTrafficService(t)

	profile := AutobanProfile{
		Limit:  3,
		Reason: "Test",
		Group:  []string{"hour", "tenminute", "minute"},
		Time:   time.Hour,
	}

	ip1 := net.IPv4(127, 0, 0, 1)
	ip2 := net.IPv4(127, 0, 0, 2)

	err := s.Monitoring.ClearIP(ip1)
	require.NoError(t, err)
	err = s.Monitoring.ClearIP(ip2)
	require.NoError(t, err)

	err = s.Ban.Remove(ip1)
	require.NoError(t, err)
	err = s.Ban.Remove(ip2)
	require.NoError(t, err)

	err = s.Monitoring.Add(ip1, time.Now())
	require.NoError(t, err)
	for i := 0; i < 4; i++ {
		err = s.Monitoring.Add(ip2, time.Now())
		require.NoError(t, err)
	}

	err = s.AutoBanByProfile(profile)
	require.NoError(t, err)

	exists, err := s.Ban.Exists(ip1)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = s.Ban.Exists(ip2)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestWhitelistedNotBanned(t *testing.T) {

	s := createTrafficService(t)

	profile := AutobanProfile{
		Limit:  3,
		Reason: "TestWhitelistedNotBanned",
		Group:  []string{"hour", "tenminute", "minute"},
		Time:   time.Hour,
	}

	ip := net.IPv4(178, 154, 244, 21)

	err := s.Whitelist.Add(ip, "TestWhitelistedNotBanned")
	require.NoError(t, err)

	for i := 0; i < 4; i++ {
		err = s.Monitoring.Add(ip, time.Now())
		require.NoError(t, err)
	}

	err = s.AutoWhitelistIP(ip)
	require.NoError(t, err)

	err = s.AutoBanByProfile(profile)
	require.NoError(t, err)

	exists, err := s.Ban.Exists(ip)
	require.NoError(t, err)
	require.False(t, exists)
}

func TestHttpBanPost(t *testing.T) {
	s := createTrafficService(t)

	err := s.Ban.Remove(net.IPv4(127, 0, 0, 1))
	require.NoError(t, err)

	r := gin.New()
	apiGroup := r.Group("/api")
	s.SetupPublicRouter(apiGroup)

	w := httptest.NewRecorder()
	b, err := json.Marshal(map[string]interface{}{
		"ip":     "127.0.0.1",
		"period": 3,
		"reason": "Test",
	})
	require.NoError(t, err)
	req, err := http.NewRequest("POST", "/api/traffic/blacklist", bytes.NewBuffer(b))
	require.NoError(t, err)
	req.Header.Add("Authorization", adminAuthorizationHeader)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	exists, err := s.Ban.Exists(net.IPv4(127, 0, 0, 1))
	require.NoError(t, err)
	require.True(t, exists)

	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/api/traffic/blacklist/127.0.0.1", nil)
	require.NoError(t, err)
	req.Header.Add("Authorization", adminAuthorizationHeader)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)

	exists, err = s.Ban.Exists(net.IPv4(127, 0, 0, 1))
	require.NoError(t, err)
	require.False(t, exists)
}

func TestTop(t *testing.T) {
	s := createTrafficService(t)

	r := gin.New()
	apiGroup := r.Group("/api")
	s.SetupPublicRouter(apiGroup)

	err := s.Ban.Clear()
	require.NoError(t, err)

	err = s.Monitoring.Clear()
	require.NoError(t, err)

	err = s.Monitoring.Add(net.IPv4(192, 168, 0, 1), time.Now())
	require.NoError(t, err)

	now := time.Now()
	for i := 0; i < 10; i++ {
		err = s.Monitoring.Add(net.IPv6loopback, now)
		require.NoError(t, err)
	}

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/traffic", nil)
	require.NoError(t, err)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	body, err := ioutil.ReadAll(w.Body)
	require.Equal(t, `{"items":[{"ip":"::1","count":10,"ban":null,"in_whitelist":false,"whois_url":"http://nic.ru/whois/?query=%3A%3A1"},{"ip":"192.168.0.1","count":1,"ban":null,"in_whitelist":false,"whois_url":"http://nic.ru/whois/?query=192.168.0.1"}]}`, string(body))
}
