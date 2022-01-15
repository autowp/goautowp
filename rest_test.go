package goautowp

import (
	"context"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetVehicleTypesInaccessibleAnonymously(t *testing.T) {
	cnt := NewContainer(config.LoadConfig("."))
	defer util.Close(cnt)
	srv, err := cnt.GRPCServer()
	require.NoError(t, err)

	_, err = srv.GetVehicleTypes(context.Background(), &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypesInaccessibleWithEmptyToken(t *testing.T) {
	cnt := NewContainer(config.LoadConfig("."))
	defer util.Close(cnt)
	srv, err := cnt.GRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer "}))

	_, err = srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypesInaccessibleWithInvalidToken(t *testing.T) {
	cnt := NewContainer(config.LoadConfig("."))
	defer util.Close(cnt)
	srv, err := cnt.GRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer abc"}))

	_, err = srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypesInaccessibleWithWronglySignedToken(t *testing.T) {
	cnt := NewContainer(config.LoadConfig("."))
	defer util.Close(cnt)
	srv, err := cnt.GRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJkZWZhdWx0Iiwic3ViIjoiMSJ9.yuzUurjlDfEKchYseIrHQ1D5_RWnSuMxM-iK9FDNlQBBw8kCz3H-94xHvyd9pAA6Ry2-YkGi1v6Y3AHIpkDpcQ"}))

	_, err = srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypesInaccessibleWithoutModeratePrivilege(t *testing.T) {
	cfg := config.LoadConfig(".")

	ctx := context.Background()

	cnt := NewContainer(cfg)
	defer util.Close(cnt)
	oauth, err := cnt.OAuth()
	require.NoError(t, err)

	token, _, err := oauth.TokenByPassword(ctx, testUsername, testPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	srv, err := cnt.GRPCServer()
	require.NoError(t, err)

	_, err = srv.GetVehicleTypes(
		metadata.NewIncomingContext(
			ctx,
			metadata.New(map[string]string{"authorization": "Bearer " + token.AccessToken}),
		),
		&emptypb.Empty{},
	)
	require.Error(t, err)
}

func TestGetVehicleTypes(t *testing.T) {
	cfg := config.LoadConfig(".")

	ctx := context.Background()

	cnt := NewContainer(cfg)
	defer util.Close(cnt)
	oauth, err := cnt.OAuth()
	require.NoError(t, err)

	token, _, err := oauth.TokenByPassword(ctx, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	srv, err := cnt.GRPCServer()
	require.NoError(t, err)

	result, err := srv.GetVehicleTypes(
		metadata.NewIncomingContext(
			ctx,
			metadata.New(map[string]string{"authorization": "Bearer " + token.AccessToken}),
		),
		&emptypb.Empty{},
	)
	require.NoError(t, err)

	require.Greater(t, len(result.Items), 0)
}
