package goautowp

import (
	"context"
	"github.com/Nerzal/gocloak/v9"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestCreateDeleteContact(t *testing.T) {

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer util.Close(conn)
	client := NewContactsClient(conn)

	cfg := config.LoadConfig(".")

	keycloakClient := gocloak.NewClient(cfg.Keycloak.URL)

	var contactUserID int64 = 1

	// create
	_, err = client.CreateContact(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+getUserToken(t, adminUsername, adminPassword, keycloakClient, cfg.Keycloak)),
		&CreateContactRequest{UserId: contactUserID},
	)
	require.NoError(t, err)

	// get contact
	_, err = client.GetContact(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+getUserToken(t, adminUsername, adminPassword, keycloakClient, cfg.Keycloak)),
		&GetContactRequest{UserId: contactUserID},
	)
	require.NoError(t, err)

	// get contacts
	items, err := client.GetContacts(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+getUserToken(t, adminUsername, adminPassword, keycloakClient, cfg.Keycloak)),
		&GetContactsRequest{
			Fields: []string{"avatar", "gravatar"},
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	var contactUser *Contact
	for _, i := range items.Items {
		if i.ContactUserId == contactUserID {
			contactUser = i
			break
		}
	}
	require.NotNil(t, contactUser)
	require.NotEmpty(t, contactUser.GetUser().GetGravatar())

	// delete
	_, err = client.DeleteContact(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+getUserToken(t, adminUsername, adminPassword, keycloakClient, cfg.Keycloak)),
		&DeleteContactRequest{UserId: 1},
	)
	require.NoError(t, err)
}
