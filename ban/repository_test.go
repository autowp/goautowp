package ban

import (
	"database/sql"
	"net"
	"testing"
	"time"

	"github.com/doug-martin/goqu/v9"

	"github.com/autowp/goautowp/config"
	"github.com/stretchr/testify/require"
)

func createBanService(t *testing.T) *Repository {
	t.Helper()

	cfg := config.LoadConfig("..")

	db, err := sql.Open("postgres", cfg.PostgresDSN)
	require.NoError(t, err)

	goquDB := goqu.New("postgres", db)

	s, err := NewRepository(goquDB)
	require.NoError(t, err)

	return s
}

func TestAddRemove(t *testing.T) {
	t.Parallel()

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
