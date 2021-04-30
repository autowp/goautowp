package goautowp

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func createWhitelistService(t *testing.T) *Whitelist {
	config := LoadConfig()

	pool, err := pgxpool.Connect(context.Background(), config.TrafficDSN)
	require.NoError(t, err)

	s, err := NewWhitelist(pool)
	require.NoError(t, err)

	return s
}

func TestMatchAuto(t *testing.T) {

	s := createWhitelistService(t)

	// match, _ := s.MatchAuto(net.IPv4(178, 154, 255, 146)) // yandex
	// assert.True(t, match)

	match, _ := s.MatchAuto(net.IPv4(66, 249, 73, 139)) // google
	require.True(t, match)

	// match, _ = s.MatchAuto(net.IPv4(157, 55, 39, 127)) // msn
	// require.True(t, match)

	ip := net.IP{0x2a, 0x02, 0x06, 0xb8, 0xb0, 0x10, 0xa2, 0xfa, 0xfe, 0xaa, 0x00, 0x00, 0x8d, 0x08, 0x8e, 0xb7}
	match, _ = s.MatchAuto(ip) // yandex ipv6
	require.True(t, match)

	match, _ = s.MatchAuto(net.IPv4(127, 0, 0, 1)) // loopback
	require.False(t, match)

}

func TestContains(t *testing.T) {

	s := createWhitelistService(t)

	ip := net.IPv4(66, 249, 73, 139)

	err := s.Add(ip, "test")
	require.NoError(t, err)

	exists, err := s.Exists(ip)
	require.NoError(t, err)
	require.True(t, exists)

}
