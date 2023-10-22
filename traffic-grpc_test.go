package goautowp

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestHttpBanPost(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	cnt := NewContainer(cfg)
	defer util.Close(cnt)

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(context.Background(), "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	srv, err := cnt.GRPCServer()
	require.NoError(t, err)

	trafficSrv, err := cnt.TrafficGRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.New(map[string]string{authorizationHeader: bearerPrefix + token.AccessToken}),
	)

	_, err = trafficSrv.DeleteFromBlacklist(ctx, &DeleteFromTrafficBlacklistRequest{Ip: "127.0.0.1"})
	require.NoError(t, err)

	_, err = trafficSrv.AddToBlacklist(ctx, &AddToTrafficBlacklistRequest{
		Ip:     "127.0.0.1",
		Period: 3,
		Reason: "Test",
	})
	require.NoError(t, err)

	ip, err := srv.GetIP(ctx, &APIGetIPRequest{
		Ip:     "127.0.0.1",
		Fields: []string{"blacklist"},
	})
	require.NoError(t, err)
	require.NotNil(t, ip.Blacklist)

	_, err = trafficSrv.DeleteFromBlacklist(ctx, &DeleteFromTrafficBlacklistRequest{Ip: "127.0.0.1"})
	require.NoError(t, err)

	ip, err = srv.GetIP(ctx, &APIGetIPRequest{Ip: "127.0.0.1"})
	require.NoError(t, err)
	require.Nil(t, ip.Blacklist)
}

func TestTop(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig(".")

	cnt := NewContainer(cfg)
	defer util.Close(cnt)

	kc := gocloak.NewClient(cfg.Keycloak.URL)
	token, err := kc.Login(context.Background(), "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, token)

	srv, err := cnt.TrafficGRPCServer()
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.New(map[string]string{authorizationHeader: bearerPrefix + token.AccessToken}),
	)

	_, err = srv.GetTop(ctx, &emptypb.Empty{})
	require.NoError(t, err)
}
