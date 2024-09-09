package goautowp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/require"
)

func TestPostgresMigrations(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	err := applyMigrations(cfg.PostgresMigrations)
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

func TestServe(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")
	cfg.PublicRest.Listen = ":9090"
	app := NewApplication(cfg)
	ctx := context.Background()

	done := make(chan bool)
	go func() {
		time.Sleep(5 * time.Second)
		close(done)
	}()

	err := app.Serve(ctx, ServeOptions{
		DuplicateFinderAMQP: true,
		MonitoringAMQP:      true,
		GRPC:                true,
		Public:              true,
		Private:             true,
		Autoban:             true,
	}, done)
	require.NoError(t, err)
}

func TestImageStorageListBrokenImages(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")
	app := NewApplication(cfg)
	ctx := context.Background()

	err := app.ImageStorageListBrokenImages(ctx, "picture", "")
	require.NoError(t, err)
}

func TestImageStorageListUnlinkedObjects(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")
	app := NewApplication(cfg)
	ctx := context.Background()

	err := app.ImageStorageListUnlinkedObjects(ctx, "format", true, "")
	require.NoError(t, err)
}

func TestGenerateIndexCache(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")
	app := NewApplication(cfg)
	ctx := context.Background()

	err := app.GenerateIndexCache(ctx)
	require.NoError(t, err)
}
