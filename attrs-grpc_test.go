package goautowp

import (
	"context"
	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"google.golang.org/grpc/metadata"
	"testing"
	"time"

	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetUnits(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewAttrsClient(conn)

	_, err = client.GetUnits(
		ctx,
		&emptypb.Empty{},
	)
	require.NoError(t, err)
}

func TestGetZones(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewAttrsClient(conn)

	_, err = client.GetZones(
		ctx,
		&emptypb.Empty{},
	)
	require.NoError(t, err)
}

func TestGetAttributeTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewAttrsClient(conn)

	_, err = client.GetAttributeTypes(
		ctx,
		&emptypb.Empty{},
	)
	require.NoError(t, err)
}

func TestGetAttributes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ctxTimeout, cancel := context.WithTimeout(ctx, 5000*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	cfg := config.LoadConfig(".")

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctxTimeout, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewAttrsClient(conn)

	_, err = client.GetAttributes(
		metadata.AppendToOutgoingContext(ctxTimeout, authorizationHeader, bearerPrefix+token.AccessToken),
		&AttrAttributesRequest{
			ZoneId: 1,
		},
	)
	require.NoError(t, err)
}

func TestGetZoneAttributes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	client := NewAttrsClient(conn)

	_, err = client.GetZoneAttributes(
		ctx,
		&AttrZoneAttributesRequest{
			ZoneId: 1,
		},
	)
	require.NoError(t, err)
}

func TestGetListOptions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ctxTimeout, cancel := context.WithTimeout(ctx, 5000*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		"localhost",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	defer util.Close(conn)

	cfg := config.LoadConfig(".")

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(ctxTimeout, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	client := NewAttrsClient(conn)

	_, err = client.GetListOptions(
		metadata.AppendToOutgoingContext(ctxTimeout, authorizationHeader, bearerPrefix+token.AccessToken),
		&AttrListOptionsRequest{
			AttributeId: 1,
		},
	)
	require.NoError(t, err)
}
