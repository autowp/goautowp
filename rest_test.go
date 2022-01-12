package goautowp

import (
	"context"
	"github.com/Nerzal/gocloak/v9"
	"github.com/autowp/goautowp/config"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetVehicleTypesInaccessibleAnonymously(t *testing.T) {
	srv, err := NewContainer(config.LoadConfig(".")).GRPCServer()
	require.NoError(t, err)

	_, err = srv.GetVehicleTypes(context.Background(), &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypesInaccessibleWithEmptyToken(t *testing.T) {
	srv, err := NewContainer(config.LoadConfig(".")).GRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer "}))

	_, err = srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypesInaccessibleWithInvalidToken(t *testing.T) {
	srv, err := NewContainer(config.LoadConfig(".")).GRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer abc"}))

	_, err = srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypesInaccessibleWithWronglySignedToken(t *testing.T) {
	srv, err := NewContainer(config.LoadConfig(".")).GRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJkZWZhdWx0Iiwic3ViIjoiMSJ9.yuzUurjlDfEKchYseIrHQ1D5_RWnSuMxM-iK9FDNlQBBw8kCz3H-94xHvyd9pAA6Ry2-YkGi1v6Y3AHIpkDpcQ"}))

	_, err = srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypesInaccessibleWithoutModeratePrivilege(t *testing.T) {
	cfg := config.LoadConfig(".")

	keycloakClient := gocloak.NewClient(cfg.Keycloak.URL)

	srv, err := NewContainer(cfg).GRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.New(map[string]string{"authorization": "Bearer " + getUserToken(t, testUsername, testPassword, keycloakClient, cfg.Keycloak)}),
	)

	_, err = srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypes(t *testing.T) {
	cfg := config.LoadConfig(".")

	srv, err := NewContainer(cfg).GRPCServer()
	require.NoError(t, err)

	keycloakClient := gocloak.NewClient(cfg.Keycloak.URL)

	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.New(map[string]string{"authorization": "Bearer " + getUserToken(t, adminUsername, adminPassword, keycloakClient, cfg.Keycloak)}),
	)

	result, err := srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	require.Greater(t, len(result.Items), 0)
}
