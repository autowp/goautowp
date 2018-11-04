package goautowp

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/autowp/goautowp/util"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql" // enable mysql driver
	"github.com/streadway/amqp"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/mysql" // enable mysql migrations
	_ "github.com/golang-migrate/migrate/source/file"    // enable file migration source
)

// Service Main Object
type Service struct {
	config          Config
	logger          *util.Logger
	db              *sql.DB
	Loc             *time.Location
	rabbitMQ        *amqp.Connection
	waitGroup       *sync.WaitGroup
	DuplicateFinder *DuplicateFinder
	httpServer      *http.Server
	router          *gin.Engine
}

// NewService constructor
func NewService(config Config) (*Service, error) {

	var err error

	logger := util.NewLogger(config.Rollbar)

	loc, _ := time.LoadLocation("UTC")

	start := time.Now()
	timeout := 60 * time.Second

	log.Println("Waiting for mysql")

	var db *sql.DB
	for {
		db, err = sql.Open("mysql", config.DSN)
		if err != nil {
			return nil, err
		}

		err = db.Ping()
		if err == nil {
			log.Println("Started.")
			break
		}

		if time.Since(start) > timeout {
			logger.Fatal(err)
			return nil, err
		}

		fmt.Print(".")
		time.Sleep(100 * time.Millisecond)
	}

	err = applyMigrations(config.Migrations)
	if err != nil && err != migrate.ErrNoChange {
		logger.Fatal(err)
		return nil, err
	}

	start = time.Now()
	timeout = 60 * time.Second

	log.Println("Waiting for rabbitMQ")

	var rabbitMQ *amqp.Connection
	for {
		rabbitMQ, err = amqp.Dial(config.RabbitMQ)
		if err == nil {
			log.Println("Started.")
			break
		}

		if time.Since(start) > timeout {
			logger.Fatal(err)
			return nil, err
		}

		fmt.Print(".")
		time.Sleep(100 * time.Millisecond)
	}

	wg := &sync.WaitGroup{}

	df, err := NewDuplicateFinder(wg, db, rabbitMQ, config.DuplicateFinderQueue, config.ImagesDir, logger)
	if err != nil {
		return nil, err
	}

	s := &Service{
		config:          config,
		logger:          logger,
		db:              db,
		Loc:             loc,
		rabbitMQ:        rabbitMQ,
		waitGroup:       wg,
		DuplicateFinder: df,
	}

	s.setupRouter()

	s.ListenHTTP()

	return s, nil
}

// Close Destructor
func (s *Service) Close() {
	log.Println("Closing service")

	s.DuplicateFinder.Close()

	if s.httpServer != nil {
		err := s.httpServer.Shutdown(nil)
		if err != nil {
			panic(err) // failure/timeout shutting down the server gracefully
		}
	}

	s.waitGroup.Wait()

	if s.db != nil {
		err := s.db.Close()
		if err != nil {
			s.logger.Warning(err)
		}
	}

	if s.rabbitMQ != nil {
		err := s.rabbitMQ.Close()
		if err != nil {
			s.logger.Warning(err)
		}
	}

	log.Println("Service closed")
}

// ListenHTTP HTTP thread
func (s *Service) ListenHTTP() {
	s.waitGroup.Add(1)
	go func() {
		defer s.waitGroup.Done()
		log.Println("HTTP listener started")

		s.httpServer = &http.Server{Addr: ":80", Handler: s.router}
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
