package traffic

import (
	"context"
	"database/sql"
	"net"
	"testing"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // enable postgres migrations
	"github.com/stretchr/testify/require"
)

func createMonitoringService(t *testing.T) *Monitoring {
	t.Helper()

	cfg := config.LoadConfig("..")

	db, err := sql.Open("postgres", cfg.PostgresDSN)
	require.NoError(t, err)

	goquDB := goqu.New("postgres", db)

	s, err := NewMonitoring(goquDB)
	require.NoError(t, err)

	return s
}

func TestMonitoringAdd(t *testing.T) {
	t.Parallel()

	s := createMonitoringService(t)

	ctx := context.Background()

	err := s.Add(ctx, net.IPv4(192, 168, 0, 1), time.Now())
	require.NoError(t, err)

	err = s.Add(ctx, net.IPv6loopback, time.Now())
	require.NoError(t, err)
}

func TestMonitoringGC(t *testing.T) {
	t.Parallel()

	s := createMonitoringService(t)

	ctx := context.Background()

	err := s.Clear(ctx)
	require.NoError(t, err)

	err = s.Add(ctx, net.IPv4(192, 168, 0, 77), time.Now())
	require.NoError(t, err)

	affected, err := s.GC(ctx)
	require.NoError(t, err)
	require.Zero(t, affected)

	_, err = s.ListOfTop(ctx, 10)
	require.NoError(t, err)
}

func TestListByBanProfile(t *testing.T) { //nolint:paralleltest
	s := createMonitoringService(t)

	ctx := context.Background()

	err := s.Clear(ctx)
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		err = s.Add(ctx, net.IPv4(192, 168, 0, 77), time.Now())
		require.NoError(t, err)
	}

	ips, err := s.ListByBanProfile(ctx, AutobanProfile{
		Limit:  1,
		Reason: "test",
		Group:  []interface{}{},
		Time:   dailyLimitDuration,
	})
	require.NoError(t, err)
	require.NotEmpty(t, ips)

	require.Equal(t, ips[0].String(), "192.168.0.77")
}
