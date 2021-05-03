package goautowp

import (
	"context"
	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"testing"
)

var lis *bufconn.Listener

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestCreateDeleteContact(t *testing.T) {

	ctx := context.Background()
	rpcCreds := oauth.NewOauthAccess(&oauth2.Token{AccessToken: adminAccessToken})
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithPerRPCCredentials(rpcCreds),
		grpc.WithBlock(),
		grpc.WithContextDialer(bufDialer),
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer util.Close(conn)

	client := NewAutowpClient(conn)

	// create
	_, err = client.CreateContact(ctx, &CreateContactRequest{UserId: 1})
	require.NoError(t, err)

	// get contact
	_, err = client.GetContact(ctx, &GetContactRequest{UserId: 1})
	require.NoError(t, err)

	// get contacts
	_, err = client.GetContacts(ctx, &GetContactsRequest{})
	require.NoError(t, err)

	// delete
	_, err = client.DeleteContact(ctx, &DeleteContactRequest{UserId: 1})
	require.NoError(t, err)
}
