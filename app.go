package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/autowp/goautowp/util"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	sq "github.com/Masterminds/squirrel"
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
type Application struct {
	container *Container
}

// NewApplication constructor
func NewApplication(config Config) (*Application, error) {

	s := &Application{
		container: NewContainer(config),
	}

	gin.SetMode(config.GinMode)

	return s, nil
}

func (s *Application) MigrateAutowp() error {
	_, err := s.container.GetAutowpDB()
	if err != nil {
		return err
	}

	config, err := s.container.GetConfig()
	if err != nil {
		return err
	}

	err = applyAutowpMigrations(config.AutowpMigrations)
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func (s *Application) ServePublic(quit chan bool) error {

	httpServer, err := s.container.GetPublicHttpServer()
	if err != nil {
		return err
	}

	go func() {
		<-quit
		err := httpServer.Shutdown(context.Background())
		if err != nil {
			log.Println(err.Error())
		}
	}()

	log.Println("public HTTP listener started")

	err = httpServer.ListenAndServe()
	if err != nil {
		// cannot panic, because this probably is an intentional close
		log.Printf("Httpserver: ListenAndServe() error: %s", err)
	}

	log.Println("public HTTP listener stopped")

	return nil
}

func (s *Application) ListenDuplicateFinderAMQP(quit chan bool) error {

	df, err := s.container.GetDuplicateFinder()
	if err != nil {
		return err
	}

	config, err := s.container.GetConfig()
	if err != nil {
		return err
	}

	log.Println("DuplicateFinder listener started")
	err = df.ListenAMQP(config.DuplicateFinder.RabbitMQ, config.DuplicateFinder.Queue, quit)
	if err != nil {
		log.Println(err.Error())
		sentry.CaptureException(err)
		return err
	}
	log.Println("DuplicateFinder listener stopped")

	return nil
}

// Close Destructor
func (s *Application) Close() error {
	log.Println("Closing service")

	err := s.container.Close()
	if err != nil {
		return err
	}

	log.Println("Service closed")
	return nil
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

func applyTrafficMigrations(config MigrationsConfig) error {
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

func (s *Application) MigrateTraffic() error {
	_, err := s.container.GetTrafficDB()
	if err != nil {
		return err
	}

	config, err := s.container.GetConfig()
	if err != nil {
		return err
	}

	err = applyTrafficMigrations(config.TrafficMigrations)
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func (s *Application) ServePrivate(quit chan bool) error {

	httpServer, err := s.container.GetPrivateHttpServer()
	if err != nil {
		return err
	}

	go func() {
		<-quit
		err := httpServer.Shutdown(context.Background())
		if err != nil {
			log.Println(err.Error())
		}
	}()

	log.Println("HTTP server started")
	err = httpServer.ListenAndServe()
	if err != nil {
		// cannot panic, because this probably is an intentional close
		log.Printf("Httpserver: ListenAndServe() error: %s", err)
	}
	log.Println("HTTP server stopped")

	return nil
}

func (s *Application) SchedulerHourly() error {
	traffic, err := s.container.GetTraffic()
	if err != nil {
		return err
	}

	deleted, err := traffic.Monitoring.GC()
	if err != nil {
		log.Println(err.Error())
		return err
	}
	log.Printf("`%v` items of monitoring deleted\n", deleted)

	deleted, err = traffic.Ban.GC()
	if err != nil {
		log.Println(err.Error())
		return err
	}
	log.Printf("`%v` items of ban deleted\n", deleted)

	err = traffic.AutoWhitelist()
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func (s *Application) Autoban(quit chan bool) error {

	traffic, err := s.container.GetTraffic()
	if err != nil {
		return err
	}

	banTicker := time.NewTicker(time.Minute)
	log.Println("AutoBan scheduler started")
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

	log.Println("AutoBan scheduler stopped")

	return nil
}

func (s *Application) ListenMonitoringAMQP(quit chan bool) error {
	traffic, err := s.container.GetTraffic()
	if err != nil {
		return err
	}

	config, err := s.container.GetConfig()
	if err != nil {
		return err
	}

	log.Println("Monitoring listener started")
	err = traffic.Monitoring.Listen(config.RabbitMQ, config.MonitoringQueue, quit)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	log.Println("Monitoring listener stopped")

	return nil
}
