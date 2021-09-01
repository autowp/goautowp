package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Nerzal/gocloak/v8"
	"github.com/casbin/casbin"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/grpc"
	"log"
	"net/http"
	"time"
)

// Container Container
type Container struct {
	autowpDB           *sql.DB
	banRepository      *BanRepository
	catalogue          *Catalogue
	comments           *Comments
	config             Config
	contactsRepository *ContactsRepository
	duplicateFinder    *DuplicateFinder
	enforcer           *casbin.Enforcer
	feedback           *Feedback
	grpcServer         *GRPCServer
	location           *time.Location
	privateHttpServer  *http.Server
	privateRouter      *gin.Engine
	publicHttpServer   *http.Server
	publicRouter       http.HandlerFunc
	traffic            *Traffic
	trafficDB          *pgxpool.Pool
	userRepository     *UserRepository
	forums             *Forums
	messages           *Messages
	keyCloak           gocloak.GoCloak
	passwordRecovery   *PasswordRecovery
	emailSender        EmailSender
	events             *Events
	usersGrpcServer    *UsersGRPCServer
}

// NewContainer constructor
func NewContainer(config Config) *Container {
	return &Container{
		config: config,
	}
}

func (s *Container) Close() error {
	s.banRepository = nil
	s.catalogue = nil
	s.comments = nil
	s.contactsRepository = nil
	s.duplicateFinder = nil
	s.traffic = nil
	s.userRepository = nil
	s.feedback = nil

	if s.autowpDB != nil {
		err := s.autowpDB.Close()
		if err != nil {
			log.Println(err.Error())
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

func (s *Container) GetAutowpDB() (*sql.DB, error) {
	if s.autowpDB != nil {
		return s.autowpDB, nil
	}

	start := time.Now()
	timeout := 60 * time.Second

	log.Println("Waiting for mysql")

	var db *sql.DB
	var err error
	for {
		db, err = sql.Open("mysql", s.config.AutowpDSN)
		if err != nil {
			return nil, err
		}

		err = db.Ping()
		if err == nil {
			log.Println("Started.")
			break
		}

		if time.Since(start) > timeout {
			return nil, err
		}

		fmt.Print(".")
		time.Sleep(100 * time.Millisecond)
	}

	s.autowpDB = db

	return s.autowpDB, nil
}

func (s *Container) GetBanRepository() (*BanRepository, error) {
	if s.banRepository == nil {
		db, err := s.GetTrafficDB()
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

func (s *Container) GetCatalogue() (*Catalogue, error) {
	if s.catalogue == nil {
		db, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		enforcer := s.GetEnforcer()
		config := s.GetConfig()

		s.catalogue, err = NewCatalogue(db, enforcer, config.OAuth)
		if err != nil {
			return nil, err
		}
	}

	return s.catalogue, nil
}

func (s *Container) GetComments() (*Comments, error) {
	if s.comments == nil {
		db, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		extractor := s.GetUserExtractor()

		s.comments = NewComments(db, extractor)
	}

	return s.comments, nil
}

func (s *Container) GetConfig() Config {
	return s.config
}

func (s *Container) GetContactsRepository() (*ContactsRepository, error) {
	if s.contactsRepository == nil {
		db, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		s.contactsRepository = NewContactsRepository(db)
	}

	return s.contactsRepository, nil
}

func (s *Container) GetDuplicateFinder() (*DuplicateFinder, error) {
	if s.duplicateFinder == nil {
		db, err := s.GetAutowpDB()
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

func (s *Container) GetEnforcer() *casbin.Enforcer {
	if s.enforcer == nil {
		s.enforcer = casbin.NewEnforcer("model.conf", "policy.csv")
	}

	return s.enforcer
}

func (s *Container) GetFeedback() (*Feedback, error) {
	if s.feedback == nil {

		config := s.GetConfig()

		emailSender := s.GetEmailSender()

		var err error
		s.feedback, err = NewFeedback(config.Feedback, config.Recaptcha, config.Captcha, emailSender)
		if err != nil {
			return nil, err
		}
	}

	return s.feedback, nil
}

func (s *Container) GetIPExtractor() *IPExtractor {
	return NewIPExtractor(s)
}

// GetLocation GetLocation
func (s *Container) GetLocation() (*time.Location, error) {
	if s.location == nil {
		loc, err := time.LoadLocation("UTC")
		if err != nil {
			return nil, err
		}

		s.location = loc
	}

	return s.location, nil
}

func (s *Container) GetPrivateHttpServer() (*http.Server, error) {
	if s.privateHttpServer == nil {
		config := s.GetConfig()

		router, err := s.GetPrivateRouter()
		if err != nil {
			return nil, err
		}

		s.privateHttpServer = &http.Server{Addr: config.PrivateRest.Listen, Handler: router}
	}

	return s.privateHttpServer, nil
}

func (s *Container) GetPrivateRouter() (*gin.Engine, error) {
	if s.privateRouter != nil {
		return s.privateRouter, nil
	}

	traffic, err := s.GetTraffic()
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

func (s *Container) GetPublicHttpServer() (*http.Server, error) {
	if s.publicHttpServer == nil {
		config := s.GetConfig()

		r, err := s.GetPublicRouter()
		if err != nil {
			return nil, err
		}

		s.publicHttpServer = &http.Server{Addr: config.PublicRest.Listen, Handler: r}
	}

	return s.publicHttpServer, nil
}

func (s *Container) GetPublicRouter() (http.HandlerFunc, error) {

	if s.publicRouter != nil {
		return s.publicRouter, nil
	}

	srv, err := s.GetGRPCServer()
	if err != nil {
		return nil, err
	}

	usersSrv, err := s.GetUsersGRPCServer()
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	RegisterAutowpServer(grpcServer, srv)
	RegisterUsersServer(grpcServer, usersSrv)

	wrappedGrpc := grpcweb.WrapServer(grpcServer)

	s.publicRouter = wrappedGrpc.ServeHTTP

	return s.publicRouter, nil
}

func (s *Container) GetTraffic() (*Traffic, error) {
	if s.traffic == nil {
		db, err := s.GetTrafficDB()
		if err != nil {
			return nil, err
		}

		autowpDB, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		ban, err := s.GetBanRepository()
		if err != nil {
			return nil, err
		}

		enforcer := s.GetEnforcer()
		config := s.GetConfig()

		userExtractor := s.GetUserExtractor()

		traffic, err := NewTraffic(db, autowpDB, enforcer, ban, userExtractor, config.OAuth)
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}

		s.traffic = traffic
	}

	return s.traffic, nil
}

func (s *Container) GetTrafficDB() (*pgxpool.Pool, error) {

	if s.trafficDB != nil {
		return s.trafficDB, nil
	}

	config := s.GetConfig()

	start := time.Now()
	timeout := 60 * time.Second

	log.Println("Waiting for postgres")

	var pool *pgxpool.Pool
	var err error
	for {
		pool, err = pgxpool.Connect(context.Background(), config.TrafficDSN)
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
			log.Println("Started.")
			break
		}

		if time.Since(start) > timeout {
			return nil, err
		}

		log.Println(err)
		fmt.Print(".")
		time.Sleep(100 * time.Millisecond)
	}

	s.trafficDB = pool

	return pool, nil
}

func (s *Container) GetUserExtractor() *UserExtractor {
	return NewUserExtractor(s)
}

func (s *Container) GetUserRepository() (*UserRepository, error) {

	if s.userRepository == nil {
		autowpDB, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		config := s.GetConfig()

		s.userRepository = NewUserRepository(
			autowpDB,
			config.UsersSalt,
			config.EmailSalt,
			config.Languages,
			s.GetEmailSender(),
			s.GetKeyCloak(),
			config.KeyCloak,
		)
	}

	return s.userRepository, nil
}

func (s *Container) GetGRPCServer() (*GRPCServer, error) {
	if s.grpcServer == nil {
		catalogue, err := s.GetCatalogue()
		if err != nil {
			return nil, err
		}

		config := s.GetConfig()

		db, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		enforcer := s.GetEnforcer()

		contactsRepository, err := s.GetContactsRepository()
		if err != nil {
			return nil, err
		}

		userRepository, err := s.GetUserRepository()
		if err != nil {
			return nil, err
		}

		userExtractor := s.GetUserExtractor()

		comments, err := s.GetComments()
		if err != nil {
			return nil, err
		}

		traffic, err := s.GetTraffic()
		if err != nil {
			return nil, err
		}

		ipExtractor := s.GetIPExtractor()

		feedback, err := s.GetFeedback()
		if err != nil {
			return nil, err
		}

		forums, err := s.GetForums()
		if err != nil {
			return nil, err
		}

		messages, err := s.GetMessages()
		if err != nil {
			return nil, err
		}

		s.grpcServer = NewGRPCServer(
			catalogue,
			config.Recaptcha,
			config.FileStorage,
			db,
			enforcer,
			config.OAuth,
			contactsRepository,
			userRepository,
			userExtractor,
			comments,
			traffic,
			ipExtractor,
			feedback,
			forums,
			messages,
		)
	}

	return s.grpcServer, nil
}

func (s *Container) GetUsersGRPCServer() (*UsersGRPCServer, error) {
	if s.usersGrpcServer == nil {
		config := s.GetConfig()

		db, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		enforcer := s.GetEnforcer()

		contactsRepository, err := s.GetContactsRepository()
		if err != nil {
			return nil, err
		}

		userRepository, err := s.GetUserRepository()
		if err != nil {
			return nil, err
		}

		events, err := s.GetEvents()
		if err != nil {
			return nil, err
		}

		pr, err := s.GetPasswordRecovery()
		if err != nil {
			return nil, err
		}

		s.usersGrpcServer = NewUsersGRPCServer(
			config.OAuth,
			db,
			enforcer,
			contactsRepository,
			userRepository,
			events,
			config.Languages,
			config.Captcha,
			pr,
		)
	}

	return s.usersGrpcServer, nil
}

func (s *Container) GetForums() (*Forums, error) {
	if s.forums == nil {
		db, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		s.forums = NewForums(db)
	}

	return s.forums, nil
}

func (s *Container) GetMessages() (*Messages, error) {
	if s.messages == nil {
		db, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		s.messages = NewMessages(db)
	}

	return s.messages, nil
}

func (s *Container) GetKeyCloak() gocloak.GoCloak {
	if s.keyCloak == nil {
		config := s.GetConfig()

		client := gocloak.NewClient(config.KeyCloak.URL)

		s.keyCloak = client
	}

	return s.keyCloak
}

func (s *Container) GetPasswordRecovery() (*PasswordRecovery, error) {
	if s.passwordRecovery == nil {
		config := s.GetConfig()

		autowpDB, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		emailSender := s.GetEmailSender()

		s.passwordRecovery = NewPasswordRecovery(autowpDB, config.Captcha, config.Languages, emailSender)
	}

	return s.passwordRecovery, nil
}

func (s *Container) GetEmailSender() EmailSender {
	if s.emailSender == nil {
		config := s.GetConfig()

		if s.config.MockEmailSender {
			s.emailSender = &MockEmailSender{}
		} else {
			s.emailSender = &SmtpEmailSender{config: config.SMTP}
		}
	}

	return s.emailSender
}

func (s *Container) SetEmailSender(emailSender EmailSender) {
	s.emailSender = emailSender
}

func (s *Container) GetEvents() (*Events, error) {
	if s.events == nil {
		autowpDB, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		s.events = NewEvents(autowpDB)
	}

	return s.events, nil
}
