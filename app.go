package goautowp

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Nerzal/gocloak/v9"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/users"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql" // enable mysql driver
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"    // enable mysql migrations
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // enable postgres migrations
	_ "github.com/golang-migrate/migrate/v4/source/file"       // enable file migration source
)

// Application is Service Main Object
type Application struct {
	container *Container
}

// NewApplication constructor
func NewApplication(cfg config.Config) *Application {

	s := &Application{
		container: NewContainer(cfg),
	}

	gin.SetMode(cfg.GinMode)

	return s
}

func (s *Application) MigrateAutowp() error {
	_, err := s.container.AutowpDB()
	if err != nil {
		return err
	}

	cfg := s.container.Config()

	err = applyMigrations(cfg.AutowpMigrations)
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func (s *Application) ServePublic(quit chan bool) error {
	httpServer, err := s.container.PublicHttpServer()
	if err != nil {
		return err
	}

	go func() {
		<-quit
		if err := httpServer.Shutdown(context.Background()); err != nil {
			logrus.Error(err.Error())
		}
	}()

	logrus.Println("public HTTP listener started")

	err = httpServer.ListenAndServe()
	if err != nil {
		// cannot panic, because this probably is an intentional close
		logrus.Printf("Httpserver: ListenAndServe() error: %s", err)
	}

	logrus.Println("public HTTP listener stopped")

	return nil
}

func (s *Application) ListenDuplicateFinderAMQP(quit chan bool) error {

	df, err := s.container.DuplicateFinder()
	if err != nil {
		return err
	}

	cfg := s.container.Config()

	logrus.Println("DuplicateFinder listener started")
	err = df.ListenAMQP(cfg.DuplicateFinder.RabbitMQ, cfg.DuplicateFinder.Queue, quit)
	if err != nil {
		logrus.Error(err.Error())
		sentry.CaptureException(err)
		return err
	}
	logrus.Println("DuplicateFinder listener stopped")

	return nil
}

// Close Destructor
func (s *Application) Close() error {
	logrus.Println("Closing service")

	err := s.container.Close()
	if err != nil {
		return err
	}

	logrus.Println("Service closed")
	return nil
}

func validateGRPCAuthorization(ctx context.Context, db *sql.DB, keycloak gocloak.GoCloak, keycloakCfg config.KeycloakConfig) (int64, string, error) {
	const bearerSchema = "Bearer"

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, "", status.Errorf(codes.InvalidArgument, "missing metadata")
	}

	lines := md["authorization"]

	if len(lines) < 1 {
		return 0, "", nil
	}

	tokenString := strings.TrimPrefix(lines[0], bearerSchema+" ")

	return validateTokenAuthorization(ctx, tokenString, db, keycloak, keycloakCfg)
}

func validateTokenAuthorization(ctx context.Context, tokenString string, db *sql.DB, keycloak gocloak.GoCloak, keycloakCfg config.KeycloakConfig) (int64, string, error) {
	if len(tokenString) <= 0 {
		return 0, "", fmt.Errorf("authorization token is invalid")
	}

	_, claims, err := keycloak.DecodeAccessToken(ctx, tokenString, keycloakCfg.Realm, "")
	if err != nil {
		return 0, "", err
	}

	guid := (*claims)["sub"].(string)

	var id int64
	role := ""
	err = db.QueryRow(`
		SELECT users.id, users.role
		FROM users
			JOIN user_account ON users.id = user_account.user_id
		WHERE user_account.external_id = ? AND user_account.service_id = ? AND not users.deleted
	`, guid, users.KeycloakExternalAccountID).Scan(&id, &role)
	if err == sql.ErrNoRows {
		return 0, "", fmt.Errorf("user `%v` not found", guid)
	}

	if err != nil {
		return 0, "", err
	}

	if role == "" {
		return 0, "", fmt.Errorf("failed role detection for `%v`", guid)
	}

	return id, role, nil
}

func applyMigrations(config config.MigrationsConfig) error {
	logrus.Info("Apply migrations")

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
	logrus.Info("Migrations applied")

	return nil
}

func (s *Application) MigrateTraffic() error {
	_, err := s.container.TrafficDB()
	if err != nil {
		return err
	}

	cfg := s.container.Config()

	err = applyMigrations(cfg.TrafficMigrations)
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func (s *Application) ServePrivate(quit chan bool) error {

	httpServer, err := s.container.PrivateHttpServer()
	if err != nil {
		return err
	}

	go func() {
		<-quit
		if err := httpServer.Shutdown(context.Background()); err != nil {
			logrus.Error(err.Error())
		}
	}()

	logrus.Info("HTTP server started")
	if err = httpServer.ListenAndServe(); err != nil {
		// cannot panic, because this probably is an intentional close
		logrus.Infof("Httpserver: ListenAndServe() error: %s", err)
	}
	logrus.Info("HTTP server stopped")

	return nil
}

func (s *Application) SchedulerHourly() error {
	traffic, err := s.container.Traffic()
	if err != nil {
		return err
	}

	deleted, err := traffic.Monitoring.GC()
	if err != nil {
		logrus.Error(err.Error())
		return err
	}
	logrus.Infof("`%v` items of monitoring deleted", deleted)

	deleted, err = traffic.Ban.GC()
	if err != nil {
		logrus.Error(err.Error())
		return err
	}
	logrus.Infof("`%v` items of ban deleted", deleted)

	err = traffic.AutoWhitelist()
	if err != nil {
		logrus.Error(err.Error())
		return err
	}

	return nil
}

func (s *Application) SchedulerDaily() error {
	usersRep, err := s.container.UsersRepository()
	if err != nil {
		return err
	}

	err = usersRep.UserRenamesGC()
	if err != nil {
		logrus.Error(err.Error())
		return err
	}

	err = usersRep.UpdateSpecsVolumes()
	if err != nil {
		logrus.Error(err.Error())
		return err
	}

	pr, err := s.container.PasswordRecovery()
	if err != nil {
		return err
	}
	count, err := pr.GC()
	if err != nil {
		logrus.Error(err.Error())
		return err
	}
	logrus.Infof("`%d` password remind rows was deleted", count)

	return nil
}

func (s *Application) SchedulerMidnight() error {
	ur, err := s.container.UsersRepository()
	if err != nil {
		return err
	}

	err = ur.RestoreVotes()
	if err != nil {
		logrus.Error(err.Error())
		return err
	}

	affected, err := ur.UpdateVotesLimits()
	if err != nil {
		logrus.Error(err.Error())
		return err
	}
	logrus.Infof("Updated %d users vote limits", affected)

	return nil
}

func (s *Application) Autoban(quit chan bool) error {

	traffic, err := s.container.Traffic()
	if err != nil {
		return err
	}

	banTicker := time.NewTicker(time.Minute)
	logrus.Info("AutoBan scheduler started")
loop:
	for {
		select {
		case <-banTicker.C:
			err := traffic.AutoBan()
			if err != nil {
				logrus.Error(err.Error())
			}
		case <-quit:
			banTicker.Stop()
			break loop
		}
	}

	logrus.Info("AutoBan scheduler stopped")

	return nil
}

func (s *Application) ExportUsersToKeycloak() error {
	ur, err := s.container.UsersRepository()
	if err != nil {
		return err
	}
	return ur.ExportUsersToKeycloak()
}

func (s *Application) ListenMonitoringAMQP(quit chan bool) error {
	traffic, err := s.container.Traffic()
	if err != nil {
		return err
	}

	cfg := s.container.Config()

	logrus.Info("Monitoring listener started")
	err = traffic.Monitoring.Listen(cfg.RabbitMQ, cfg.MonitoringQueue, quit)
	if err != nil {
		logrus.Error(err.Error())
		return err
	}
	logrus.Info("Monitoring listener stopped")

	return nil
}

func (s *Application) ImageStorageGetImage(imageID int) (*APIImage, error) {
	is, err := s.container.ImageStorage()
	if err != nil {
		return nil, err
	}
	img, err := is.Image(imageID)
	if err != nil {
		return nil, err
	}

	return ImageToAPIImage(img), nil
}

func (s *Application) ImageStorageGetFormattedImage(imageID int, format string) (*APIImage, error) {
	is, err := s.container.ImageStorage()
	if err != nil {
		return nil, err
	}
	img, err := is.FormattedImage(imageID, format)
	if err != nil {
		return nil, err
	}

	return ImageToAPIImage(img), nil
}
