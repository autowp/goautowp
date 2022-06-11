package goautowp

import (
	"context"
	"net"

	"github.com/autowp/goautowp/config"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var container *Container

var lis *bufconn.Listener

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func getContainer() *Container {
	if container == nil {
		cfg := config.LoadConfig(".")
		container = NewContainer(cfg)
	}

	return container
}

func init() { // nolint: gochecknoinits
	cnt := getContainer()

	grpcServer := grpc.NewServer()

	contactsSrv, err := cnt.ContactsGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterContactsServer(grpcServer, contactsSrv)

	donationsSrv, err := cnt.DonationsGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterDonationsServer(grpcServer, donationsSrv)

	usersSrv, err := cnt.UsersGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterUsersServer(grpcServer, usersSrv)

	itemsSrv, err := cnt.ItemsGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterItemsServer(grpcServer, itemsSrv)

	messagingSrv, err := cnt.MessagingGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterMessagingServer(grpcServer, messagingSrv)

	statsSrv, err := cnt.StatisticsGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterStatisticsServer(grpcServer, statsSrv)

	lis = bufconn.Listen(bufSize)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logrus.Errorf("Server exited with error: %v", err)
		}
	}()
}
