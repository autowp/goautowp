package messaging

import (
	"context"
	"database/sql"
	"math/rand"
	"strconv"
	"testing"

	"github.com/google/uuid"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/telegram"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql" // enable mysql dialect
	_ "github.com/go-sql-driver/mysql"               // enable mysql driver
	"github.com/stretchr/testify/require"
)

func createRepository(t *testing.T) *Repository {
	t.Helper()

	cfg := config.LoadConfig("..")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)
	hostsManager := hosts.NewManager(cfg.Languages)
	tg := telegram.NewService(cfg.Telegram, goquDB, hostsManager)

	s := NewRepository(goquDB, tg)

	return s
}

func TestGetUserNewMessagesCount(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	_, err := s.GetUserNewMessagesCount(context.Background(), 1)
	require.NoError(t, err)
}

func TestGetInboxCount(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	_, err := s.GetInboxCount(context.Background(), 1)
	require.NoError(t, err)
}

func TestGetInboxNewCount(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	_, err := s.GetInboxNewCount(context.Background(), 1)
	require.NoError(t, err)
}

func TestGetSentCount(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	_, err := s.GetSentCount(context.Background(), 1)
	require.NoError(t, err)
}

func TestGetSystemCount(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	_, err := s.GetSystemCount(context.Background(), 1)
	require.NoError(t, err)
}

func TestGetSystemNewCount(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	_, err := s.GetSystemNewCount(context.Background(), 1)
	require.NoError(t, err)
}

func TestGetDialogCount(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	_, err := s.GetDialogCount(context.Background(), 1, 2)
	require.NoError(t, err)
}

func TestDeleteMessage(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	err := s.DeleteMessage(context.Background(), 1, 1)
	require.NoError(t, err)
}

func TestClearSent(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	err := s.ClearSent(context.Background(), 1)
	require.NoError(t, err)
}

func TestClearSystem(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	err := s.ClearSystem(context.Background(), 1)
	require.NoError(t, err)
}

func TestGetInbox(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	_, _, err := s.GetInbox(context.Background(), 1, 1)
	require.NoError(t, err)
}

func TestGetSentbox(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	_, _, err := s.GetSentbox(context.Background(), 1, 1)
	require.NoError(t, err)
}

func TestGetSystembox(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	_, _, err := s.GetSystembox(context.Background(), 1, 1)
	require.NoError(t, err)
}

func TestGetDialogbox(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	_, _, err := s.GetDialogbox(context.Background(), 1, 2, 1)
	require.NoError(t, err)
}

func createRandomUser(t *testing.T, s *Repository) int64 {
	t.Helper()

	emailAddr := "test" + strconv.Itoa(rand.Int()) + "@example.com" //nolint: gosec
	name := "ivan"
	r, err := s.db.Insert("users").
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
		Executor().Exec()
	require.NoError(t, err)

	id, err := r.LastInsertId()
	require.NoError(t, err)

	return id
}

func TestDialogCount(t *testing.T) {
	t.Parallel()

	s := createRepository(t)
	ctx := context.Background()

	user1 := createRandomUser(t, s)
	user2 := createRandomUser(t, s)

	countBefore, err := s.GetDialogCount(ctx, user1, user2)
	require.NoError(t, err)

	err = s.CreateMessage(ctx, user1, user2, "Test message")
	require.NoError(t, err)

	countAfter, err := s.GetDialogCount(ctx, user1, user2)
	require.NoError(t, err)

	require.Greater(t, countAfter, countBefore)

	messages, _, err := s.GetSentbox(context.Background(), user1, 1)
	require.NoError(t, err)
	require.Equal(t, countAfter, messages[0].DialogCount)
}
