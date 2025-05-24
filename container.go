package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/autowp/goautowp/attrs"
	"github.com/autowp/goautowp/ban"
	"github.com/autowp/goautowp/comments"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/email"
	"github.com/autowp/goautowp/hosts"
	"github.com/autowp/goautowp/i18nbundle"
	"github.com/autowp/goautowp/image/storage"
	"github.com/autowp/goautowp/index"
	"github.com/autowp/goautowp/itemofday"
	"github.com/autowp/goautowp/items"
	"github.com/autowp/goautowp/log"
	"github.com/autowp/goautowp/messaging"
	"github.com/autowp/goautowp/mosts"
	"github.com/autowp/goautowp/pictures"
	"github.com/autowp/goautowp/telegram"
	"github.com/autowp/goautowp/textstorage"
	"github.com/autowp/goautowp/traffic"
	"github.com/autowp/goautowp/users"
	"github.com/autowp/goautowp/util"
	"github.com/autowp/goautowp/votings"
	"github.com/doug-martin/goqu/v9"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/realip"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const readHeaderTimeout = time.Second * 30

// Container Container.
type Container struct {
	articlesGRPCServer     *ArticlesGRPCServer
	attrsRepository        *attrs.Repository
	autowpDB               *sql.DB
	autowpDBMutex          sync.Mutex
	banRepository          *ban.Repository
	catalogue              *Catalogue
	commentsRepository     *comments.Repository
	mostsRepository        *mosts.Repository
	config                 config.Config
	commentsGrpcServer     *CommentsGRPCServer
	mostsGrpcServer        *MostsGRPCServer
	contactsGrpcServer     *ContactsGRPCServer
	contactsRepository     *ContactsRepository
	duplicateFinder        *DuplicateFinder
	donationsGrpcServer    *DonationsGRPCServer
	emailSender            email.Sender
	events                 *Events
	feedback               *Feedback
	forums                 *Forums
	goquDB                 *goqu.Database
	goquPostgresDB         *goqu.Database
	grpcServer             *GRPCServer
	hostsManager           *hosts.Manager
	imageStorage           *storage.Storage
	i18n                   *i18nbundle.I18n
	itemOfDayRepository    *itemofday.Repository
	itemsGrpcServer        *ItemsGRPCServer
	ratingGrpcServer       *RatingGRPCServer
	votingsGrpcServer      *VotingsGRPCServer
	itemsRepository        *items.Repository
	keyCloak               *gocloak.GoCloak
	messagingGrpcServer    *MessagingGRPCServer
	messagingRepository    *messaging.Repository
	publicHTTPServer       *http.Server
	publicRouter           http.HandlerFunc
	grpcServerWithServices *grpc.Server
	telegramService        *telegram.Service
	textGrpcServer         *TextGRPCServer
	traffic                *traffic.Traffic
	trafficGrpcServer      *TrafficGRPCServer
	votingsRepository      *votings.Repository
	usersRepository        *users.Repository
	usersGrpcServer        *UsersGRPCServer
	redis                  *redis.Client
	auth                   *Auth
	mapGrpcServer          *MapGRPCServer
	picturesRepository     *pictures.Repository
	picturesGrpcServer     *PicturesGRPCServer
	statisticsGrpcServer   *StatisticsGRPCServer
	forumsGrpcServer       *ForumsGRPCServer
	attrsGRPCServer        *AttrsGRPCServer
	textStorageRepository  *textstorage.Repository
	yoomoneyHandler        *YoomoneyHandler
	logRepository          *log.Repository
	LogGrpcServer          *LogGRPCServer
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
	s.autowpDBMutex.Lock()
	defer s.autowpDBMutex.Unlock()

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

		logrus.Infof(". %s", err.Error())
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

		usersRepository, err := s.UsersRepository()
		if err != nil {
			return nil, err
		}

		messagingRepository, err := s.MessagingRepository()
		if err != nil {
			return nil, err
		}

		s.commentsRepository = comments.NewRepository(
			db,
			usersRepository,
			messagingRepository,
			s.HostsManager(),
		)
	}

	return s.commentsRepository, nil
}

func (s *Container) MostsRepository() (*mosts.Repository, error) {
	if s.mostsRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		itemsRepository, err := s.ItemsRepository()
		if err != nil {
			return nil, err
		}

		attrsRepository, err := s.AttrsRepository()
		if err != nil {
			return nil, err
		}

		s.mostsRepository = mosts.NewRepository(db, itemsRepository, attrsRepository)
	}

	return s.mostsRepository, nil
}

func (s *Container) Config() config.Config {
	return s.config
}

func (s *Container) AttrsRepository() (*attrs.Repository, error) {
	if s.attrsRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		i18n, err := s.I18n()
		if err != nil {
			return nil, err
		}

		itemsRepository, err := s.ItemsRepository()
		if err != nil {
			return nil, err
		}

		picturesRepository, err := s.PicturesRepository()
		if err != nil {
			return nil, err
		}

		is, err := s.ImageStorage()
		if err != nil {
			return nil, err
		}

		s.attrsRepository = attrs.NewRepository(db, i18n, itemsRepository, picturesRepository, is)
	}

	return s.attrsRepository, nil
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

		s.duplicateFinder, err = NewDuplicateFinder(db, s.Config().DuplicateFinder)
		if err != nil {
			return nil, err
		}
	}

	return s.duplicateFinder, nil
}

func (s *Container) Feedback() (*Feedback, error) {
	if s.feedback == nil {
		cfg := s.Config()

		emailSender := s.EmailSender()

		s.feedback = NewFeedback(cfg.Feedback, cfg.Recaptcha, cfg.Captcha, emailSender)
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

	return NewIPExtractor(banRepository, userRepository, userExtractor), nil
}

func (s *Container) HostsManager() *hosts.Manager {
	if s.hostsManager == nil {
		s.hostsManager = hosts.NewManager(s.Config().Languages)
	}

	return s.hostsManager
}

func (s *Container) LogRepository() (*log.Repository, error) {
	if s.logRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.logRepository = log.NewRepository(db)
	}

	return s.logRepository, nil
}

func (s *Container) PicturesRepository() (*pictures.Repository, error) {
	if s.picturesRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		is, err := s.ImageStorage()
		if err != nil {
			return nil, err
		}

		textStorageRepository, err := s.TextStorageRepository()
		if err != nil {
			return nil, err
		}

		itemsRepository, err := s.ItemsRepository()
		if err != nil {
			return nil, err
		}

		commentsRepository, err := s.CommentsRepository()
		if err != nil {
			return nil, err
		}

		cfg := s.Config()

		s.picturesRepository = pictures.NewRepository(
			db, is, textStorageRepository, itemsRepository, cfg.DuplicateFinder, commentsRepository,
		)
	}

	return s.picturesRepository, nil
}

func (s *Container) PublicHTTPServer(ctx context.Context) (*http.Server, error) {
	if s.publicHTTPServer == nil {
		cfg := s.Config()

		handler, err := s.PublicRouter(ctx)
		if err != nil {
			return nil, fmt.Errorf("PublicRouter(): %w", err)
		}

		s.publicHTTPServer = &http.Server{
			Addr:              cfg.PublicRest.Listen,
			Handler:           handler,
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

func (s *Container) ItemsREST() (*ItemsREST, error) {
	itemsRepo, err := s.ItemsRepository()
	if err != nil {
		return nil, err
	}

	auth, err := s.Auth()
	if err != nil {
		return nil, err
	}

	events, err := s.Events()
	if err != nil {
		return nil, err
	}

	return NewItemsREST(auth, itemsRepo, events), nil
}

func (s *Container) UsersREST() (*UsersREST, error) {
	usersRepo, err := s.UsersRepository()
	if err != nil {
		return nil, err
	}

	auth, err := s.Auth()
	if err != nil {
		return nil, err
	}

	return NewUsersREST(auth, usersRepo), nil
}

func (s *Container) PicturesREST() (*PicturesREST, error) {
	auth, err := s.Auth()
	if err != nil {
		return nil, err
	}

	picturesRepo, err := s.PicturesRepository()
	if err != nil {
		return nil, err
	}

	imageStorage, err := s.ImageStorage()
	if err != nil {
		return nil, err
	}

	itemOfDayRepo, err := s.ItemOfDayRepository()
	if err != nil {
		return nil, err
	}

	itemsRepo, err := s.ItemsRepository()
	if err != nil {
		return nil, err
	}

	i18nBundle, err := s.I18n()
	if err != nil {
		return nil, err
	}

	usersRepo, err := s.UsersRepository()
	if err != nil {
		return nil, err
	}

	commentsRepo, err := s.CommentsRepository()
	if err != nil {
		return nil, err
	}

	df, err := s.DuplicateFinder()
	if err != nil {
		return nil, err
	}

	ts, err := s.TelegramService()
	if err != nil {
		return nil, err
	}

	pictureNameFormatter := pictures.NewPictureNameFormatter(
		items.NewItemNameFormatter(i18nBundle),
		i18nBundle,
	)

	return NewPicturesREST(
		auth,
		picturesRepo,
		pictureNameFormatter,
		s.HostsManager(),
		imageStorage,
		itemOfDayRepo,
		itemsRepo,
		usersRepo,
		commentsRepo,
		df,
		ts,
	), nil
}

func (s *Container) PublicRouter(ctx context.Context) (http.HandlerFunc, error) {
	if s.publicRouter != nil {
		return s.publicRouter, nil
	}

	grpcServer, err := s.GRPCServerWithServices()
	if err != nil {
		return nil, fmt.Errorf("GRPCServerWithServices(): %w", err)
	}

	originFunc := func(origin string) bool {
		return util.Contains(s.config.PublicRest.Cors.Origin, origin)
	}
	wrappedGrpc := grpcweb.WrapServer(grpcServer, grpcweb.WithOriginFunc(originFunc))

	yoomoney, err := s.YoomoneyHandler()
	if err != nil {
		return nil, err
	}

	tg, err := s.TelegramService()
	if err != nil {
		return nil, fmt.Errorf("TelegramService(): %w", err)
	}

	ginEngine := gin.New()
	ginEngine.Use(gin.Recovery())

	err = ginEngine.SetTrustedProxies([]string{s.Config().TrustedNetwork})
	if err != nil {
		return nil, fmt.Errorf("SetTrustedProxies(): %w", err)
	}

	if len(s.config.PublicRest.Cors.Origin) > 0 {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = s.config.PublicRest.Cors.Origin
		corsConfig.AllowCredentials = true
		ginEngine.Use(cors.New(corsConfig))
	}

	yoomoney.SetupRouter(ctx, ginEngine)

	tg.SetupRouter(ginEngine) //nolint: contextcheck

	picturesREST, err := s.PicturesREST()
	if err != nil {
		return nil, fmt.Errorf("PicturesREST(): %w", err)
	}

	picturesREST.SetupRouter(ginEngine) //nolint: contextcheck

	itemsREST, err := s.ItemsREST()
	if err != nil {
		return nil, fmt.Errorf("ItemsREST(): %w", err)
	}

	itemsREST.SetupRouter(ginEngine) //nolint: contextcheck

	usersREST, err := s.UsersREST()
	if err != nil {
		return nil, fmt.Errorf("UsersREST(): %w", err)
	}

	usersREST.SetupRouter(ginEngine) //nolint: contextcheck

	s.publicRouter = func(resp http.ResponseWriter, req *http.Request) {
		if wrappedGrpc.IsAcceptableGrpcCorsRequest(req) || wrappedGrpc.IsGrpcWebRequest(req) {
			wrappedGrpc.ServeHTTP(resp, req)

			return
		}
		// Fall back to gRPC+h2c server
		ginEngine.ServeHTTP(resp, req)
	}

	s.grpcServerWithServices = grpcServer

	return s.publicRouter, nil
}

func (s *Container) GRPCServerWithServices() (*grpc.Server, error) {
	if s.grpcServerWithServices != nil {
		return s.grpcServerWithServices, nil
	}

	srv, err := s.GRPCServer()
	if err != nil {
		return nil, err
	}

	articlesSrv, err := s.ArticlesGRPCServer()
	if err != nil {
		return nil, err
	}

	attrsSrv, err := s.AttrsGRPCServer()
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

	forumsSrv, err := s.ForumsGRPCServer()
	if err != nil {
		return nil, err
	}

	itemsSrv, err := s.ItemsGRPCServer()
	if err != nil {
		return nil, err
	}

	logSrv, err := s.LogGRPCServer()
	if err != nil {
		return nil, err
	}

	mapSrv, err := s.MapGRPCServer()
	if err != nil {
		return nil, err
	}

	mostsSrv, err := s.MostsGRPCServer()
	if err != nil {
		return nil, err
	}

	textSrv, err := s.TextGRPCServer()
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

	ratingSrv, err := s.RatingGRPCServer()
	if err != nil {
		return nil, err
	}

	trustedPeers := []netip.Prefix{
		netip.MustParsePrefix(s.Config().TrustedNetwork),
	}

	opts := []realip.Option{
		realip.WithTrustedPeers(trustedPeers),
		realip.WithHeaders([]string{realip.XForwardedFor}),
	}

	logger := logrus.StandardLogger()
	loggerOpts := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(InterceptorLogger(logger), loggerOpts...),
			realip.UnaryServerInterceptorOpts(opts...),
		),
		grpc.ChainStreamInterceptor(
			logging.StreamServerInterceptor(InterceptorLogger(logger), loggerOpts...),
			realip.StreamServerInterceptorOpts(opts...),
		),
	)
	RegisterArticlesServer(grpcServer, articlesSrv)
	RegisterAttrsServer(grpcServer, attrsSrv)
	RegisterAutowpServer(grpcServer, srv)
	RegisterCommentsServer(grpcServer, commentsSrv)
	RegisterContactsServer(grpcServer, contactsSrv)
	RegisterDonationsServer(grpcServer, donationsSrv)
	RegisterForumsServer(grpcServer, forumsSrv)
	RegisterItemsServer(grpcServer, itemsSrv)
	RegisterLogServer(grpcServer, logSrv)
	RegisterMapServer(grpcServer, mapSrv)
	RegisterMostsServer(grpcServer, mostsSrv)
	RegisterMessagingServer(grpcServer, messagingSrv)
	RegisterPicturesServer(grpcServer, picturesSrv)
	RegisterStatisticsServer(grpcServer, statSrv)
	RegisterTextServer(grpcServer, textSrv)
	RegisterTrafficServer(grpcServer, trafficSrv)
	RegisterUsersServer(grpcServer, usersSrv)
	RegisterRatingServer(grpcServer, ratingSrv)

	reflection.Register(grpcServer)

	s.grpcServerWithServices = grpcServer

	return s.grpcServerWithServices, nil
}

func (s *Container) TelegramService() (*telegram.Service, error) {
	if s.telegramService == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		userRepository, err := s.UsersRepository()
		if err != nil {
			return nil, err
		}

		itemRepository, err := s.ItemsRepository()
		if err != nil {
			return nil, err
		}

		messagingRepository, err := s.MessagingRepository()
		if err != nil {
			return nil, err
		}

		picturesRepository, err := s.PicturesRepository()
		if err != nil {
			return nil, err
		}

		s.telegramService = telegram.NewService(
			s.Config().Telegram,
			db,
			s.HostsManager(),
			userRepository,
			itemRepository,
			messagingRepository,
			picturesRepository,
		)
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

		traf, err := traffic.NewTraffic(db, autowpDB, banRepository)
		if err != nil {
			logrus.Error(err.Error())

			return nil, err
		}

		s.traffic = traf
	}

	return s.traffic, nil
}

func (s *Container) UserExtractor() (*UserExtractor, error) {
	is, err := s.ImageStorage()
	if err != nil {
		return nil, err
	}

	picRepository, err := s.PicturesRepository()
	if err != nil {
		return nil, err
	}

	return NewUserExtractor(is, picRepository), nil
}

func (s *Container) VotingsRepository() (*votings.Repository, error) {
	if s.votingsRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.votingsRepository = votings.NewRepository(db)
	}

	return s.votingsRepository, nil
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

		is, err := s.ImageStorage()
		if err != nil {
			return nil, err
		}

		s.usersRepository = users.NewRepository(
			db,
			postgresDB,
			cfg.UsersSalt,
			cfg.Languages,
			s.Keycloak(),
			cfg.Keycloak,
			cfg.MessageInterval,
			is,
		)
	}

	return s.usersRepository, nil
}

func (s *Container) I18n() (*i18nbundle.I18n, error) {
	if s.i18n == nil {
		i, err := i18nbundle.New()
		if err != nil {
			return nil, err
		}

		s.i18n = i
	}

	return s.i18n, nil
}

func (s *Container) ItemsRepository() (*items.Repository, error) {
	if s.itemsRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		cfg := s.Config()

		textStorageRepository, err := s.TextStorageRepository()
		if err != nil {
			return nil, err
		}

		imageStorage, err := s.ImageStorage()
		if err != nil {
			return nil, err
		}

		s.itemsRepository = items.NewRepository(
			db,
			cfg.MostsMinCarsCount,
			s.Config().ContentLanguages,
			textStorageRepository,
			imageStorage,
		)
	}

	return s.itemsRepository, nil
}

func (s *Container) ItemExtractor() *ItemExtractor {
	return NewItemExtractor(s)
}

func (s *Container) PictureItemExtractor() *PictureItemExtractor {
	return NewPictureItemExtractor(s)
}

func (s *Container) PictureExtractor() *PictureExtractor {
	return NewPictureExtractor(s)
}

func (s *Container) ItemParentCacheExtractor() *ItemParentCacheExtractor {
	return NewItemParentCacheExtractor(s)
}

func (s *Container) DfDistanceExtractor() *DfDistanceExtractor {
	return NewDfDistanceExtractor(s)
}

func (s *Container) ItemParentExtractor() *ItemParentExtractor {
	return NewItemParentExtractor(s)
}

func (s *Container) NewLinkExtractor() *LinkExtractor {
	return NewLinkExtractor(s)
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
			commentsRepository,
			ipExtractor,
			feedback,
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

		s.statisticsGrpcServer = NewStatisticsGRPCServer(db, s.Config().About)
	}

	return s.statisticsGrpcServer, nil
}

func (s *Container) TextGRPCServer() (*TextGRPCServer, error) {
	if s.textGrpcServer == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.textGrpcServer = NewTextGRPCServer(db)
	}

	return s.textGrpcServer, nil
}

func (s *Container) TrafficGRPCServer() (*TrafficGRPCServer, error) {
	if s.trafficGrpcServer == nil {
		traf, err := s.Traffic()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		usersRepo, err := s.UsersRepository()
		if err != nil {
			return nil, err
		}

		userExtractor, err := s.UserExtractor()
		if err != nil {
			return nil, err
		}

		s.trafficGrpcServer = NewTrafficGRPCServer(auth, usersRepo, userExtractor, traf)
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

func (s *Container) VotingsGRPCServer() (*VotingsGRPCServer, error) {
	if s.votingsGrpcServer == nil {
		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		votingsRepo, err := s.VotingsRepository()
		if err != nil {
			return nil, err
		}

		s.votingsGrpcServer = NewVotingsGRPCServer(votingsRepo, auth)
	}

	return s.votingsGrpcServer, nil
}

func (s *Container) RatingGRPCServer() (*RatingGRPCServer, error) {
	if s.ratingGrpcServer == nil {
		commentsRepository, err := s.CommentsRepository()
		if err != nil {
			return nil, err
		}

		itemsRepository, err := s.ItemsRepository()
		if err != nil {
			return nil, err
		}

		userRepository, err := s.UsersRepository()
		if err != nil {
			return nil, err
		}

		picturesRepository, err := s.PicturesRepository()
		if err != nil {
			return nil, err
		}

		attrsRepository, err := s.AttrsRepository()
		if err != nil {
			return nil, err
		}

		s.ratingGrpcServer = NewRatingGRPCServer(
			picturesRepository,
			userRepository,
			itemsRepository,
			commentsRepository,
			attrsRepository,
		)
	}

	return s.ratingGrpcServer, nil
}

func (s *Container) ItemsGRPCServer() (*ItemsGRPCServer, error) {
	if s.itemsGrpcServer == nil {
		repo, err := s.ItemsRepository()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		textStorageRepository, err := s.TextStorageRepository()
		if err != nil {
			return nil, err
		}

		extractor := s.ItemExtractor()

		i18n, err := s.I18n()
		if err != nil {
			return nil, err
		}

		attrsRepository, err := s.AttrsRepository()
		if err != nil {
			return nil, err
		}

		picturesRepository, err := s.PicturesRepository()
		if err != nil {
			return nil, err
		}

		idx, err := s.Index()
		if err != nil {
			return nil, err
		}

		events, err := s.Events()
		if err != nil {
			return nil, err
		}

		usersRepository, err := s.UsersRepository()
		if err != nil {
			return nil, err
		}

		messagingRepository, err := s.MessagingRepository()
		if err != nil {
			return nil, err
		}

		itemOfDayRepository, err := s.ItemOfDayRepository()
		if err != nil {
			return nil, err
		}

		redisClient, err := s.Redis()
		if err != nil {
			return nil, err
		}

		s.itemsGrpcServer = NewItemsGRPCServer(
			repo,
			db,
			auth,
			s.Config().ContentLanguages,
			textStorageRepository,
			extractor,
			i18n,
			attrsRepository,
			picturesRepository,
			idx,
			events,
			usersRepository,
			messagingRepository,
			s.HostsManager(),
			s.ItemParentExtractor(),
			s.NewLinkExtractor(),
			itemOfDayRepository,
			redisClient,
		)
	}

	return s.itemsGrpcServer, nil
}

func (s *Container) MostsGRPCServer() (*MostsGRPCServer, error) {
	if s.mostsGrpcServer == nil {
		mostsRepository, err := s.MostsRepository()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		s.mostsGrpcServer = NewMostsGRPCServer(
			auth,
			s.ItemExtractor(),
			mostsRepository,
		)
	}

	return s.mostsGrpcServer, nil
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

		picturesRepository, err := s.PicturesRepository()
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
			picturesRepository,
			userExtractor,
		)
	}

	return s.commentsGrpcServer, nil
}

func (s *Container) ArticlesGRPCServer() (*ArticlesGRPCServer, error) {
	if s.articlesGRPCServer == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.articlesGRPCServer = NewArticlesGRPCServer(db)
	}

	return s.articlesGRPCServer, nil
}

func (s *Container) AttrsGRPCServer() (*AttrsGRPCServer, error) {
	if s.attrsGRPCServer == nil {
		repository, err := s.AttrsRepository()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		s.attrsGRPCServer = NewAttrsGRPCServer(repository, auth)
	}

	return s.attrsGRPCServer, nil
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

func (s *Container) LogGRPCServer() (*LogGRPCServer, error) {
	if s.LogGrpcServer == nil {
		repository, err := s.LogRepository()
		if err != nil {
			return nil, err
		}

		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		s.LogGrpcServer = NewLogGRPCServer(repository, auth)
	}

	return s.LogGrpcServer, nil
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

		events, err := s.Events()
		if err != nil {
			return nil, err
		}

		messagingRepository, err := s.MessagingRepository()
		if err != nil {
			return nil, err
		}

		userRepository, err := s.UsersRepository()
		if err != nil {
			return nil, err
		}

		duplicateFinder, err := s.DuplicateFinder()
		if err != nil {
			return nil, err
		}

		textStorageRepository, err := s.TextStorageRepository()
		if err != nil {
			return nil, err
		}

		tg, err := s.TelegramService()
		if err != nil {
			return nil, err
		}

		itemRepository, err := s.ItemsRepository()
		if err != nil {
			return nil, err
		}

		commentsRepository, err := s.CommentsRepository()
		if err != nil {
			return nil, err
		}

		s.picturesGrpcServer = NewPicturesGRPCServer(
			repository,
			auth,
			events,
			s.HostsManager(),
			messagingRepository,
			userRepository,
			duplicateFinder,
			textStorageRepository,
			tg,
			itemRepository,
			commentsRepository,
			s.PictureExtractor(),
			s.PictureItemExtractor(),
			s.ItemExtractor(),
		)
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

		i18n, err := s.I18n()
		if err != nil {
			return nil, err
		}

		s.mapGrpcServer = NewMapGRPCServer(db, imageStorage, i18n)
	}

	return s.mapGrpcServer, nil
}

func (s *Container) DonationsGRPCServer() (*DonationsGRPCServer, error) {
	if s.donationsGrpcServer == nil {
		repository, err := s.ItemOfDayRepository()
		if err != nil {
			return nil, err
		}

		db, err := s.GoquPostgresDB()
		if err != nil {
			return nil, err
		}

		s.donationsGrpcServer = NewDonationsGRPCServer(repository, s.Config().DonationsVodPrice, db)
	}

	return s.donationsGrpcServer, nil
}

func (s *Container) ForumsGRPCServer() (*ForumsGRPCServer, error) {
	if s.forumsGrpcServer == nil {
		auth, err := s.Auth()
		if err != nil {
			return nil, err
		}

		forums, err := s.Forums()
		if err != nil {
			return nil, err
		}

		commentsRepo, err := s.CommentsRepository()
		if err != nil {
			return nil, err
		}

		usersRepo, err := s.UsersRepository()
		if err != nil {
			return nil, err
		}

		s.forumsGrpcServer = NewForumsGRPCServer(auth, forums, commentsRepo, usersRepo)
	}

	return s.forumsGrpcServer, nil
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

		commentsRepository, err := s.CommentsRepository()
		if err != nil {
			return nil, err
		}

		s.forums = NewForums(db, commentsRepository)
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

		i18n, err := s.I18n()
		if err != nil {
			return nil, err
		}

		s.messagingRepository = messaging.NewRepository(
			db,
			func(ctx context.Context, fromUserID int64, toUserID int64, text string) error {
				tg, err := s.TelegramService()
				if err != nil {
					return err
				}

				return tg.NotifyMessage(ctx, fromUserID, toUserID, text)
			},
			i18n,
		)
	}

	return s.messagingRepository, nil
}

func (s *Container) Keycloak() *gocloak.GoCloak {
	if s.keyCloak == nil {
		client := gocloak.NewClient(s.Config().Keycloak.URL)

		s.keyCloak = client
	}

	return s.keyCloak
}

func (s *Container) EmailSender() email.Sender { //nolint: ireturn
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

func (s *Container) Redis() (*redis.Client, error) {
	if s.redis == nil {
		opts, err := redis.ParseURL(s.Config().Redis)
		if err != nil {
			return nil, err
		}

		s.redis = redis.NewClient(opts)
	}

	return s.redis, nil
}

func (s *Container) Index() (*index.Index, error) {
	redisClient, err := s.Redis()
	if err != nil {
		return nil, err
	}

	repository, err := s.ItemsRepository()
	if err != nil {
		return nil, err
	}

	return index.NewIndex(redisClient, repository), nil
}

func (s *Container) TextStorageRepository() (*textstorage.Repository, error) {
	if s.textStorageRepository == nil {
		db, err := s.GoquDB()
		if err != nil {
			return nil, err
		}

		s.textStorageRepository = textstorage.New(db)
	}

	return s.textStorageRepository, nil
}

func (s *Container) YoomoneyHandler() (*YoomoneyHandler, error) {
	if s.yoomoneyHandler == nil {
		repository, err := s.ItemOfDayRepository()
		if err != nil {
			return nil, err
		}

		cfg := s.Config().YoomoneyConfig

		s.yoomoneyHandler, err = NewYoomoneyHandler(cfg.Price, cfg.Secret, repository)
		if err != nil {
			return nil, err
		}
	}

	return s.yoomoneyHandler, nil
}
