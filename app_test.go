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

func TestListenDuplicateFinderAMQP(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	app := NewApplication(cfg)

	ctx := context.Background()

	done := make(chan bool)
	go func() {
		time.Sleep(5 * time.Second)
		done <- false
	}()

	err := app.ListenDuplicateFinderAMQP(ctx, done)
	require.NoError(t, err)
}

func TestListenMonitoringAMQP(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")
	app := NewApplication(cfg)
	ctx := context.Background()

	done := make(chan bool)
	go func() {
		time.Sleep(5 * time.Second)
		done <- false
	}()

	err := app.ListenMonitoringAMQP(ctx, done)
	require.NoError(t, err)
}

func TestServeGRPC(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")
	cfg.GRPC.Listen = ":9093"
	app := NewApplication(cfg)

	done := make(chan bool)
	go func() {
		time.Sleep(5 * time.Second)
		done <- false
	}()

	err := app.ServeGRPC(done)
	require.NoError(t, err)
}

func TestServePublic(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")
	cfg.PublicRest.Listen = ":9090"
	app := NewApplication(cfg)
	ctx := context.Background()

	done := make(chan bool)
	go func() {
		time.Sleep(5 * time.Second)
		done <- false
	}()

	err := app.ServePublic(ctx, done)
	require.NoError(t, err)
}

func TestServePrivate(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")
	cfg.PrivateRest.Listen = ":9091"
	app := NewApplication(cfg)
	ctx := context.Background()

	done := make(chan bool)
	go func() {
		time.Sleep(5 * time.Second)
		done <- false
	}()

	err := app.ServePrivate(ctx, done)
	require.NoError(t, err)
}

func TestImageStorageListBrokenImages(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")
	app := NewApplication(cfg)
	ctx := context.Background()

	err := app.ImageStorageListBrokenImages(ctx, "picture")
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
