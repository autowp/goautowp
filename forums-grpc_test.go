package goautowp

import (
	"context"
	"database/sql"
	"testing"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func TestGetThemes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewForumsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	_, token := getUserWithCleanHistory(t, conn, cfg, goquDB, testUsername, testPassword)

	themes, err := client.GetThemes(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token),
		&APIGetForumsThemesRequest{},
	)
	require.NoError(t, err)
	require.NotEmpty(t, themes.Items)
}

func TestGetTheme(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewForumsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	_, token := getUserWithCleanHistory(t, conn, cfg, goquDB, testUsername, testPassword)

	theme, err := client.GetTheme(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token),
		&APIGetForumsThemeRequest{Id: 2},
	)
	require.NoError(t, err)
	require.NotEmpty(t, theme)
	require.NotEmpty(t, theme.Id)
	require.NotEmpty(t, theme.Name)
}

func TestGetLastTopicAndLastMessage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewForumsClient(conn)
	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	require.NoError(t, err)

	goquDB := goqu.New("mysql", db)

	_, token := getUserWithCleanHistory(t, conn, cfg, goquDB, testUsername, testPassword)

	topicID, err := client.CreateTopic(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token),
		&APICreateTopicRequest{
			ThemeId:            2,
			Name:               "Topic name",
			Message:            "Test message",
			ModeratorAttention: false,
			Subscription:       true,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, topicID)

	topic, err := client.GetLastTopic(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token),
		&APIGetForumsThemeRequest{Id: 2},
	)
	require.NoError(t, err)
	require.NotEmpty(t, topic)

	message, err := client.GetLastMessage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token),
		&APIGetForumsTopicRequest{Id: topic.Id},
	)
	require.NoError(t, err)
	require.NotEmpty(t, message)

	topics, err := client.GetTopics(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+token),
		&APIGetForumsTopicsRequest{ThemeId: 2, Page: 1},
	)
	require.NoError(t, err)
	require.NotEmpty(t, topics.Items)
}
