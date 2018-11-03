package goautowp

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/autowp/goautowp/util"
	_ "github.com/go-sql-driver/mysql" // enable mysql driver
	"github.com/streadway/amqp"
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
}

// NewService constructor
func NewService(config Config) (*Service, error) {

	var err error

	logger := util.NewLogger(config.Rollbar)

	loc, _ := time.LoadLocation("UTC")

	start := time.Now()
	timeout := 60 * time.Second

	fmt.Println("Waiting for mysql")

	var db *sql.DB
	for {
		db, err = sql.Open("mysql", config.DSN)
		if err != nil {
			return nil, err
		}

		err = db.Ping()
		if err == nil {
			fmt.Println("Started.")
			break
		}

		if time.Since(start) > timeout {
			logger.Fatal(err)
			return nil, err
		}

		fmt.Print(".")
		time.Sleep(100 * time.Millisecond)
	}

	start = time.Now()
	timeout = 60 * time.Second

	fmt.Println("Waiting for rabbitMQ")

	var rabbitMQ *amqp.Connection
	for {
		rabbitMQ, err = amqp.Dial(config.RabbitMQ)
		if err == nil {
			fmt.Println("Started.")
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

	return s, nil
}

// Close Destructor
func (s *Service) Close() {
	fmt.Println("Closing service")

	s.DuplicateFinder.Close()

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

	fmt.Println("Service closed")
}
