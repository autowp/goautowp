package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/autowp/goautowp/util"
	"github.com/urfave/cli/v2"

	"github.com/autowp/goautowp"
	"github.com/autowp/goautowp/config"
	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
	"gopkg.in/gographics/imagick.v2/imagick"
)

const sentryFlushTime = time.Second * 5

var autowpApp *goautowp.Application

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

	autowpApp = goautowp.NewApplication(cfg)
	defer util.Close(autowpApp)

	app := &cli.App{
		Name:        "goautowp",
		Description: "autowp cli interface",
		Commands: []*cli.Command{
			{
				Name: "image-storage",
				Subcommands: []*cli.Command{
					{
						Name: "get-image",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "image_id",
								Value:    "english",
								Usage:    "Image ID",
								Required: true,
							},
						},
						Action: func(cCtx *cli.Context) error {
							i, err := autowpApp.ImageStorageGetImage(cCtx.Context, cCtx.Int("image_id"))
							if err != nil {
								return err
							}

							logrus.Printf("%v", i)

							return nil
						},
					},
					{
						Name: "get-formatted-image",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "image_id",
								Usage:    "Image ID",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "format",
								Usage:    "Format",
								Required: true,
							},
						},
						Action: func(cCtx *cli.Context) error {
							i, err := autowpApp.ImageStorageGetFormattedImage(
								cCtx.Context,
								cCtx.Int("image_id"),
								cCtx.String("format"),
							)
							if err != nil {
								return err
							}

							logrus.Printf("%v", i)

							return nil
						},
					},
				},
			},
			{
				Name: "autoban",
				Action: func(cCtx *cli.Context) error {
					quit := captureOsInterrupt()

					return autowpApp.Autoban(quit)
				},
			},
			{
				Name: "listen-df-amqp",
				Action: func(cCtx *cli.Context) error {
					quit := captureOsInterrupt()

					return autowpApp.ListenDuplicateFinderAMQP(quit)
				},
			},
			{
				Name: "listen-monitoring-amqp",
				Action: func(cCtx *cli.Context) error {
					quit := captureOsInterrupt()

					return autowpApp.ListenMonitoringAMQP(quit)
				},
			},
			{
				Name: "serve-grpc",
				Action: func(cCtx *cli.Context) error {
					err := autowpApp.MigrateAutowp()
					if err != nil {
						return err
					}

					err = autowpApp.MigratePostgres()
					if err != nil {
						return err
					}

					quit := captureOsInterrupt()

					return autowpApp.ServeGRPC(quit)
				},
			},
			{
				Name: "serve-public",
				Action: func(cCtx *cli.Context) error {
					err := autowpApp.MigrateAutowp()
					if err != nil {
						return err
					}

					err = autowpApp.MigratePostgres()
					if err != nil {
						return err
					}

					quit := captureOsInterrupt()

					return autowpApp.ServePublic(quit)
				},
			},
			{
				Name: "serve-private",
				Action: func(cCtx *cli.Context) error {
					quit := captureOsInterrupt()

					return autowpApp.ServePrivate(quit)
				},
			},
			{
				Name: "migrate-autowp",
				Action: func(cCtx *cli.Context) error {
					return autowpApp.MigrateAutowp()
				},
			},
			{
				Name: "migrate-postgres",
				Action: func(cCtx *cli.Context) error {
					return autowpApp.MigratePostgres()
				},
			},
			{
				Name: "scheduler-hourly",
				Action: func(cCtx *cli.Context) error {
					return autowpApp.SchedulerHourly(cCtx.Context)
				},
			},
			{
				Name: "scheduler-daily",
				Action: func(cCtx *cli.Context) error {
					return autowpApp.SchedulerDaily(cCtx.Context)
				},
			},
			{
				Name: "scheduler-midnight",
				Action: func(cCtx *cli.Context) error {
					return autowpApp.SchedulerMidnight(cCtx.Context)
				},
			},
			{
				Name: "export-users-to-keycloak",
				Action: func(cCtx *cli.Context) error {
					return autowpApp.ExportUsersToKeycloak(cCtx.Context)
				},
			},
		},
	}

	if err = app.Run(os.Args); err != nil {
		logrus.Fatal(err)
		sentry.CaptureException(err)

		return 1
	}

	return 0
}
