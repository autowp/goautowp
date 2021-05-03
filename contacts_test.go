package goautowp

import (
	"context"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestCreateDeleteContact(t *testing.T) {
	ctx := context.Background()
	config := LoadConfig()
	container := NewContainer(config)
	srv, err := container.GetGRPCServer()
	require.NoError(t, err)

	ctx = metadata.NewIncomingContext(ctx, metadata.New(map[string]string{"authorization": "Bearer " + adminAccessToken}))

	// create
	_, err = srv.CreateContact(ctx, &CreateContactRequest{UserId: 1})
	require.NoError(t, err)

	// get contact
	_, err = srv.GetContact(ctx, &GetContactRequest{UserId: 1})
	require.NoError(t, err)

	// get contacts
	_, err = srv.GetContacts(ctx, &GetContactsRequest{})
	require.NoError(t, err)

	// delete
	_, err = srv.DeleteContact(ctx, &DeleteContactRequest{UserId: 1})
	require.NoError(t, err)
}
