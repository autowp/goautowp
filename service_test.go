package goautowp

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTrafficMigrations(t *testing.T) {

	config := LoadConfig()

	err := applyTrafficMigrations(config.TrafficMigrations)
	if err != migrate.ErrNoChange {
		require.NoError(t, err)
	}
}

func TestAutowpMigrations(t *testing.T) {

	config := LoadConfig()

	err := applyAutowpMigrations(config.AutowpMigrations)
	if err != migrate.ErrNoChange {
		require.NoError(t, err)
	}
}
