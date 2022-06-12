package traffic

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"
)

func createMonitoringService(t *testing.T) *Monitoring {
	t.Helper()

	cfg := config.LoadConfig("..")

	pool, err := pgxpool.Connect(context.Background(), cfg.TrafficDSN)
	require.NoError(t, err)

	s, err := NewMonitoring(pool)
	require.NoError(t, err)

	return s
}

func TestMonitoringAdd(t *testing.T) {
	t.Parallel()

	s := createMonitoringService(t)

	err := s.Add(net.IPv4(192, 168, 0, 1), time.Now())
	require.NoError(t, err)

	err = s.Add(net.IPv6loopback, time.Now())
	require.NoError(t, err)
}

func TestMonitoringGC(t *testing.T) {
	t.Parallel()

	s := createMonitoringService(t)

	err := s.Clear()
	require.NoError(t, err)

	err = s.Add(net.IPv4(192, 168, 0, 77), time.Now())
	require.NoError(t, err)

	affected, err := s.GC()
	require.NoError(t, err)
	require.Zero(t, affected)

	_, err = s.ListOfTop(10)
	require.NoError(t, err)
}
