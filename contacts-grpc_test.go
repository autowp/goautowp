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

func TestCreateDeleteContact(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewContactsClient(conn)

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
	_, err = client.CreateContact(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&CreateContactRequest{UserId: tester.Id},
	)
	require.NoError(t, err)

	// get contact
	_, err = client.GetContact(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&GetContactRequest{UserId: tester.Id},
	)
	require.NoError(t, err)

	// get contacts
	items, err := client.GetContacts(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&GetContactsRequest{},
	)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	var contactUser *Contact

	for _, i := range items.Items {
		if i.ContactUserId == tester.Id {
			contactUser = i

			break
		}
	}

	require.NotNil(t, contactUser)
	require.NotEmpty(t, contactUser.GetUser().GetGravatar())

	// delete
	_, err = client.DeleteContact(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+adminToken.AccessToken),
		&DeleteContactRequest{UserId: tester.Id},
	)
	require.NoError(t, err)
}
