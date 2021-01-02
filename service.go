package goautowp

import (
	"context"
	"database/sql"
	"fmt"
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

	sentry "github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql" // enable mysql driver
	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/mysql" // enable mysql migrations
	_ "github.com/golang-migrate/migrate/source/file"    // enable file migration source
)

// Service Main Object
type Service struct {
	config          Config
	db              *sql.DB
	Loc             *time.Location
	waitGroup       *sync.WaitGroup
	DuplicateFinder *DuplicateFinder
	httpServer      *http.Server
	router          *gin.Engine
	enforcer        *casbin.Enforcer
	comments        *Comments
	catalogue       *Catalogue
	acl             *ACL
}

// NewService constructor
func NewService(wg *sync.WaitGroup, config Config, enforcer *casbin.Enforcer) (*Service, error) {

	var err error

	loc, err := time.LoadLocation("UTC")
	if err != nil {
		return nil, err
	}

	db, err := connectDb(config.DSN)
	if err != nil {
		fmt.Println(err)
		sentry.CaptureException(err)
		return nil, err
	}

	err = applyMigrations(config.Migrations)
	if err != nil && err != migrate.ErrNoChange {
		fmt.Println(err)
		sentry.CaptureException(err)
		return nil, err
	}

	df, err := NewDuplicateFinder(db, config.DuplicateFinder)
	if err != nil {
		return nil, err
	}

	df.Listen(wg)

	s := &Service{
		config:          config,
		db:              db,
		Loc:             loc,
		waitGroup:       wg,
		DuplicateFinder: df,
		enforcer:        enforcer,
		comments:        NewComments(db, enforcer),
		catalogue:       NewCatalogue(db, enforcer, config.FileStorage, config.OAuth),
		acl:             NewACL(db, enforcer, config.OAuth),
	}

	s.setupRouter()

	s.ListenHTTP()

	return s, nil
}

func connectDb(dsn string) (*sql.DB, error) {
	start := time.Now()
	timeout := 60 * time.Second

	log.Println("Waiting for mysql")

	var db *sql.DB
	var err error
	for {
		db, err = sql.Open("mysql", dsn)
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

	return db, nil
}

// Close Destructor
func (s *Service) Close() {
	log.Println("Closing service")

	s.DuplicateFinder.Close()

	if s.httpServer != nil {
		err := s.httpServer.Shutdown(context.Background())
		if err != nil {
			log.Println(err)
			panic(err) // failure/timeout shutting down the server gracefully
		}
	}
	log.Println("Closing service wait")
	s.waitGroup.Wait()
	log.Println("Disconnecting DB")
	if s.db != nil {
		err := s.db.Close()
		if err != nil {
			sentry.CaptureException(err)
		}
	}

	log.Println("Service closed")
}

// ListenHTTP HTTP thread
func (s *Service) ListenHTTP() {

	s.httpServer = &http.Server{Addr: s.config.Rest.Listen, Handler: s.router}

	s.waitGroup.Add(1)
	go func() {
		defer s.waitGroup.Done()
		log.Println("HTTP listener started")

		err := s.httpServer.ListenAndServe()
		if err != nil {
			// cannot panic, because this probably is an intentional close
			log.Printf("Httpserver: ListenAndServe() error: %s", err)
		}

		log.Println("HTTP listener stopped")
	}()
}

func applyMigrations(config MigrationsConfig) error {
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

func validateAuthorization(c *gin.Context, db *sql.DB, config OAuthConfig) (string, error) {
	const bearerSchema = "Bearer"
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) <= len(bearerSchema) {
		return "", fmt.Errorf("authorization header is required")
	}
	tokenString := authHeader[len(bearerSchema)+1:]

	if len(tokenString) <= 0 {
		return "", fmt.Errorf("authorization header is invalid")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, isvalid := token.Method.(*jwt.SigningMethodHMAC); !isvalid {
			return nil, fmt.Errorf("invalid token alg %v", token.Header["alg"])

		}
		return []byte(config.Secret), nil
	})

	if err != nil {
		return "", err
	}

	claims := token.Claims.(jwt.MapClaims)
	idStr := claims["sub"].(string)

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return "", err
	}

	sqSelect := sq.Select("role").From("users").Where(sq.Eq{"id": id})

	rows, err := sqSelect.RunWith(db).Query()
	if err != nil {
		panic(err.Error())
	}

	if !rows.Next() {
		return "", fmt.Errorf("user `%v` not found", id)
	}

	role := ""
	err = rows.Scan(&role)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("user `%v` not found", id)
	}

	if err != nil {
		return "", err
	}

	if role == "" {
		return "", fmt.Errorf("failed role detection for `%v`", id)
	}

	return role, nil
}

func (s *Service) setupRouter() {

	gin.SetMode(s.config.Rest.Mode)

	r := gin.New()
	r.Use(gin.Recovery())

	goapiGroup := r.Group("/go-api")
	{
		s.catalogue.Routes(goapiGroup)
	}

	apiGroup := r.Group("/api")
	{
		s.catalogue.Routes(apiGroup)

		s.acl.Routes(apiGroup)

		s.comments.Routes(apiGroup)
	}

	s.router = r
}

// GetRouter GetRouter
func (s *Service) GetRouter() *gin.Engine {
	return s.router
}
