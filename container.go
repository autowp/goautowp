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

		enforcer, err := s.GetEnforcer()
		if err != nil {
			return nil, err
		}

		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

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

		extractor, err := s.GetUserExtractor()
		if err != nil {
			return nil, err
		}

		s.comments = NewComments(db, extractor)
	}

	return s.comments, nil
}

func (s *Container) GetConfig() (Config, error) {
	return s.config, nil
}

func (s *Container) GetContactsRepository() (*ContactsRepository, error) {
	if s.contactsRepository == nil {
		db, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		s.contactsRepository, err = NewContactsRepository(db)
		if err != nil {
			return nil, err
		}
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

func (s *Container) GetEnforcer() (*casbin.Enforcer, error) {
	if s.enforcer == nil {
		s.enforcer = casbin.NewEnforcer("model.conf", "policy.csv")
	}

	return s.enforcer, nil
}

func (s *Container) GetFeedback() (*Feedback, error) {
	if s.feedback == nil {

		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

		emailSender, err := s.GetEmailSender()
		if err != nil {
			return nil, err
		}

		s.feedback, err = NewFeedback(config.Feedback, config.Recaptcha, config.Captcha, emailSender)
		if err != nil {
			return nil, err
		}
	}

	return s.feedback, nil
}

func (s *Container) GetIPExtractor() (*IPExtractor, error) {
	return NewIPExtractor(s), nil
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
		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

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
		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

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

	grpcServer := grpc.NewServer()
	RegisterAutowpServer(grpcServer, srv)

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

		enforcer, err := s.GetEnforcer()
		if err != nil {
			return nil, err
		}

		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

		userExtractor, err := s.GetUserExtractor()
		if err != nil {
			return nil, err
		}

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

	config, err := s.GetConfig()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	timeout := 60 * time.Second

	log.Println("Waiting for postgres")

	var pool *pgxpool.Pool
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

func (s *Container) GetUserExtractor() (*UserExtractor, error) {
	return NewUserExtractor(s), nil
}

func (s *Container) GetUserRepository() (*UserRepository, error) {

	if s.userRepository == nil {
		autowpDB, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

		keycloak, err := s.GetKeyCloak()
		if err != nil {
			return nil, err
		}

		emailSender, err := s.GetEmailSender()
		if err != nil {
			return nil, err
		}

		s.userRepository, err = NewUserRepository(
			autowpDB,
			config.UsersSalt,
			config.EmailSalt,
			config.Languages,
			emailSender,
			keycloak,
			config.KeyCloak,
		)
		if err != nil {
			return nil, err
		}
	}

	return s.userRepository, nil
}

func (s *Container) GetGRPCServer() (*GRPCServer, error) {
	if s.grpcServer == nil {
		catalogue, err := s.GetCatalogue()
		if err != nil {
			return nil, err
		}

		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

		db, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		enforcer, err := s.GetEnforcer()
		if err != nil {
			return nil, err
		}

		contactsRepository, err := s.GetContactsRepository()
		if err != nil {
			return nil, err
		}

		userRepository, err := s.GetUserRepository()
		if err != nil {
			return nil, err
		}

		userExtractor, err := s.GetUserExtractor()
		if err != nil {
			return nil, err
		}

		comments, err := s.GetComments()
		if err != nil {
			return nil, err
		}

		traffic, err := s.GetTraffic()
		if err != nil {
			return nil, err
		}

		ipExtractor, err := s.GetIPExtractor()
		if err != nil {
			return nil, err
		}

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

		s.grpcServer, err = NewGRPCServer(
			s,
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
		if err != nil {
			return nil, err
		}
	}

	return s.grpcServer, nil
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

func (s *Container) GetKeyCloak() (gocloak.GoCloak, error) {
	if s.keyCloak == nil {
		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

		client := gocloak.NewClient(config.KeyCloak.URL)

		s.keyCloak = client
	}

	return s.keyCloak, nil
}

func (s *Container) GetPasswordRecovery() (*PasswordRecovery, error) {
	if s.passwordRecovery == nil {
		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

		autowpDB, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		emailSender, err := s.GetEmailSender()
		if err != nil {
			return nil, err
		}

		s.passwordRecovery = NewPasswordRecovery(autowpDB, config.Captcha, config.Languages, emailSender)
	}

	return s.passwordRecovery, nil
}

func (s *Container) GetEmailSender() (EmailSender, error) {
	if s.emailSender == nil {
		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

		if s.config.MockEmailSender {
			s.emailSender = &MockEmailSender{}
		} else {
			s.emailSender = &SmtpEmailSender{config: config.SMTP}
		}
	}

	return s.emailSender, nil
}

func (s *Container) SetEmailSender(emailSender EmailSender) {
	s.emailSender = emailSender
}
