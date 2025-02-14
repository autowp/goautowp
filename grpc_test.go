package goautowp

import (
	"testing"

	"github.com/autowp/goautowp/config"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetBrandVehicleTypes(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	grpcClient := NewAutowpClient(conn)

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()

	// tester
	testerToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, testUsername, testPassword)
	require.NoError(t, err)
	require.NotNil(t, testerToken)

	_, err = grpcClient.GetBrandVehicleTypes(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+testerToken.AccessToken),
		&GetBrandVehicleTypesRequest{
			BrandId: 1,
		},
	)
	require.NoError(t, err)
}

func TestGetPerspectives(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	grpcClient := NewAutowpClient(conn)

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()

	// tester
	testerToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, testUsername, testPassword)
	require.NoError(t, err)
	require.NotNil(t, testerToken)

	_, err = grpcClient.GetPerspectives(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+testerToken.AccessToken),
		&emptypb.Empty{},
	)
	require.NoError(t, err)
}

func TestGetPerspectivePages(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	grpcClient := NewAutowpClient(conn)

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()

	// tester
	testerToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, testUsername, testPassword)
	require.NoError(t, err)
	require.NotNil(t, testerToken)

	_, err = grpcClient.GetPerspectivePages(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+testerToken.AccessToken),
		&emptypb.Empty{},
	)
	require.NoError(t, err)
}

func TestGetSpecs(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	grpcClient := NewAutowpClient(conn)

	cfg := config.LoadConfig(".")

	kc := cnt.Keycloak()

	// tester
	testerToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, testUsername, testPassword)
	require.NoError(t, err)
	require.NotNil(t, testerToken)

	_, err = grpcClient.GetSpecs(
		metadata.AppendToOutgoingContext(ctx, authorizationHeader, bearerPrefix+testerToken.AccessToken),
		&emptypb.Empty{},
	)
	require.NoError(t, err)
}
