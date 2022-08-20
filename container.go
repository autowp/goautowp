package goautowp

import (
	"database/sql"
	"net/http"
	"time"

	"google.golang.org/grpc/reflection"

	"github.com/autowp/goautowp/traffic"

	"github.com/Nerzal/gocloak/v11"
	"github.com/autowp/goautowp/ban"
	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/email"
	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/itemofday"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/telegram"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/casbin/casbin"
	"github.com/doug-martin/goqu/v9"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpclogrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpcctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const readHeaderTimeout = time.Second * 30

// Container Container.
type Container struct {
	autowpDB             *sql.DB
	banRepository        *ban.Repository
	catalogue            *Catalogue
	commentsRepository   *comments.Repository
	config               config.Config
	commentsGrpcServer   *CommentsGRPCServer
	contactsGrpcServer   *ContactsGRPCServer
	contactsRepository   *ContactsRepository
	duplicateFinder      *DuplicateFinder
	donationsGrpcServer  *DonationsGRPCServer
	emailSender          email.Sender
	enforcer             *casbin.Enforcer
	events               *Events
	feedback             *Feedback
	forums               *Forums
	goquDB               *goqu.Database
	goquPostgresDB       *goqu.Database
	grpcServer           *GRPCServer
	hostsManager         *hosts.Manager
	imageStorage         *storage.Storage
	itemOfDayRepository  *itemofday.Repository
	itemsGrpcServer      *ItemsGRPCServer
	itemsRepository      *items.Repository
	keyCloak             gocloak.GoCloak
	location             *time.Location
	messagingGrpcServer  *MessagingGRPCServer
	messagingRepository  *messaging.Repository
	privateHTTPServer    *http.Server
	privateRouter        *gin.Engine
	publicHTTPServer     *http.Server
	publicRouter         http.HandlerFunc
	telegramService      *telegram.Service
	traffic              *traffic.Traffic
	trafficGrpcServer    *TrafficGRPCServer
	usersRepository      *users.Repository
	usersGrpcServer      *UsersGRPCServer
	memcached            *memcache.Client
	auth                 *Auth
	mapGrpcServer        *MapGRPCServer
	picturesRepository   *pictures.Repository
	picturesGrpcServer   *PicturesGRPCServer
	statisticsGrpcServer *StatisticsGRPCServer
}

// NewContainer constructor.
func NewContainer(cfg config.Config) *Container {
	return &Container{
		config: cfg,
	}
}

func (s *Container) Close() error {
	s.banRepository = nil
	s.catalogue = nil
	s.commentsRepository = nil
	s.contactsRepository = nil
	s.duplicateFinder = nil
	s.traffic = nil
	s.usersRepository = nil
	s.feedback = nil

	if s.autowpDB != nil {
		err := s.autowpDB.Close()
		if err != nil {
			logrus.Error(err.Error())
			sentry.CaptureException(err)
		}

		s.autowpDB = nil
	}

	/*if s.goquPostgresDB != nil {
		s.goquPostgresDB.Close()
		s.goquPostgresDB = nil
	}*/

	return nil
}

func (s *Container) AutowpDB() (*sql.DB, error) {
	if s.autowpDB != nil {
		return s.autowpDB, nil
	}

	start := time.Now()

	const (
		connectionTimeout = 60 * time.Second
		reconnectDelay    = 100 * time.Millisecond
	)

	logrus.Info("Waiting for mysql")

	var (
		db  *sql.DB
		err error
	)

	for {
		db, err = sql.Open("mysql", s.config.AutowpDSN)
		if err != nil {
			return nil, err
		}

		err = db.Ping()
		if err == nil {
			logrus.Info("Started.")

			break
		}

		if time.Since(start) > connectionTimeout {
			return nil, err
		}

		logrus.Info(".")
		time.Sleep(reconnectDelay)
	}

	s.autowpDB = db

	return s.autowpDB, nil
}

func (s *Container) GoquDB() (*goqu.Database, error) {
	if s.goquDB == nil {
		db, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		s.goquDB = goqu.New("mysql", db)
	}

	return s.goquDB, nil
}

func (s *Container) GoquPostgresDB() (*goqu.Database, error) {
	if s.goquPostgresDB != nil {
		return s.goquPostgresDB, nil
	}

	start := time.Now()

	const (
		connectionTimeout = 60 * time.Second
		reconnectDelay    = 100 * time.Millisecond
	)

	logrus.Info("Waiting for postgres (goqu)")

	var (
		db  *sql.DB
		err error
	)

	for {
		db, err = sql.Open("postgres", s.config.PostgresDSN)
		if err != nil {
			return nil, err
		}

		err = db.Ping()
		if err == nil {
			logrus.Info("Started.")

			break
		}

		if time.Since(start) > connectionTimeout {
			return nil, err
		}

		logrus.Info(".")
		time.Sleep(reconnectDelay)
	}

	s.goquPostgresDB = goqu.New("postgres", db)

	return s.goquPostgresDB, nil
}

func (s *Container) BanRepository() (*ban.Repository, error) {
	if s.banRepository == nil {
		db, err := s.GoquPostgresDB()
		if err != nil {
			return nil, err
		}

		s.banRepository, err = ban.NewRepository(db)
		if err != nil {
			return nil, err
		}
	}

	return s.banRepository, nil
}

func (s *Container) Catalogue() (*Catalogue, error) {
	if s.catalogue == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.catalogue, err = NewCatalogue(db)
		if err != nil {
			return nil, err
		}
	}

	return s.catalogue, nil
}

func (s *Container) CommentsRepository() (*comments.Repository, error) {
	if s.commentsRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.commentsRepository = comments.NewRepository(db)
	}

	return s.commentsRepository, nil
}

func (s *Container) Config() config.Config {
	return s.config
}

func (s *Container) ContactsRepository() (*ContactsRepository, error) {
	if s.contactsRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.contactsRepository = NewContactsRepository(db)
	}

	return s.contactsRepository, nil
}

func (s *Container) DuplicateFinder() (*DuplicateFinder, error) {
	if s.duplicateFinder == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.duplicateFinder, err = NewDuplicateFinder(db)
		if err != nil {
			return nil, err
		}
	}

	return s.duplicateFinder, nil
}

func (s *Container) Enforcer() *casbin.Enforcer {
	if s.enforcer == nil {
		s.enforcer = casbin.NewEnforcer("model.conf", "policy.csv")
	}

	return s.enforcer
}

func (s *Container) Feedback() (*Feedback, error) {
	if s.feedback == nil {
		cfg := s.Config()

		emailSender := s.EmailSender()

		var err error

		s.feedback, err = NewFeedback(cfg.Feedback, cfg.Recaptcha, cfg.Captcha, emailSender)
		if err != nil {
			return nil, err
		}
	}

	return s.feedback, nil
}

func (s *Container) IPExtractor() (*IPExtractor, error) {
	banRepository, err := s.BanRepository()
	if err != nil {
		return nil, err
	}

	userRepository, err := s.UsersRepository()
	if err != nil {
		return nil, err
	}

	userExtractor, err := s.UserExtractor()
	if err != nil {
		return nil, err
	}

	return NewIPExtractor(s.Enforcer(), banRepository, userRepository, userExtractor), nil
}

func (s *Container) HostsManager() *hosts.Manager {
	if s.hostsManager == nil {
		s.hostsManager = hosts.NewManager(s.Config().Languages)
	}

	return s.hostsManager
}

// Location Location.
func (s *Container) Location() (*time.Location, error) {
	if s.location == nil {
		loc, err := time.LoadLocation("UTC")
		if err != nil {
			return nil, err
		}

		s.location = loc
	}

	return s.location, nil
}

func (s *Container) PicturesRepository() (*pictures.Repository, error) {
	if s.picturesRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.picturesRepository = pictures.NewRepository(db)
	}

	return s.picturesRepository, nil
}

func (s *Container) PrivateHTTPServer() (*http.Server, error) {
	if s.privateHTTPServer == nil {
		cfg := s.Config()

		router, err := s.PrivateRouter()
		if err != nil {
			return nil, err
		}

		s.privateHTTPServer = &http.Server{
			Addr:              cfg.PrivateRest.Listen,
			Handler:           router,
			ReadHeaderTimeout: readHeaderTimeout,
		}
	}

	return s.privateHTTPServer, nil
}

func (s *Container) PrivateRouter() (*gin.Engine, error) {
	if s.privateRouter != nil {
		return s.privateRouter, nil
	}

	trafficRepo, err := s.Traffic()
	if err != nil {
		return nil, err
	}

	usersRepo, err := s.UsersRepository()
	if err != nil {
		return nil, err
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(sentrygin.New(sentrygin.Options{}))

	trafficRepo.SetupPrivateRouter(r)
	usersRepo.SetupPrivateRouter(r)

	s.privateRouter = r

	return s.privateRouter, nil
}

func (s *Container) PublicHTTPServer() (*http.Server, error) {
	if s.publicHTTPServer == nil {
		cfg := s.Config()

		r, err := s.PublicRouter()
		if err != nil {
			return nil, err
		}

		s.publicHTTPServer = &http.Server{
			Addr:              cfg.PublicRest.Listen,
			Handler:           r,
			ReadHeaderTimeout: readHeaderTimeout,
		}
	}

	return s.publicHTTPServer, nil
}

type TokenForm struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

func (s *Container) PublicRouter() (http.HandlerFunc, error) {
	if s.publicRouter != nil {
		return s.publicRouter, nil
	}

	srv, err := s.GRPCServer()
	if err != nil {
		return nil, err
	}

	commentsSrv, err := s.CommentsGRPCServer()
	if err != nil {
		return nil, err
	}

	contactsSrv, err := s.ContactsGRPCServer()
	if err != nil {
		return nil, err
	}

	donationsSrv, err := s.DonationsGRPCServer()
	if err != nil {
		return nil, err
	}

	itemsSrv, err := s.ItemsGRPCServer()
	if err != nil {
		return nil, err
	}

	mapSrv, err := s.MapGRPCServer()
	if err != nil {
		return nil, err
	}

	trafficSrv, err := s.TrafficGRPCServer()
	if err != nil {
		return nil, err
	}

	picturesSrv, err := s.PicturesGRPCServer()
	if err != nil {
		return nil, err
	}

	messagingSrv, err := s.MessagingGRPCServer()
	if err != nil {
		return nil, err
	}

	usersSrv, err := s.UsersGRPCServer()
	if err != nil {
		return nil, err
	}

	statSrv, err := s.StatisticsGRPCServer()
	if err != nil {
		return nil, err
	}

	logrusLogger := logrus.New()
	logrusEntry := logrus.NewEntry(logrusLogger)

	grpclogrus.ReplaceGrpcLogger(logrusEntry)

	grpcServer := grpc.NewServer(
		grpcmiddleware.WithUnaryServerChain(
			grpcctxtags.UnaryServerInterceptor(grpcctxtags.WithFieldExtractor(grpcctxtags.CodeGenRequestFieldExtractor)),
			grpclogrus.UnaryServerInterceptor(logrusEntry),
		),
		grpcmiddleware.WithStreamServerChain(
			grpcctxtags.StreamServerInterceptor(grpcctxtags.WithFieldExtractor(grpcctxtags.CodeGenRequestFieldExtractor)),
			grpclogrus.StreamServerInterceptor(logrusEntry),
		),
	)
	RegisterAutowpServer(grpcServer, srv)
	RegisterCommentsServer(grpcServer, commentsSrv)
	RegisterContactsServer(grpcServer, contactsSrv)
	RegisterDonationsServer(grpcServer, donationsSrv)
	RegisterItemsServer(grpcServer, itemsSrv)
	RegisterMapServer(grpcServer, mapSrv)
	RegisterMessagingServer(grpcServer, messagingSrv)
	RegisterPicturesServer(grpcServer, picturesSrv)
	RegisterStatisticsServer(grpcServer, statSrv)
	RegisterTrafficServer(grpcServer, trafficSrv)
	RegisterUsersServer(grpcServer, usersSrv)

	reflection.Register(grpcServer)

	originFunc := func(origin string) bool {
		return util.Contains(s.config.PublicRest.Cors.Origin, origin)
	}
	wrappedGrpc := grpcweb.WrapServer(grpcServer, grpcweb.WithOriginFunc(originFunc))

	s.publicRouter = func(resp http.ResponseWriter, req *http.Request) {
		if wrappedGrpc.IsGrpcWebRequest(req) {
			wrappedGrpc.ServeHTTP(resp, req)

			return
		}
		// Fall back to gRPC server
		grpcServer.ServeHTTP(resp, req)
	}

	return s.publicRouter, nil
}

func (s *Container) TelegramService() (*telegram.Service, error) {
	if s.telegramService == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.telegramService = telegram.NewService(s.Config().Telegram, db, s.HostsManager())
	}

	return s.telegramService, nil
}

func (s *Container) Traffic() (*traffic.Traffic, error) {
	if s.traffic == nil {
		db, err := s.GoquPostgresDB()
		if err != nil {
			return nil, err
		}

		autowpDB, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		banRepository, err := s.BanRepository()
		if err != nil {
			return nil, err
		}

		userExtractor, err := s.UserExtractor()
		if err != nil {
			return nil, err
		}

		traf, err := traffic.NewTraffic(db, autowpDB, s.Enforcer(), banRepository, userExtractor)
		if err != nil {
			logrus.Error(err.Error())

			return nil, err
		}

		s.traffic = traf
	}

	return s.traffic, nil
}

func (s *Container) UserExtractor() (*users.UserExtractor, error) {
	is, err := s.ImageStorage()
	if err != nil {
		return nil, err
	}

	return users.NewUserExtractor(s.Enforcer(), is), nil
}

func (s *Container) UsersRepository() (*users.Repository, error) {
	if s.usersRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		postgresDB, err := s.GoquPostgresDB()
		if err != nil {
			return nil, err
		}

		cfg := s.Config()

		s.usersRepository = users.NewRepository(
			db,
			postgresDB,
			cfg.UsersSalt,
			cfg.Languages,
			s.Keycloak(),
			cfg.Keycloak,
		)
	}

	return s.usersRepository, nil
}

func (s *Container) ItemsRepository() (*items.Repository, error) {
	if s.itemsRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.itemsRepository = items.NewRepository(db)
	}

	return s.itemsRepository, nil
}

func (s *Container) Auth() (*Auth, error) {
	if s.auth == nil {
		cfg := s.Config()

		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		rep, err := s.UsersRepository()
		if err != nil {
			return nil, err
		}

		s.auth = NewAuth(db, s.Keycloak(), cfg.Keycloak, rep)
	}

	return s.auth, nil
}

func (s *Container) GRPCServer() (*GRPCServer, error) {
	if s.grpcServer == nil {
		catalogue, err := s.Catalogue()
		if err != nil {
			return nil, err
		}

		cfg := s.Config()

		commentsRepository, err := s.CommentsRepository()
		if err != nil {
			return nil, err
		}

		feedback, err := s.Feedback()
		if err != nil {
			return nil, err
		}

		forums, err := s.Forums()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		ipExtractor, err := s.IPExtractor()
		if err != nil {
			return nil, err
		}

		s.grpcServer = NewGRPCServer(
			auth,
			catalogue,
			cfg.Recaptcha,
			cfg.FileStorage,
			s.Enforcer(),
			commentsRepository,
			ipExtractor,
			feedback,
			forums,
		)
	}

	return s.grpcServer, nil
}

func (s *Container) StatisticsGRPCServer() (*StatisticsGRPCServer, error) {
	if s.statisticsGrpcServer == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.statisticsGrpcServer = NewStatisticsGRPCServer(
			db,
			s.Enforcer(),
			s.Config().About,
		)
	}

	return s.statisticsGrpcServer, nil
}

func (s *Container) TrafficGRPCServer() (*TrafficGRPCServer, error) {
	if s.trafficGrpcServer == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		traf, err := s.Traffic()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		userExtractor, err := s.UserExtractor()
		if err != nil {
			return nil, err
		}

		s.trafficGrpcServer = NewTrafficGRPCServer(
			auth,
			db,
			s.Enforcer(),
			userExtractor,
			traf,
		)
	}

	return s.trafficGrpcServer, nil
}

func (s *Container) UsersGRPCServer() (*UsersGRPCServer, error) {
	if s.usersGrpcServer == nil {
		cfg := s.Config()

		contactsRepository, err := s.ContactsRepository()
		if err != nil {
			return nil, err
		}

		userRepository, err := s.UsersRepository()
		if err != nil {
			return nil, err
		}

		events, err := s.Events()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		userExtractor, err := s.UserExtractor()
		if err != nil {
			return nil, err
		}

		s.usersGrpcServer = NewUsersGRPCServer(
			auth,
			s.Enforcer(),
			contactsRepository,
			userRepository,
			events,
			cfg.Languages,
			cfg.Captcha,
			userExtractor,
		)
	}

	return s.usersGrpcServer, nil
}

func (s *Container) ItemsGRPCServer() (*ItemsGRPCServer, error) {
	if s.itemsGrpcServer == nil {
		r, err := s.ItemsRepository()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		s.itemsGrpcServer = NewItemsGRPCServer(r, s.Memcached(), auth, s.Enforcer())
	}

	return s.itemsGrpcServer, nil
}

func (s *Container) CommentsGRPCServer() (*CommentsGRPCServer, error) {
	if s.commentsGrpcServer == nil {
		commentsRepository, err := s.CommentsRepository()
		if err != nil {
			return nil, err
		}

		usersRepository, err := s.UsersRepository()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		userExtractor, err := s.UserExtractor()
		if err != nil {
			return nil, err
		}

		s.commentsGrpcServer = NewCommentsGRPCServer(
			auth,
			commentsRepository,
			usersRepository,
			userExtractor,
			s.Enforcer(),
		)
	}

	return s.commentsGrpcServer, nil
}

func (s *Container) ContactsGRPCServer() (*ContactsGRPCServer, error) {
	if s.contactsGrpcServer == nil {
		contactsRepository, err := s.ContactsRepository()
		if err != nil {
			return nil, err
		}

		userRepository, err := s.UsersRepository()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		userExtractor, err := s.UserExtractor()
		if err != nil {
			return nil, err
		}

		s.contactsGrpcServer = NewContactsGRPCServer(
			auth,
			contactsRepository,
			userRepository,
			userExtractor,
		)
	}

	return s.contactsGrpcServer, nil
}

func (s *Container) PicturesGRPCServer() (*PicturesGRPCServer, error) {
	if s.picturesGrpcServer == nil {
		repository, err := s.PicturesRepository()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		s.picturesGrpcServer = NewPicturesGRPCServer(repository, auth, s.Enforcer())
	}

	return s.picturesGrpcServer, nil
}

func (s *Container) MapGRPCServer() (*MapGRPCServer, error) {
	if s.mapGrpcServer == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		imageStorage, err := s.ImageStorage()
		if err != nil {
			return nil, err
		}

		s.mapGrpcServer = NewMapGRPCServer(db, imageStorage)
	}

	return s.mapGrpcServer, nil
}

func (s *Container) DonationsGRPCServer() (*DonationsGRPCServer, error) {
	if s.donationsGrpcServer == nil {
		repository, err := s.ItemOfDayRepository()
		if err != nil {
			return nil, err
		}

		s.donationsGrpcServer = NewDonationsGRPCServer(repository, s.Config().DonationsVodPrice)
	}

	return s.donationsGrpcServer, nil
}

func (s *Container) MessagingGRPCServer() (*MessagingGRPCServer, error) {
	if s.messagingGrpcServer == nil {
		repository, err := s.MessagingRepository()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		s.messagingGrpcServer = NewMessagingGRPCServer(repository, auth)
	}

	return s.messagingGrpcServer, nil
}

func (s *Container) Forums() (*Forums, error) {
	if s.forums == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.forums = NewForums(db)
	}

	return s.forums, nil
}

func (s *Container) ItemOfDayRepository() (*itemofday.Repository, error) {
	if s.itemOfDayRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.itemOfDayRepository = itemofday.NewRepository(db)
	}

	return s.itemOfDayRepository, nil
}

func (s *Container) MessagingRepository() (*messaging.Repository, error) {
	if s.messagingRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		tg, err := s.TelegramService()
		if err != nil {
			return nil, err
		}

		s.messagingRepository = messaging.NewRepository(db, tg)
	}

	return s.messagingRepository, nil
}

func (s *Container) Keycloak() gocloak.GoCloak {
	if s.keyCloak == nil {
		client := gocloak.NewClient(s.Config().Keycloak.URL)

		s.keyCloak = client
	}

	return s.keyCloak
}

func (s *Container) EmailSender() email.Sender {
	if s.emailSender == nil {
		cfg := s.Config()

		if s.config.MockEmailSender {
			s.emailSender = &email.MockSender{}
		} else {
			s.emailSender = &email.SMTPSender{Config: cfg.SMTP}
		}
	}

	return s.emailSender
}

func (s *Container) SetEmailSender(emailSender email.Sender) {
	s.emailSender = emailSender
}

func (s *Container) Events() (*Events, error) {
	if s.events == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.events = NewEvents(db)
	}

	return s.events, nil
}

func (s *Container) ImageStorage() (*storage.Storage, error) {
	if s.imageStorage == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		imageStorage, err := storage.NewStorage(db, s.Config().ImageStorage)
		if err != nil {
			return nil, err
		}

		s.imageStorage = imageStorage
	}

	return s.imageStorage, nil
}

func (s *Container) Memcached() *memcache.Client {
	if s.memcached == nil {
		s.memcached = memcache.New(s.Config().Memcached...)
	}

	return s.memcached
}
