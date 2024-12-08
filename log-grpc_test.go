package goautowp

import (
	"context"
	"testing"

	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func TestGetEvents(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()
	token, err := kc.Login(context.Background(), "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewLogClient(conn)

	ctx := metadata.AppendToOutgoingContext(context.Background(), authorizationHeader, bearerPrefix+token.AccessToken)

	_, err = client.GetEvents(ctx, &LogEventsRequest{})
	require.NoError(t, err)
}

func TestGetEventsWithFilters(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()
	token, err := kc.Login(context.Background(), "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewLogClient(conn)

	ctx := metadata.AppendToOutgoingContext(context.Background(), authorizationHeader, bearerPrefix+token.AccessToken)

	_, err = client.GetEvents(ctx, &LogEventsRequest{
		UserId:    1,
		PictureId: 1,
		ItemId:    1,
		ArticleId: 1,
		Page:      2,
	})
	require.NoError(t, err)
}
