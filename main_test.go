package goautowp

import (
	"context"
	"net"
	"os"
	"testing"

	"google.golang.org/grpc"

	"github.com/autowp/goautowp/config"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/test/bufconn"
)

var (
	cnt       *Container
	bufDialer func(context.Context, string) (net.Conn, error)
)

func TestMain(m *testing.M) {
	const bufSize = 1024 * 1024

	var lis *bufconn.Listener

	cfg := config.LoadConfig(".")
	cnt = NewContainer(cfg)

	grpcServer := grpc.NewServer()

	articlesSrv, err := cnt.ArticlesGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterArticlesServer(grpcServer, articlesSrv)

	commentsSrv, err := cnt.CommentsGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterCommentsServer(grpcServer, commentsSrv)

	forumsSrv, err := cnt.ForumsGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterForumsServer(grpcServer, forumsSrv)

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

	textSrv, err := cnt.TextGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterTextServer(grpcServer, textSrv)

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

	autowpSrv, err := cnt.GRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterAutowpServer(grpcServer, autowpSrv)

	attrsSrv, err := cnt.AttrsGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterAttrsServer(grpcServer, attrsSrv)

	mapSrv, err := cnt.MapGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterMapServer(grpcServer, mapSrv)

	pictursSrv, err := cnt.PicturesGRPCServer()
	if err != nil {
		panic(err)
	}

	RegisterPicturesServer(grpcServer, pictursSrv)

	lis = bufconn.Listen(bufSize)

	bufDialer = func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logrus.Errorf("Server exited with error: %v", err)
		}
	}()

	/*return cnt, func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}*/

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logrus.Errorf("Server exited with error: %v", err)
		}
	}()

	os.Exit(m.Run())
}
