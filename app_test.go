package goautowp

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTrafficMigrations(t *testing.T) {

	config := LoadConfig()

	err := applyMigrations(config.TrafficMigrations)
	if err != migrate.ErrNoChange {
		require.NoError(t, err)
	}
}

func TestAutowpMigrations(t *testing.T) {

	config := LoadConfig()

	err := applyMigrations(config.AutowpMigrations)
	if err != migrate.ErrNoChange {
		require.NoError(t, err)
	}
}
