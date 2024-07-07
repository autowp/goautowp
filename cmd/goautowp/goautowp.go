package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/autowp/goautowp"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"gopkg.in/gographics/imagick.v2/imagick"
)

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

	autowpApp = goautowp.NewApplication(cfg)
	defer util.Close(autowpApp)

	app := &cli.Command{
		Name:        "goautowp",
		Description: "autowp cli interface",
		Commands: []*cli.Command{
			{
				Name: "image-storage",
				Commands: []*cli.Command{
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
						Action: func(ctx context.Context, command *cli.Command) error {
							i, err := autowpApp.ImageStorageGetImage(ctx, int(command.Int("image_id")))
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
						Action: func(ctx context.Context, command *cli.Command) error {
							img, err := autowpApp.ImageStorageGetFormattedImage(
								ctx,
								int(command.Int("image_id")),
								command.String("format"),
							)
							if err != nil {
								return err
							}

							logrus.Printf("%v", img)

							return nil
						},
					},
					{
						Name: "list-broken-images",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "dir",
								Value:    "",
								Usage:    "dir",
								Required: true,
							},
						},
						Action: func(ctx context.Context, command *cli.Command) error {
							return autowpApp.ImageStorageListBrokenImages(ctx, command.String("dir"))
						},
					},
					{
						Name: "list-unlinked-objects",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "dir",
								Value:    "",
								Usage:    "dir",
								Required: true,
							},
						},
						Action: func(ctx context.Context, command *cli.Command) error {
							return autowpApp.ImageStorageListUnlinkedObjects(ctx, command.String("dir"))
						},
					},
				},
			},
			{
				Name: "autoban",
				Action: func(ctx context.Context, _ *cli.Command) error {
					quit := captureOsInterrupt()

					return autowpApp.Autoban(ctx, quit)
				},
			},
			{
				Name: "listen-df-amqp",
				Action: func(ctx context.Context, _ *cli.Command) error {
					quit := captureOsInterrupt()

					return autowpApp.ListenDuplicateFinderAMQP(ctx, quit)
				},
			},
			{
				Name: "listen-monitoring-amqp",
				Action: func(ctx context.Context, _ *cli.Command) error {
					quit := captureOsInterrupt()

					return autowpApp.ListenMonitoringAMQP(ctx, quit)
				},
			},
			{
				Name: "serve-grpc",
				Action: func(ctx context.Context, _ *cli.Command) error {
					err := autowpApp.MigrateAutowp(ctx)
					if err != nil {
						return err
					}

					err = autowpApp.MigratePostgres(ctx)
					if err != nil {
						return err
					}

					quit := captureOsInterrupt()

					return autowpApp.ServeGRPC(quit)
				},
			},
			{
				Name: "serve-public",
				Action: func(ctx context.Context, _ *cli.Command) error {
					err := autowpApp.MigrateAutowp(ctx)
					if err != nil {
						return err
					}

					err = autowpApp.MigratePostgres(ctx)
					if err != nil {
						return err
					}

					quit := captureOsInterrupt()

					return autowpApp.ServePublic(ctx, quit)
				},
			},
			{
				Name: "serve-private",
				Action: func(ctx context.Context, _ *cli.Command) error {
					quit := captureOsInterrupt()

					return autowpApp.ServePrivate(ctx, quit)
				},
			},
			{
				Name: "migrate-autowp",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.MigrateAutowp(ctx)
				},
			},
			{
				Name: "migrate-postgres",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.MigratePostgres(ctx)
				},
			},
			{
				Name: "scheduler-hourly",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.SchedulerHourly(ctx)
				},
			},
			{
				Name: "scheduler-daily",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.SchedulerDaily(ctx)
				},
			},
			{
				Name: "scheduler-midnight",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.SchedulerMidnight(ctx)
				},
			},
			{
				Name: "export-users-to-keycloak",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.ExportUsersToKeycloak(ctx)
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		logrus.Fatal(err)

		return 1
	}

	return 0
}
