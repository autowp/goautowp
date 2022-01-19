package goautowp

import (
	"context"
	"database/sql"
	"github.com/Nerzal/gocloak/v9"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/email"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/users"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/casbin/casbin"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpclogrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpcctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net/http"
	"time"
)

// Container Container
type Container struct {
	autowpDB           *sql.DB
	banRepository      *BanRepository
	catalogue          *Catalogue
	comments           *Comments
	config             config.Config
	contactsGrpcServer *ContactsGRPCServer
	contactsRepository *ContactsRepository
	duplicateFinder    *DuplicateFinder
	emailSender        email.Sender
	enforcer           *casbin.Enforcer
	events             *Events
	feedback           *Feedback
	forums             *Forums
	grpcServer         *GRPCServer
	imageStorage       *storage.Storage
	itemsGrpcServer    *ItemsGRPCServer
	itemsRepository    *items.Repository
	keyCloak           gocloak.GoCloak
	location           *time.Location
	messages           *Messages
	passwordRecovery   *PasswordRecovery
	privateHttpServer  *http.Server
	privateRouter      *gin.Engine
	publicHttpServer   *http.Server
	publicRouter       http.HandlerFunc
	traffic            *Traffic
	trafficDB          *pgxpool.Pool
	trafficGrpcServer  *TrafficGRPCServer
	usersRepository    *users.Repository
	usersGrpcServer    *UsersGRPCServer
	memcached          *memcache.Client
	oauth              *OAuth
	auth               *Auth
}

// NewContainer constructor
func NewContainer(cfg config.Config) *Container {
	return &Container{
		config: cfg,
	}
}

func (s *Container) Close() error {
	s.banRepository = nil
	s.catalogue = nil
	s.comments = nil
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

	if s.trafficDB != nil {
		s.trafficDB.Close()
		s.trafficDB = nil
	}

	return nil
}

func (s *Container) OAuth() (*OAuth, error) {
	if s.oauth == nil {

		ur, err := s.UsersRepository()
		if err != nil {
			return nil, err
		}

		kcConfig := s.Config().Keycloak

		s.oauth = NewOAuth(kcConfig, s.Keycloak(), ur)
	}

	return s.oauth, nil
}

func (s *Container) AutowpDB() (*sql.DB, error) {
	if s.autowpDB != nil {
		return s.autowpDB, nil
	}

	start := time.Now()
	timeout := 60 * time.Second

	logrus.Info("Waiting for mysql")

	var db *sql.DB
	var err error
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

		if time.Since(start) > timeout {
			return nil, err
		}

		logrus.Info(".")
		time.Sleep(100 * time.Millisecond)
	}

	s.autowpDB = db

	return s.autowpDB, nil
}

func (s *Container) BanRepository() (*BanRepository, error) {
	if s.banRepository == nil {
		db, err := s.TrafficDB()
		if err != nil {
			return nil, err
		}

		s.banRepository, err = NewBanRepository(db)
		if err != nil {
			return nil, err
		}
	}

	return s.banRepository, nil
}

func (s *Container) Catalogue() (*Catalogue, error) {
	if s.catalogue == nil {
		db, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		enforcer := s.Enforcer()

		s.catalogue, err = NewCatalogue(db, enforcer)
		if err != nil {
			return nil, err
		}
	}

	return s.catalogue, nil
}

func (s *Container) Comments() (*Comments, error) {
	if s.comments == nil {
		db, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		extractor := s.UserExtractor()

		s.comments = NewComments(db, extractor)
	}

	return s.comments, nil
}

func (s *Container) Config() config.Config {
	return s.config
}

func (s *Container) ContactsRepository() (*ContactsRepository, error) {
	if s.contactsRepository == nil {
		db, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		s.contactsRepository = NewContactsRepository(db)
	}

	return s.contactsRepository, nil
}

func (s *Container) DuplicateFinder() (*DuplicateFinder, error) {
	if s.duplicateFinder == nil {
		db, err := s.AutowpDB()
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

func (s *Container) IPExtractor() *IPExtractor {
	return NewIPExtractor(s)
}

// Location Location
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

func (s *Container) PrivateHttpServer() (*http.Server, error) {
	if s.privateHttpServer == nil {
		cfg := s.Config()

		router, err := s.PrivateRouter()
		if err != nil {
			return nil, err
		}

		s.privateHttpServer = &http.Server{Addr: cfg.PrivateRest.Listen, Handler: router}
	}

	return s.privateHttpServer, nil
}

func (s *Container) PrivateRouter() (*gin.Engine, error) {
	if s.privateRouter != nil {
		return s.privateRouter, nil
	}

	traffic, err := s.Traffic()
	if err != nil {
		return nil, err
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(sentrygin.New(sentrygin.Options{}))

	traffic.SetupPrivateRouter(r)

	s.privateRouter = r

	return s.privateRouter, nil
}

func (s *Container) PublicHttpServer() (*http.Server, error) {
	if s.publicHttpServer == nil {
		cfg := s.Config()

		r, err := s.PublicRouter()
		if err != nil {
			return nil, err
		}

		s.publicHttpServer = &http.Server{Addr: cfg.PublicRest.Listen, Handler: r}
	}

	return s.publicHttpServer, nil
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

	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/api/oauth/token", func(c *gin.Context) {

		form := TokenForm{}
		err := c.BindJSON(&form)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		oauth, err := s.OAuth()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		}

		switch form.GrantType {
		case "refresh_token":
			jwtToken, err := oauth.TokenByRefreshToken(c, form.RefreshToken)
			if err != nil {
				if apiErr, ok := err.(*gocloak.APIError); ok {
					if apiErr.Code > 0 {
						c.String(apiErr.Code, apiErr.Message)
						return
					}
				}
				c.String(http.StatusInternalServerError, err.Error())
				return
			}

			c.JSON(http.StatusOK, jwtToken)
		case "password":
			jwtToken, userId, err := oauth.TokenByPassword(c, form.Username, form.Password)
			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}
			if userId == 0 {
				c.Status(http.StatusBadRequest)
				return
			}
			if err != nil {
				logrus.Debugf("Login `%s` to Keycloak by credentials failed: %s", form.Username, err.Error())
				if apiErr, ok := err.(*gocloak.APIError); ok {
					if apiErr.Code > 0 {
						c.String(apiErr.Code, apiErr.Message)
						return
					}
				}
				c.String(http.StatusInternalServerError, err.Error())
				return
			}

			c.JSON(http.StatusOK, jwtToken)

		default:
			c.String(http.StatusBadRequest, "Unexpected grant_type")
		}
	})

	srv, err := s.GRPCServer()
	if err != nil {
		return nil, err
	}

	trafficSrv, err := s.TrafficGRPCServer()
	if err != nil {
		return nil, err
	}

	usersSrv, err := s.UsersGRPCServer()
	if err != nil {
		return nil, err
	}

	contactsSrv, err := s.ContactsGRPCServer()
	if err != nil {
		return nil, err
	}

	itemsSrv, err := s.ItemsGRPCServer()
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
	RegisterTrafficServer(grpcServer, trafficSrv)
	RegisterUsersServer(grpcServer, usersSrv)
	RegisterContactsServer(grpcServer, contactsSrv)
	RegisterItemsServer(grpcServer, itemsSrv)

	wrappedGrpc := grpcweb.WrapServer(grpcServer)

	s.publicRouter = func(resp http.ResponseWriter, req *http.Request) {
		if wrappedGrpc.IsGrpcWebRequest(req) {
			wrappedGrpc.ServeHTTP(resp, req)
			return
		}
		// Fall back to other servers.
		r.ServeHTTP(resp, req)
	}

	return s.publicRouter, nil
}

func (s *Container) Traffic() (*Traffic, error) {
	if s.traffic == nil {
		db, err := s.TrafficDB()
		if err != nil {
			return nil, err
		}

		autowpDB, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		ban, err := s.BanRepository()
		if err != nil {
			return nil, err
		}

		enforcer := s.Enforcer()

		userExtractor := s.UserExtractor()

		traffic, err := NewTraffic(db, autowpDB, enforcer, ban, userExtractor)
		if err != nil {
			logrus.Error(err.Error())
			return nil, err
		}

		s.traffic = traffic
	}

	return s.traffic, nil
}

func (s *Container) TrafficDB() (*pgxpool.Pool, error) {

	if s.trafficDB != nil {
		return s.trafficDB, nil
	}

	cfg := s.Config()

	start := time.Now()
	timeout := 60 * time.Second

	logrus.Info("Waiting for postgres")

	var pool *pgxpool.Pool
	var err error
	for {
		pool, err = pgxpool.Connect(context.Background(), cfg.TrafficDSN)
		if err != nil {
			return nil, err
		}

		db, err := pool.Acquire(context.Background())
		if err != nil {
			return nil, err
		}

		err = db.Conn().Ping(context.Background())
		db.Release()
		if err == nil {
			logrus.Info("Started.")
			break
		}

		if time.Since(start) > timeout {
			return nil, err
		}

		logrus.Error(err)
		logrus.Info(".")
		time.Sleep(100 * time.Millisecond)
	}

	s.trafficDB = pool

	return pool, nil
}

func (s *Container) UserExtractor() *UserExtractor {
	return NewUserExtractor(s)
}

func (s *Container) UsersRepository() (*users.Repository, error) {

	if s.usersRepository == nil {
		autowpDB, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		cfg := s.Config()

		s.usersRepository = users.NewRepository(
			autowpDB,
			cfg.UsersSalt,
			cfg.EmailSalt,
			cfg.Languages,
			s.EmailSender(),
			s.Keycloak(),
			cfg.Keycloak,
		)
	}

	return s.usersRepository, nil
}

func (s *Container) ItemsRepository() (*items.Repository, error) {

	if s.itemsRepository == nil {
		autowpDB, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		s.itemsRepository = items.NewRepository(
			autowpDB,
		)
	}

	return s.itemsRepository, nil
}

func (s *Container) Auth() (*Auth, error) {
	if s.auth == nil {

		cfg := s.Config()

		db, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		s.auth = NewAuth(db, s.Keycloak(), cfg.Keycloak)
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

		comments, err := s.Comments()
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

		messages, err := s.Messages()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		s.grpcServer = NewGRPCServer(
			auth,
			catalogue,
			cfg.Recaptcha,
			cfg.FileStorage,
			s.Enforcer(),
			s.UserExtractor(),
			comments,
			s.IPExtractor(),
			feedback,
			forums,
			messages,
		)
	}

	return s.grpcServer, nil
}

func (s *Container) TrafficGRPCServer() (*TrafficGRPCServer, error) {
	if s.trafficGrpcServer == nil {
		db, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		traffic, err := s.Traffic()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		s.trafficGrpcServer = NewTrafficGRPCServer(
			auth,
			db,
			s.Enforcer(),
			s.UserExtractor(),
			traffic,
		)
	}

	return s.trafficGrpcServer, nil
}

func (s *Container) UsersGRPCServer() (*UsersGRPCServer, error) {
	if s.usersGrpcServer == nil {
		cfg := s.Config()

		enforcer := s.Enforcer()

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

		pr, err := s.PasswordRecovery()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		s.usersGrpcServer = NewUsersGRPCServer(
			auth,
			enforcer,
			contactsRepository,
			userRepository,
			events,
			cfg.Languages,
			cfg.Captcha,
			pr,
			s.UserExtractor(),
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

		s.itemsGrpcServer = NewItemsGRPCServer(r, s.Memcached())
	}

	return s.itemsGrpcServer, nil
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

		s.contactsGrpcServer = NewContactsGRPCServer(
			auth,
			contactsRepository,
			userRepository,
			s.UserExtractor(),
		)
	}

	return s.contactsGrpcServer, nil
}

func (s *Container) Forums() (*Forums, error) {
	if s.forums == nil {
		db, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		s.forums = NewForums(db)
	}

	return s.forums, nil
}

func (s *Container) Messages() (*Messages, error) {
	if s.messages == nil {
		db, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		s.messages = NewMessages(db)
	}

	return s.messages, nil
}

func (s *Container) Keycloak() gocloak.GoCloak {
	if s.keyCloak == nil {
		client := gocloak.NewClient(s.Config().Keycloak.URL)

		s.keyCloak = client
	}

	return s.keyCloak
}

func (s *Container) PasswordRecovery() (*PasswordRecovery, error) {
	if s.passwordRecovery == nil {
		cfg := s.Config()

		autowpDB, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		emailSender := s.EmailSender()

		s.passwordRecovery = NewPasswordRecovery(autowpDB, cfg.Captcha, cfg.Languages, emailSender)
	}

	return s.passwordRecovery, nil
}

func (s *Container) EmailSender() email.Sender {
	if s.emailSender == nil {
		cfg := s.Config()

		if s.config.MockEmailSender {
			s.emailSender = &email.MockSender{}
		} else {
			s.emailSender = &email.SmtpSender{Config: cfg.SMTP}
		}
	}

	return s.emailSender
}

func (s *Container) SetEmailSender(emailSender email.Sender) {
	s.emailSender = emailSender
}

func (s *Container) Events() (*Events, error) {
	if s.events == nil {
		autowpDB, err := s.AutowpDB()
		if err != nil {
			return nil, err
		}

		s.events = NewEvents(autowpDB)
	}

	return s.events, nil
}

func (s *Container) ImageStorage() (*storage.Storage, error) {
	if s.imageStorage == nil {
		db, err := s.AutowpDB()
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
