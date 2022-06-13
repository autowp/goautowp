package traffic

import (
	"context"
	"database/sql"
	"net"
	"testing"
	"time"

	"github.com/autowp/goautowp/ban"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/users"
	"github.com/casbin/casbin"
	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"
)

func createTrafficService(t *testing.T) *Traffic {
	t.Helper()

	cfg := config.LoadConfig("..")

	db, err := pgxpool.Connect(context.Background(), cfg.TrafficDSN)
	require.NoError(t, err)

	autowpDB, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", autowpDB)

	banRepository, err := ban.NewRepository(db)
	require.NoError(t, err)

	enforcer := casbin.NewEnforcer("../model.conf", "../policy.csv")

	imageStorage, err := storage.NewStorage(goquDB, cfg.ImageStorage)
	require.NoError(t, err)

	userExtractor := users.NewUserExtractor(enforcer, imageStorage)

	traf, err := NewTraffic(db, goquDB, enforcer, banRepository, userExtractor)
	require.NoError(t, err)

	return traf
}

func TestAutoWhitelist(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	s := createTrafficService(t)

	profile := AutobanProfile{
		Limit:  3,
		Reason: "Test",
		Group:  []string{"hour", "tenminute", "minute"},
		Time:   time.Hour,
	}

	ip1 := net.IPv4(127, 0, 0, 11)
	ip2 := net.IPv4(127, 0, 0, 12)

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
	t.Parallel()

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

func TestTop(t *testing.T) {
	t.Parallel()

	s := createTrafficService(t)

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
}
