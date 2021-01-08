package goautowp

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
	"time"
)

func createBanService(t *testing.T) *Ban {
	config := LoadConfig()

	pool, err := pgxpool.Connect(context.Background(), config.TrafficDSN)
	require.NoError(t, err)

	s, err := NewBan(pool)
	require.NoError(t, err)

	return s
}

func TestAddRemove(t *testing.T) {

	s := createBanService(t)

	ip := net.IPv4(66, 249, 73, 139)

	err := s.Add(ip, time.Hour, 1, "Test")
	require.NoError(t, err)

	exists, err := s.Exists(ip)
	require.NoError(t, err)
	require.True(t, exists)

	err = s.Remove(ip)
	require.NoError(t, err)

	exists, err = s.Exists(ip)
	require.NoError(t, err)
	require.False(t, exists)
}
