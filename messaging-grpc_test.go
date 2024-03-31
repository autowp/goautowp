package goautowp

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func TestMessaging(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)
	messagingClient := NewMessagingClient(conn)

	cfg := config.LoadConfig(".")

	// cnt := NewContainer(cfg)
	// defer util.Close(cnt)

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	usersClient := NewUsersClient(conn)

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	// tester
	testerToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, testUsername, testPassword)
	require.NoError(t, err)
	require.NotNil(t, testerToken)

	// tester (me)
	tester, err := usersClient.Me(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+testerToken.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	// create
	_, err = messagingClient.CreateMessage(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&MessagingCreateMessage{
			UserId: tester.Id,
			Text:   "Test message",
		},
	)
	require.NoError(t, err)

	// get message
	r, err := messagingClient.GetMessages(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&MessagingGetMessagesRequest{
			UserId: tester.Id,
			Folder: "sent",
			Page:   1,
		},
	)
	require.NoError(t, err)
	require.Equal(t, "Test message", r.Items[0].Text)
}
