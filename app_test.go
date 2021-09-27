package goautowp

import (
	"github.com/autowp/goautowp/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTrafficMigrations(t *testing.T) {

	cfg := config.LoadConfig(".")

	err := applyMigrations(cfg.TrafficMigrations)
	if err != migrate.ErrNoChange {
		require.NoError(t, err)
	}
}

func TestAutowpMigrations(t *testing.T) {

	cfg := config.LoadConfig(".")

	err := applyMigrations(cfg.AutowpMigrations)
	if err != migrate.ErrNoChange {
		require.NoError(t, err)
	}
}
