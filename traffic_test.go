package goautowp

import (
	"context"
	"github.com/Nerzal/gocloak/v9"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func createTrafficService(t *testing.T) *Traffic {
	s, err := getContainer().Traffic()
	require.NoError(t, err)

	return s
}

func TestAutoWhitelist(t *testing.T) {

	s := createTrafficService(t)

	ip := net.IPv4(66, 249, 73, 139) // google

	err := s.Ban.Add(ip, time.Hour, 9, "test")
	require.NoError(t, err)

	exists, err := s.Ban.Exists(ip)
	require.NoError(t, err)
	require.True(t, exists)

	err = s.Monitoring.Add(ip, time.Now())
	require.NoError(t, err)

	exists, err = s.Monitoring.ExistsIP(ip)
	require.NoError(t, err)
	require.True(t, exists)

	err = s.AutoWhitelist()
	require.NoError(t, err)

	exists, err = s.Ban.Exists(ip)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = s.Monitoring.ExistsIP(ip)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = s.Whitelist.Exists(ip)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestAutoBanByProfile(t *testing.T) {

	s := createTrafficService(t)

	profile := AutobanProfile{
		Limit:  3,
		Reason: "Test",
		Group:  []string{"hour", "tenminute", "minute"},
		Time:   time.Hour,
	}

	ip1 := net.IPv4(127, 0, 0, 1)
	ip2 := net.IPv4(127, 0, 0, 2)

	err := s.Monitoring.ClearIP(ip1)
	require.NoError(t, err)
	err = s.Monitoring.ClearIP(ip2)
	require.NoError(t, err)

	err = s.Ban.Remove(ip1)
	require.NoError(t, err)
	err = s.Ban.Remove(ip2)
	require.NoError(t, err)

	err = s.Monitoring.Add(ip1, time.Now())
	require.NoError(t, err)
	for i := 0; i < 4; i++ {
		err = s.Monitoring.Add(ip2, time.Now())
		require.NoError(t, err)
	}

	err = s.AutoBanByProfile(profile)
	require.NoError(t, err)

	exists, err := s.Ban.Exists(ip1)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = s.Ban.Exists(ip2)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestWhitelistedNotBanned(t *testing.T) {

	s := createTrafficService(t)

	profile := AutobanProfile{
		Limit:  3,
		Reason: "TestWhitelistedNotBanned",
		Group:  []string{"hour", "tenminute", "minute"},
		Time:   time.Hour,
	}

	ip := net.IPv4(178, 154, 244, 21)

	err := s.Whitelist.Add(ip, "TestWhitelistedNotBanned")
	require.NoError(t, err)

	for i := 0; i < 4; i++ {
		err = s.Monitoring.Add(ip, time.Now())
		require.NoError(t, err)
	}

	err = s.AutoWhitelistIP(ip)
	require.NoError(t, err)

	err = s.AutoBanByProfile(profile)
	require.NoError(t, err)

	exists, err := s.Ban.Exists(ip)
	require.NoError(t, err)
	require.False(t, exists)
}

func TestHttpBanPost(t *testing.T) {
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
		metadata.New(map[string]string{"authorization": "Bearer " + token.AccessToken}),
	)

	_, err = trafficSrv.DeleteFromTrafficBlacklist(ctx, &DeleteFromTrafficBlacklistRequest{Ip: "127.0.0.1"})
	require.NoError(t, err)

	_, err = trafficSrv.AddToTrafficBlacklist(ctx, &AddToTrafficBlacklistRequest{
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

	_, err = trafficSrv.DeleteFromTrafficBlacklist(ctx, &DeleteFromTrafficBlacklistRequest{Ip: "127.0.0.1"})
	require.NoError(t, err)

	ip, err = srv.GetIP(ctx, &APIGetIPRequest{Ip: "127.0.0.1"})
	require.NoError(t, err)
	require.Nil(t, ip.Blacklist)
}

func TestTop(t *testing.T) {

	s := createTrafficService(t)

	err := s.Ban.Clear()
	require.NoError(t, err)

	err = s.Monitoring.Clear()
	require.NoError(t, err)

	err = s.Monitoring.Add(net.IPv4(192, 168, 0, 1), time.Now())
	require.NoError(t, err)

	now := time.Now()
	for i := 0; i < 10; i++ {
		err = s.Monitoring.Add(net.IPv6loopback, now)
		require.NoError(t, err)
	}

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
		metadata.New(map[string]string{"authorization": "Bearer " + token.AccessToken}),
	)

	top, err := srv.GetTrafficTop(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, top.Items[0].Ip, "::1")
	require.EqualValues(t, top.Items[0].Count, 10)
	require.Equal(t, top.Items[0].WhoisUrl, "https://nic.ru/whois/?query=%3A%3A1")
	// require.Equal(t, `{"items":[{"ip":"::1","count":10,"ban":null,"in_whitelist":false,"whois_url":"http://nic.ru/whois/?query=%3A%3A1"},{"ip":"192.168.0.1","count":1,"ban":null,"in_whitelist":false,"whois_url":"http://nic.ru/whois/?query=192.168.0.1"}]}`, string(body))
}
