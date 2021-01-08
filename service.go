package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/autowp/goautowp/util"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/casbin/casbin"
	"github.com/dgrijalva/jwt-go"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql" // enable mysql driver
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"    // enable mysql migrations
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // enable postgres migrations
	_ "github.com/golang-migrate/migrate/v4/source/file"       // enable file migration source
)

// Service Main Object
type Service struct {
	config            Config
	autowpDB          *sql.DB
	Loc               *time.Location
	waitGroup         *sync.WaitGroup
	publicHttpServer  *http.Server
	publicRouter      *gin.Engine
	enforcer          *casbin.Enforcer
	comments          *Comments
	catalogue         *Catalogue
	acl               *ACL
	privateRouter     *gin.Engine
	privateHttpServer *http.Server
	Traffic           *Traffic
	trafficDB         *pgxpool.Pool
}

// NewService constructor
func NewService(wg *sync.WaitGroup, config Config) (*Service, error) {

	loc, err := time.LoadLocation("UTC")
	if err != nil {
		return nil, err
	}

	s := &Service{
		config:    config,
		autowpDB:  nil,
		Loc:       loc,
		waitGroup: wg,
		enforcer:  nil,
		comments:  nil,
		catalogue: nil,
		acl:       nil,
	}

	return s, nil
}

func (s *Service) getEnforcer() (*casbin.Enforcer, error) {
	if s.enforcer == nil {
		s.enforcer = casbin.NewEnforcer("model.conf", "policy.csv")
	}

	return s.enforcer, nil

}

func (s *Service) getCatalogue() (*Catalogue, error) {
	if s.catalogue == nil {
		db, err := s.getAutowpDB()
		if err != nil {
			return nil, err
		}

		enforcer, err := s.getEnforcer()
		if err != nil {
			return nil, err
		}

		s.catalogue, err = NewCatalogue(db, enforcer, s.config.FileStorage, s.config.OAuth)
		if err != nil {
			return nil, err
		}
	}

	return s.catalogue, nil
}

func (s *Service) getComments() (*Comments, error) {
	if s.comments == nil {
		db, err := s.getAutowpDB()
		if err != nil {
			return nil, err
		}

		enforcer, err := s.getEnforcer()
		if err != nil {
			return nil, err
		}

		s.comments = NewComments(db, enforcer)
	}

	return s.comments, nil
}

func (s *Service) getACL() (*ACL, error) {
	if s.acl == nil {
		db, err := s.getAutowpDB()
		if err != nil {
			return nil, err
		}

		enforcer, err := s.getEnforcer()
		if err != nil {
			return nil, err
		}

		s.acl = NewACL(db, enforcer, s.config.OAuth)
	}

	return s.acl, nil
}

func (s *Service) getDuplicateFinder() (*DuplicateFinder, error) {
	db, err := s.getAutowpDB()
	if err != nil {
		return nil, err
	}

	return NewDuplicateFinder(db)
}

func (s *Service) getAutowpDB() (*sql.DB, error) {
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

func (s *Service) MigrateAutowp() error {
	_, err := s.getAutowpDB()
	if err != nil {
		return err
	}

	err = applyAutowpMigrations(s.config.AutowpMigrations)
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func (s *Service) ServePublic() error {
	gin.SetMode(s.config.PublicRest.Mode)

	r, err := s.GetPublicRouter()
	if err != nil {
		return err
	}

	s.publicRouter = r

	s.publicHttpServer = &http.Server{Addr: s.config.PublicRest.Listen, Handler: s.publicRouter}

	s.waitGroup.Add(1)
	go func() {
		defer s.waitGroup.Done()
		log.Println("public HTTP listener started")

		err := s.publicHttpServer.ListenAndServe()
		if err != nil {
			// cannot panic, because this probably is an intentional close
			log.Printf("Httpserver: ListenAndServe() error: %s", err)
		}

		log.Println("public HTTP listener stopped")
	}()

	return nil
}

func (s *Service) ListenDuplicateFinderAMQP(quit chan bool) error {

	df, err := s.getDuplicateFinder()
	if err != nil {
		return err
	}

	s.waitGroup.Add(1)
	go func() {
		defer s.waitGroup.Done()
		fmt.Println("DuplicateFinder listener started")
		err := df.ListenAMQP(s.config.DuplicateFinder.RabbitMQ, s.config.DuplicateFinder.Queue, quit)
		if err != nil {
			log.Println(err.Error())
			sentry.CaptureException(err)
		}
		fmt.Println("DuplicateFinder listener stopped")
	}()

	return nil
}

// Close Destructor
func (s *Service) Close() {
	log.Println("Closing service")

	if s.publicHttpServer != nil {
		err := s.publicHttpServer.Shutdown(context.Background())
		if err != nil {
			log.Println(err)
			panic(err) // failure/timeout shutting down the server gracefully
		}
	}
	log.Println("Closing service wait")
	s.waitGroup.Wait()
	log.Println("Disconnecting DB")
	if s.autowpDB != nil {
		err := s.autowpDB.Close()
		if err != nil {
			sentry.CaptureException(err)
		}
	}

	log.Println("Service closed")
}

func (s *Service) getTrafficDB() (*pgxpool.Pool, error) {

	if s.trafficDB != nil {
		return s.trafficDB, nil
	}

	start := time.Now()
	timeout := 60 * time.Second

	fmt.Println("Waiting for postgres")

	var pool *pgxpool.Pool
	var err error
	for {
		pool, err = pgxpool.Connect(context.Background(), s.config.TrafficDSN)
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
			fmt.Println("Started.")
			break
		}

		if time.Since(start) > timeout {
			return nil, err
		}

		fmt.Println(err)
		fmt.Print(".")
		time.Sleep(100 * time.Millisecond)
	}

	s.trafficDB = pool

	return pool, nil
}

func applyAutowpMigrations(config MigrationsConfig) error {
	log.Println("Apply migrations")

	dir := config.Dir
	if dir == "" {
		ex, err := os.Executable()
		if err != nil {
			return err
		}
		exPath := filepath.Dir(ex)
		dir = exPath + "/migrations"
	}

	m, err := migrate.New("file://"+dir, config.DSN)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
		return err
	}
	log.Println("Migrations applied")

	return nil
}

func validateAuthorization(c *gin.Context, db *sql.DB, config OAuthConfig) (int, string, error) {
	const bearerSchema = "Bearer"
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) <= len(bearerSchema) {
		return 0, "", fmt.Errorf("authorization header is required")
	}
	tokenString := authHeader[len(bearerSchema)+1:]

	if len(tokenString) <= 0 {
		return 0, "", fmt.Errorf("authorization header is invalid")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, isValid := token.Method.(*jwt.SigningMethodHMAC); !isValid {
			return nil, fmt.Errorf("invalid token alg %v", token.Header["alg"])

		}
		return []byte(config.Secret), nil
	})

	if err != nil {
		return 0, "", err
	}

	claims := token.Claims.(jwt.MapClaims)
	idStr := claims["sub"].(string)

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, "", err
	}

	sqSelect := sq.Select("role").From("users").Where(sq.Eq{"id": id})

	rows, err := sqSelect.RunWith(db).Query()
	if err != nil {
		panic(err.Error())
	}
	defer util.Close(rows)

	if !rows.Next() {
		return 0, "", fmt.Errorf("user `%v` not found", id)
	}

	role := ""
	err = rows.Scan(&role)
	if err == sql.ErrNoRows {
		return 0, "", fmt.Errorf("user `%v` not found", id)
	}

	if err != nil {
		return 0, "", err
	}

	if role == "" {
		return 0, "", fmt.Errorf("failed role detection for `%v`", id)
	}

	return id, role, nil
}

func (s *Service) GetPublicRouter() (*gin.Engine, error) {

	r := gin.New()
	r.Use(gin.Recovery())

	catalogue, err := s.getCatalogue()
	if err != nil {
		return nil, err
	}

	comments, err := s.getComments()
	if err != nil {
		return nil, err
	}

	acl, err := s.getACL()
	if err != nil {
		return nil, err
	}

	traffic, err := s.getTraffic()
	if err != nil {
		return nil, err
	}

	goapiGroup := r.Group("/go-api")
	{
		catalogue.Routes(goapiGroup)
	}

	apiGroup := r.Group("/api")
	{
		catalogue.Routes(apiGroup)

		acl.Routes(apiGroup)

		comments.Routes(apiGroup)

		traffic.SetupPublicRouter(apiGroup)
	}

	return r, nil
}

func applyTrafficMigrations(config MigrationsConfig) error {
	fmt.Println("Apply migrations")

	dir := config.Dir
	if dir == "" {
		ex, err := os.Executable()
		if err != nil {
			return err
		}
		exPath := filepath.Dir(ex)
		dir = exPath + "/migrations"
	}

	m, err := migrate.New("file://"+dir, config.DSN)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
		return err
	}
	fmt.Println("Migrations applied")

	return nil
}

func (s *Service) MigrateTraffic() error {
	_, err := s.getTrafficDB()
	if err != nil {
		return err
	}

	err = applyTrafficMigrations(s.config.TrafficMigrations)
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func (s *Service) ServePrivate() error {

	traffic, err := s.getTraffic()
	if err != nil {
		return err
	}

	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(sentrygin.New(sentrygin.Options{}))

	traffic.SetupPrivateRouter(r)

	s.privateRouter = r

	s.privateHttpServer = &http.Server{Addr: s.config.PrivateRest.Listen, Handler: s.privateRouter}
	s.waitGroup.Add(1)
	go func() {
		defer s.waitGroup.Done()
		fmt.Println("HTTP server started")
		err := s.privateHttpServer.ListenAndServe()
		if err != nil {
			// cannot panic, because this probably is an intentional close
			log.Printf("Httpserver: ListenAndServe() error: %s", err)
		}
		fmt.Println("HTTP server stopped")
	}()

	return nil
}

func (s *Service) SchedulerHourly() error {
	traffic, err := s.getTraffic()
	if err != nil {
		return err
	}

	deleted, err := traffic.Monitoring.GC()
	if err != nil {
		log.Println(err.Error())
		return err
	}
	fmt.Printf("`%v` items of monitoring deleted\n", deleted)

	deleted, err = traffic.Ban.GC()
	if err != nil {
		log.Println(err.Error())
		return err
	}
	fmt.Printf("`%v` items of ban deleted\n", deleted)

	err = s.Traffic.AutoWhitelist()
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func (s *Service) Autoban(quit chan bool) error {

	traffic, err := s.getTraffic()
	if err != nil {
		return err
	}

	banTicker := time.NewTicker(time.Minute)
	s.waitGroup.Add(1)
	go func() {
		defer s.waitGroup.Done()
		fmt.Println("AutoBan scheduler started")
	loop:
		for {
			select {
			case <-banTicker.C:
				err := traffic.AutoBan()
				if err != nil {
					log.Println(err.Error())
				}
			case <-quit:
				banTicker.Stop()
				break loop
			}
		}

		fmt.Println("AutoBan scheduler stopped")
	}()

	return nil
}

func (s *Service) ListenMonitoringAMQP(quit chan bool) error {
	traffic, err := s.getTraffic()
	if err != nil {
		return err
	}

	s.waitGroup.Add(1)
	go func() {
		defer s.waitGroup.Done()
		fmt.Println("Monitoring listener started")
		err := traffic.Monitoring.Listen(s.config.RabbitMQ, s.config.MonitoringQueue, quit)
		if err != nil {
			log.Println(err.Error())
		}
		fmt.Println("Monitoring listener stopped")
	}()

	return nil
}

func (s *Service) getTraffic() (*Traffic, error) {
	if s.Traffic == nil {
		db, err := s.getTrafficDB()
		if err != nil {
			return nil, err
		}

		autowpDB, err := s.getAutowpDB()
		if err != nil {
			return nil, err
		}

		enforcer, err := s.getEnforcer()
		if err != nil {
			return nil, err
		}

		traffic, err := NewTraffic(db, autowpDB, enforcer, s.config.OAuth)
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}

		s.Traffic = traffic
	}

	return s.Traffic, nil
}
