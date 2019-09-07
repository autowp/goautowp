package goautowp

import (
	"context"
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
func NewService(wg *sync.WaitGroup, config Config) (*Service, error) {

	var err error

	logger := util.NewLogger(config.Rollbar)

	loc, err := time.LoadLocation("UTC")
	if err != nil {
		return nil, err
	}

	db, err := connectDb(config.DSN)
	if err != nil {
		fmt.Println(err)
		logger.Fatal(err)
		return nil, err
	}

	err = applyMigrations(config.Migrations)
	if err != nil && err != migrate.ErrNoChange {
		fmt.Println(err)
		logger.Fatal(err)
		return nil, err
	}

	rabbitMQ, err := connectRabbitMQ(config.RabbitMQ)
	if err != nil {
		fmt.Println(err)
		logger.Fatal(err)
		return nil, err
	}

	df, err := NewDuplicateFinder(db, rabbitMQ, config.DuplicateFinderQueue, logger)
	if err != nil {
		return nil, err
	}

	df.Listen(wg)

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

func connectRabbitMQ(config string) (*amqp.Connection, error) {
	start := time.Now()
	timeout := 60 * time.Second

	log.Println("Waiting for rabbitMQ")

	var rabbitMQ *amqp.Connection
	var err error
	for {
		rabbitMQ, err = amqp.Dial(config)
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

	return rabbitMQ, nil
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
			s.logger.Warning(err)
		}
	}

	log.Println("Disconnecting RabbitMQ")
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

	s.httpServer = &http.Server{Addr: ":80", Handler: s.router}

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
