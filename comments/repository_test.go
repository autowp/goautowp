package comments

import (
	"context"
	"database/sql"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v11"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/telegram"
	"github.com/autowp/goautowp/users"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"    // enable mysql dialect
	_ "github.com/doug-martin/goqu/v9/dialect/postgres" // enable postgres dialect
	"github.com/google/uuid"
	_ "github.com/lib/pq" // enable postgres driver
	"github.com/stretchr/testify/require"
)

func createRandomUser(ctx context.Context, t *testing.T, db *goqu.Database) int64 {
	t.Helper()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	emailAddr := "test" + strconv.Itoa(random.Int()) + "@example.com"
	name := "ivan"
	r, err := db.Insert("users").
		Rows(goqu.Record{
			"login":            nil,
			"e_mail":           emailAddr,
			"password":         nil,
			"email_to_check":   nil,
			"hide_e_mail":      1,
			"email_check_code": nil,
			"name":             name,
			"reg_date":         goqu.L("NOW()"),
			"last_online":      goqu.L("NOW()"),
			"timezone":         "Europe/Moscow",
			"last_ip":          goqu.L("INET6_ATON('127.0.0.1')"),
			"language":         "en",
			"role":             "user",
			"uuid":             goqu.L("UUID_TO_BIN(?)", uuid.New().String()),
		}).
		Executor().ExecContext(ctx)
	require.NoError(t, err)

	id, err := r.LastInsertId()
	require.NoError(t, err)

	return id
}

func createRepository(t *testing.T) (*Repository, *goqu.Database) {
	t.Helper()

	cfg := config.LoadConfig("..")

	autowpDB, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", autowpDB)

	postgresDB, err := sql.Open("postgres", cfg.PostgresDSN)
	require.NoError(t, err)

	goquPostgresDB := goqu.New("postgres", postgresDB)

	client := gocloak.NewClient(cfg.Keycloak.URL)

	usersRepository := users.NewRepository(
		goquDB,
		goquPostgresDB,
		cfg.UsersSalt,
		cfg.Languages,
		client,
		cfg.Keycloak,
		cfg.MessageInterval,
	)

	hostsManager := hosts.NewManager(cfg.Languages)

	telegramService := telegram.NewService(cfg.Telegram, goquDB, hostsManager)

	messagingRepository := messaging.NewRepository(goquDB, telegramService)

	i, err := i18nbundle.New()
	require.NoError(t, err)

	repo := NewRepository(goquDB, usersRepository, messagingRepository, hostsManager, i)

	return repo, goquDB
}

func TestCleanupDeleted(t *testing.T) {
	t.Parallel()

	s, _ := createRepository(t)

	ctx := context.Background()

	_, err := s.CleanupDeleted(ctx)
	require.NoError(t, err)
}

func TestRefreshRepliesCount(t *testing.T) {
	t.Parallel()

	s, _ := createRepository(t)

	ctx := context.Background()

	_, err := s.RefreshRepliesCount(ctx)
	require.NoError(t, err)
}

func TestAdd(t *testing.T) {
	t.Parallel()

	s, db := createRepository(t)
	ctx := context.Background()
	userID := createRandomUser(ctx, t, db)

	var (
		commentType       = TypeIDPictures
		itemID      int64 = 1
	)

	_, err := s.Add(ctx, commentType, itemID, 0, userID, "Test message", "127.0.0.1", false)
	require.NoError(t, err)
}

func TestCleanBrokenMessages(t *testing.T) {
	t.Parallel()

	s, _ := createRepository(t)

	_, err := s.CleanBrokenMessages(context.Background())
	require.NoError(t, err)
}

func TestCleanTopics(t *testing.T) {
	t.Parallel()

	s, _ := createRepository(t)

	_, err := s.CleanTopics(context.Background())
	require.NoError(t, err)
}
