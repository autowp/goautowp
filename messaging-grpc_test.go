package goautowp

import (
	"context"
	"github.com/Nerzal/gocloak/v11"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestMessaging(t *testing.T) {

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer util.Close(conn)
	messagingClient := NewMessagingClient(conn)

	cfg := config.LoadConfig(".")

	//cnt := NewContainer(cfg)
	//defer util.Close(cnt)

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
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+testerToken.AccessToken),
		&APIMeRequest{},
	)
	require.NoError(t, err)

	// create
	_, err = messagingClient.CreateMessage(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&MessagingCreateMessage{
			UserId: tester.Id,
			Text:   "Test message",
		},
	)
	require.NoError(t, err)

	// get message
	r, err := messagingClient.GetMessages(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+adminToken.AccessToken),
		&MessagingGetMessagesRequest{
			UserId: tester.Id,
			Folder: "sent",
			Page:   1,
		},
	)
	require.NoError(t, err)
	require.Equal(t, "Test message", r.Items[0].Text)
}
