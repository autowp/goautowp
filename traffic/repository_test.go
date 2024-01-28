package traffic

import (
	"context"
	"database/sql"
	"net"
	"testing"
	"time"

	"github.com/autowp/goautowp/ban"
	"github.com/autowp/goautowp/config"
	"github.com/casbin/casbin"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/go-sql-driver/mysql" // enable mysql driver
	"github.com/stretchr/testify/require"
)

func createTrafficService(t *testing.T) *Traffic {
	t.Helper()

	cfg := config.LoadConfig("..")

	autowpDB, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", autowpDB)

	db, err := sql.Open("postgres", cfg.PostgresDSN)
	require.NoError(t, err)

	goquPostgresDB := goqu.New("postgres", db)

	banRepository, err := ban.NewRepository(goquPostgresDB)
	require.NoError(t, err)

	enforcer := casbin.NewEnforcer("../model.conf", "../policy.csv")

	traf, err := NewTraffic(goquPostgresDB, goquDB, enforcer, banRepository)
	require.NoError(t, err)

	return traf
}

func TestAutoWhitelist(t *testing.T) { //nolint:paralleltest
	s := createTrafficService(t)

	ctx := context.Background()

	ip := net.IPv4(66, 249, 73, 139) // google

	err := s.Ban.Add(ctx, ip, time.Hour, 9, "test")
	require.NoError(t, err)

	exists, err := s.Ban.Exists(ctx, ip)
	require.NoError(t, err)
	require.True(t, exists)

	err = s.Monitoring.Add(ip, time.Now())
	require.NoError(t, err)

	exists, err = s.Monitoring.ExistsIP(ip)
	require.NoError(t, err)
	require.True(t, exists)

	err = s.AutoWhitelist(ctx)
	require.NoError(t, err)

	exists, err = s.Ban.Exists(ctx, ip)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = s.Monitoring.ExistsIP(ip)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = s.Whitelist.Exists(ctx, ip)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestAutoBanByProfile(t *testing.T) { //nolint:paralleltest
	s := createTrafficService(t)

	ctx := context.Background()

	profile := AutobanProfile{
		Limit:  3,
		Reason: "Test",
		Group:  []string{"hour", "tenminute", "minute"},
		Time:   time.Hour,
	}

	ip1 := net.IPv4(127, 0, 0, 11)
	ip2 := net.IPv4(127, 0, 0, 12)

	err := s.Monitoring.ClearIP(ctx, ip1)
	require.NoError(t, err)
	err = s.Monitoring.ClearIP(ctx, ip2)
	require.NoError(t, err)

	err = s.Ban.Remove(ctx, ip1)
	require.NoError(t, err)
	err = s.Ban.Remove(ctx, ip2)
	require.NoError(t, err)

	err = s.Monitoring.Add(ip1, time.Now())
	require.NoError(t, err)

	for i := 0; i < 4; i++ {
		err = s.Monitoring.Add(ip2, time.Now())
		require.NoError(t, err)
	}

	err = s.AutoBanByProfile(ctx, profile)
	require.NoError(t, err)

	exists, err := s.Ban.Exists(ctx, ip1)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = s.Ban.Exists(ctx, ip2)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestWhitelistedNotBanned(t *testing.T) {
	t.Parallel()

	s := createTrafficService(t)

	ctx := context.Background()

	profile := AutobanProfile{
		Limit:  3,
		Reason: "TestWhitelistedNotBanned",
		Group:  []string{"hour", "tenminute", "minute"},
		Time:   time.Hour,
	}

	ip := net.IPv4(178, 154, 244, 21)

	err := s.Whitelist.Add(ctx, ip, "TestWhitelistedNotBanned")
	require.NoError(t, err)

	for i := 0; i < 4; i++ {
		err = s.Monitoring.Add(ip, time.Now())
		require.NoError(t, err)
	}

	err = s.AutoWhitelistIP(ctx, ip)
	require.NoError(t, err)

	err = s.AutoBanByProfile(ctx, profile)
	require.NoError(t, err)

	exists, err := s.Ban.Exists(ctx, ip)
	require.NoError(t, err)
	require.False(t, exists)
}

func TestTop(t *testing.T) {
	t.Parallel()

	s := createTrafficService(t)

	ctx := context.Background()

	err := s.Ban.Clear(ctx)
	require.NoError(t, err)

	err = s.Monitoring.Clear(ctx)
	require.NoError(t, err)

	err = s.Monitoring.Add(net.IPv4(192, 168, 0, 1), time.Now())
	require.NoError(t, err)

	now := time.Now()
	for i := 0; i < 10; i++ {
		err = s.Monitoring.Add(net.IPv6loopback, now)
		require.NoError(t, err)
	}
}
