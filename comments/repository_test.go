package comments

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Nerzal/gocloak/v11"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/telegram"
	"github.com/autowp/goautowp/users"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres" // enable postgres dialect
	_ "github.com/lib/pq"                               // enable postgres driver
	"github.com/stretchr/testify/require"
)

func createRepository(t *testing.T) *Repository {
	t.Helper()

	cfg := config.LoadConfig("..")

	autowpDB, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", autowpDB)

	db, err := sql.Open("postgres", cfg.PostgresDSN)
	require.NoError(t, err)

	goquPostgresDB := goqu.New("postgres", db)

	client := gocloak.NewClient(cfg.Keycloak.URL)

	usersRepository := users.NewRepository(
		goquDB,
		goquPostgresDB,
		cfg.UsersSalt,
		cfg.Languages,
		client,
		cfg.Keycloak,
	)

	hostsManager := hosts.NewManager(cfg.Languages)

	telegramService := telegram.NewService(cfg.Telegram, goquDB, hostsManager)

	messagingRepository := messaging.NewRepository(goquDB, telegramService)

	repo := NewRepository(goquDB, usersRepository, messagingRepository, hostsManager)

	return repo
}

func TestCleanupDeleted(t *testing.T) {
	t.Parallel()

	s := createRepository(t)

	ctx := context.Background()

	_, err := s.CleanupDeleted(ctx)
	require.NoError(t, err)
}
