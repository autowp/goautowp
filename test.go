package goautowp

import (
	"context"
	"database/sql"
	"github.com/Nerzal/gocloak/v9"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/email"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/users"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/casbin/casbin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"net"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func init() {

	cfg := config.LoadConfig(".")

	db, err := sql.Open("mysql", cfg.AutowpDSN)
	if err != nil {
		panic(err)
	}

	emailSender := &email.MockSender{}

	enforcer := casbin.NewEnforcer("model.conf", "policy.csv")

	keycloak := gocloak.NewClient(cfg.Keycloak.URL)

	contactsRepository := NewContactsRepository(db)
	userRepository := users.NewRepository(
		db,
		cfg.UsersSalt,
		cfg.EmailSalt,
		cfg.Languages,
		emailSender,
		keycloak,
		cfg.Keycloak,
	)

	userExtractor := NewUserExtractor(NewContainer(cfg))

	lis = bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()

	contactsSrv := NewContactsGRPCServer(
		db,
		contactsRepository,
		userRepository,
		userExtractor,
		keycloak,
		cfg.Keycloak,
	)
	RegisterContactsServer(grpcServer, contactsSrv)

	usersSrv := NewUsersGRPCServer(
		db,
		enforcer,
		contactsRepository,
		userRepository,
		NewEvents(db),
		cfg.Languages,
		false,
		NewPasswordRecovery(
			db,
			false,
			cfg.Languages,
			emailSender,
		),
		userExtractor,
		keycloak,
		cfg.Keycloak,
	)
	RegisterUsersServer(grpcServer, usersSrv)

	itemsSrv := NewItemsGRPCServer(
		items.NewRepository(db),
		memcache.New(cfg.Memcached...),
	)
	RegisterItemsServer(grpcServer, itemsSrv)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logrus.Errorf("Server exited with error: %v", err)
		}
	}()
}
