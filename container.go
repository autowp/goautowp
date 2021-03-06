package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/casbin/casbin"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"net/http"
	"time"
)

// Container Container
type Container struct {
	acl                 *ACL
	autowpDB            *sql.DB
	banRepository       *BanRepository
	catalogue           *Catalogue
	comments            *Comments
	config              Config
	contactsController  *ContactsController
	contactsRepository  *ContactsRepository
	duplicateFinder     *DuplicateFinder
	enforcer            *casbin.Enforcer
	feedbackController  *FeedbackController
	ipController        *IPController
	location            *time.Location
	privateHttpServer   *http.Server
	privateRouter       *gin.Engine
	publicHttpServer    *http.Server
	publicRouter        *gin.Engine
	recaptchaController *RecaptchaController
	traffic             *Traffic
	trafficDB           *pgxpool.Pool
	userRepository      *UserRepository
}

// NewContainer constructor
func NewContainer(config Config) *Container {
	return &Container{
		config: config,
	}
}

func (s *Container) Close() error {
	s.acl = nil
	s.banRepository = nil
	s.catalogue = nil
	s.comments = nil
	s.contactsController = nil
	s.contactsRepository = nil
	s.duplicateFinder = nil
	s.traffic = nil
	s.userRepository = nil
	s.feedbackController = nil

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

func (s *Container) GetACL() (*ACL, error) {
	if s.acl == nil {
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

		s.acl = NewACL(db, enforcer, config.OAuth)
	}

	return s.acl, nil
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

		s.catalogue, err = NewCatalogue(db, enforcer, config.FileStorage, config.OAuth)
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

func (s *Container) GetContactsController() (*ContactsController, error) {
	if s.contactsController == nil {
		repository, err := s.GetContactsRepository()
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

		db, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

		s.contactsController, err = NewContactsController(repository, userRepository, userExtractor, db, config.OAuth)
		if err != nil {
			return nil, err
		}
	}

	return s.contactsController, nil
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

func (s *Container) GetFeedbackController() (*FeedbackController, error) {
	if s.feedbackController == nil {

		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

		s.feedbackController, err = NewFeedbackController(config.Feedback, config.Recaptcha, config.SMTP)
		if err != nil {
			return nil, err
		}
	}

	return s.feedbackController, nil
}

func (s *Container) GetIPController() (*IPController, error) {
	if s.ipController == nil {

		autowpDB, err := s.GetAutowpDB()
		if err != nil {
			return nil, err
		}

		enforcer, err := s.GetEnforcer()
		if err != nil {
			return nil, err
		}

		ipExtractor, err := s.GetIPExtractor()
		if err != nil {
			return nil, err
		}

		banRepository, err := s.GetBanRepository()
		if err != nil {
			return nil, err
		}

		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

		s.ipController, err = NewIPController(autowpDB, enforcer, ipExtractor, banRepository, config.OAuth)
		if err != nil {
			return nil, err
		}
	}

	return s.ipController, nil
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

func (s *Container) GetPublicRouter() (*gin.Engine, error) {

	if s.publicRouter != nil {
		return s.publicRouter, nil
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(sentrygin.New(sentrygin.Options{}))

	acl, err := s.GetACL()
	if err != nil {
		return nil, err
	}

	catalogue, err := s.GetCatalogue()
	if err != nil {
		return nil, err
	}

	comments, err := s.GetComments()
	if err != nil {
		return nil, err
	}

	contactsCtrl, err := s.GetContactsController()
	if err != nil {
		return nil, err
	}

	feedbackCtrl, err := s.GetFeedbackController()
	if err != nil {
		return nil, err
	}

	ipCtrl, err := s.GetIPController()
	if err != nil {
		return nil, err
	}

	recaptchaCtrl, err := s.GetRecaptchaController()
	if err != nil {
		return nil, err
	}

	traffic, err := s.GetTraffic()
	if err != nil {
		return nil, err
	}

	apiGroup := r.Group("/api")
	{
		acl.Routes(apiGroup)
		catalogue.Routes(apiGroup)
		comments.Routes(apiGroup)
		contactsCtrl.SetupRouter(apiGroup)
		feedbackCtrl.SetupRouter(apiGroup)
		ipCtrl.SetupRouter(apiGroup)
		recaptchaCtrl.SetupRouter(apiGroup)
		traffic.SetupPublicRouter(apiGroup)
	}

	s.publicRouter = r

	return r, nil
}

func (s *Container) GetRecaptchaController() (*RecaptchaController, error) {
	if s.recaptchaController == nil {
		config, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

		s.recaptchaController, err = NewRecaptchaController(config.Recaptcha)
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}
	}

	return s.recaptchaController, nil
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

		s.userRepository, err = NewUserRepository(autowpDB)
		if err != nil {
			return nil, err
		}
	}

	return s.userRepository, nil
}
