package goautowp

import (
	"context"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetVehicleTypesInaccessibleAnonymously(t *testing.T) {
	srv, err := NewContainer(LoadConfig()).GetGRPCServer()
	require.NoError(t, err)

	_, err = srv.GetVehicleTypes(context.Background(), &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypesInaccessibleWithEmptyToken(t *testing.T) {
	srv, err := NewContainer(LoadConfig()).GetGRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer "}))

	_, err = srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypesInaccessibleWithInvalidToken(t *testing.T) {
	srv, err := NewContainer(LoadConfig()).GetGRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer abc"}))

	_, err = srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypesInaccessibleWithWronglySignedToken(t *testing.T) {
	srv, err := NewContainer(LoadConfig()).GetGRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJkZWZhdWx0Iiwic3ViIjoiMSJ9.yuzUurjlDfEKchYseIrHQ1D5_RWnSuMxM-iK9FDNlQBBw8kCz3H-94xHvyd9pAA6Ry2-YkGi1v6Y3AHIpkDpcQ"}))

	_, err = srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypesInaccessibleWithoutModeratePrivilege(t *testing.T) {
	srv, err := NewContainer(LoadConfig()).GetGRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJkZWZhdWx0Iiwic3ViIjoiMSJ9.yuzUurjlDfEKchYseIrHQ1D5_RWnSuMxM-iK9FDNlQBBw8kCz3H-94xHvyd9pAA6Ry2-YkGi1v6Y3AHIpkDpcQ"}))

	_, err = srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.Error(t, err)
}

func TestGetVehicleTypes(t *testing.T) {
	srv, err := NewContainer(LoadConfig()).GetGRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJkZWZhdWx0Iiwic3ViIjoiMyJ9.tI-wPZ4BSqmpsZN0-SgWXaokzvB8T-uYWLR9OQurxPFNoPC56U3op1gSE5n2H02GYfDGig0Eyp6U0NbDpsQaAg"}))

	result, err := srv.GetVehicleTypes(ctx, &emptypb.Empty{})
	require.NoError(t, err)

	require.Greater(t, len(result.Items), 0)
}
