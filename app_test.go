package goautowp

import (
	"errors"
	"github.com/autowp/goautowp/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTrafficMigrations(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	err := applyMigrations(cfg.TrafficMigrations)
	if !errors.Is(err, migrate.ErrNoChange) {
		require.NoError(t, err)
	}
}

func TestAutowpMigrations(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	err := applyMigrations(cfg.AutowpMigrations)
	if !errors.Is(err, migrate.ErrNoChange) {
		require.NoError(t, err)
	}
}
