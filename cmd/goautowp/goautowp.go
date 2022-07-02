package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/autowp/goautowp"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"github.com/getsentry/sentry-go"
	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"gopkg.in/gographics/imagick.v2/imagick"
)

const sentryFlushTime = time.Second * 5

var app *goautowp.Application

type ImageStorageCommand struct {
	GetImage          ImageStorageGetImageCommand   `command:"get-image"`
	GetFormattedImage ImageStorageGetFormattedImage `command:"get-formatted-image"`
}

type ImageStorageGetImageCommand struct {
	ImageID int `long:"image_id" description:"Image ID" value-name:"image_id" required:"yes"`
}

type ImageStorageGetFormattedImage struct {
	ImageID int    `long:"image_id" description:"Image ID" value-name:"image_id" required:"yes"`
	Format  string `long:"format" description:"Format" value-name:"format" required:"yes"`
}

func (r *ImageStorageGetImageCommand) Execute(_ []string) error {
	i, err := app.ImageStorageGetImage(r.ImageID)
	if err != nil {
		return err
	}

	log.Printf("%v", i)

	return nil
}

func (r *ImageStorageGetFormattedImage) Execute(_ []string) error {
	i, err := app.ImageStorageGetFormattedImage(r.ImageID, r.Format)
	if err != nil {
		return err
	}

	log.Printf("%v", i)

	return nil
}

type AutobanCommand struct{}

func (r *AutobanCommand) Execute(_ []string) error {
	quit := captureOsInterrupt()

	return app.Autoban(quit)
}

type ListenDfAMQPCommand struct{}

func (r *ListenDfAMQPCommand) Execute(_ []string) error {
	quit := captureOsInterrupt()

	return app.ListenDuplicateFinderAMQP(quit)
}

type ListenMonitoringAMQPCommand struct{}

func (r *ListenMonitoringAMQPCommand) Execute(_ []string) error {
	quit := captureOsInterrupt()

	return app.ListenMonitoringAMQP(quit)
}

type ServePublicCommand struct{}

func (r *ServePublicCommand) Execute(_ []string) error {
	err := app.MigrateAutowp()
	if err != nil {
		return err
	}

	err = app.MigratePostgres()
	if err != nil {
		return err
	}

	quit := captureOsInterrupt()

	return app.ServePublic(quit)
}

type ServePrivateCommand struct{}

func (r *ServePrivateCommand) Execute(_ []string) error {
	quit := captureOsInterrupt()

	return app.ServePrivate(quit)
}

type MigrateAutowpCommand struct{}

func (r *MigrateAutowpCommand) Execute(_ []string) error {
	return app.MigrateAutowp()
}

type MigratePostgresCommand struct{}

func (r *MigratePostgresCommand) Execute(_ []string) error {
	return app.MigratePostgres()
}

type SchedulerHourlyCommand struct{}

func (r *SchedulerHourlyCommand) Execute(_ []string) error {
	return app.SchedulerHourly()
}

type SchedulerDailyCommand struct{}

func (r *SchedulerDailyCommand) Execute(_ []string) error {
	return app.SchedulerDaily()
}

type SchedulerMidnightCommand struct{}

func (r *SchedulerMidnightCommand) Execute(_ []string) error {
	return app.SchedulerMidnight()
}

type ExportUsersToKeycloakCommand struct{}

func (r *ExportUsersToKeycloakCommand) Execute(_ []string) error {
	return app.ExportUsersToKeycloak()
}

func captureOsInterrupt() chan bool {
	quit := make(chan bool)

	go func() {
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		for sig := range c {
			logrus.Infof("captured %v, stopping and exiting.", sig)

			quit <- true
			close(quit)

			break
		}
	}()

	return quit
}

func main() { os.Exit(mainReturnWithCode()) }

func mainReturnWithCode() int {
	logrus.SetLevel(logrus.DebugLevel)

	imagick.Initialize()
	defer imagick.Terminate()

	cfg := config.LoadConfig(".")

	config.ValidateConfig(cfg)

	err := sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.Sentry.DSN,
		Environment: cfg.Sentry.Environment,
	})
	if err != nil {
		logrus.Error(err)

		return 1
	}

	defer func() {
		sentry.Flush(sentryFlushTime)
	}()

	app = goautowp.NewApplication(cfg)
	defer util.Close(app)

	var opts struct {
		ImageStorage          ImageStorageCommand          `command:"image-storage"`
		Autoban               AutobanCommand               `command:"autoban"`
		ListenDfAMQP          ListenDfAMQPCommand          `command:"listen-df-amqp"`
		ListenMonitoringAMQP  ListenMonitoringAMQPCommand  `command:"listen-monitoring-amqp"`
		ServePublic           ServePublicCommand           `command:"serve-public"`
		ServePrivate          ServePrivateCommand          `command:"serve-private"`
		MigrateAutowp         MigrateAutowpCommand         `command:"migrate-autowp"`
		MigratePostgres       MigratePostgresCommand       `command:"migrate-postgres"`
		SchedulerHourly       SchedulerHourlyCommand       `command:"scheduler-hourly"`
		SchedulerDaily        SchedulerDailyCommand        `command:"scheduler-daily"`
		SchedulerMidnight     SchedulerMidnightCommand     `command:"scheduler-midnight"`
		ExportUsersToKeycloak ExportUsersToKeycloakCommand `command:"export-users-to-keycloak"`
	}

	parser := flags.NewParser(&opts, 0)
	parser.WriteHelp(os.Stdout)
	_, err = parser.Parse()

	// args, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		logrus.Error(err)

		return 1
	}

	if err != nil {
		logrus.Error(err)
		sentry.CaptureException(err)

		return 1
	}

	return 0
}
