package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	"github.com/autowp/goautowp"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"gopkg.in/gographics/imagick.v2/imagick"
)

const attrsUpdateValuesAMQPFlag = "attrs-update-values-amqp"

var autowpApp *goautowp.Application

func captureOsInterrupt() chan bool {
	quit := make(chan bool)

	go func() {
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		for sig := range c {
			logrus.Infof("captured %v, stopping and exiting.", sig)

			close(quit)

			break
		}
	}()

	return quit
}

func main() { os.Exit(mainReturnWithCode()) }

func mainReturnWithCode() int { //nolint: maintidx
	logrus.SetLevel(logrus.WarnLevel)

	imagick.Initialize()
	defer imagick.Terminate()

	cfg := config.LoadConfig(".")

	config.ValidateConfig(cfg)

	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.SetLevel(level)

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
							&cli.StringFlag{
								Name:     "offset",
								Value:    "",
								Usage:    "offset",
								Required: false,
							},
						},
						Action: func(ctx context.Context, command *cli.Command) error {
							return autowpApp.ImageStorageListBrokenImages(
								ctx,
								command.String("dir"),
								command.String("offset"),
							)
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
							&cli.BoolFlag{
								Name:     "move-to-lost-and-found",
								Value:    false,
								Usage:    "true",
								Required: false,
							},
							&cli.StringFlag{
								Name:     "offset",
								Value:    "",
								Usage:    "offset",
								Required: false,
							},
						},
						Action: func(ctx context.Context, command *cli.Command) error {
							return autowpApp.ImageStorageListUnlinkedObjects(ctx,
								command.String("dir"),
								command.Bool("move-to-lost-and-found"),
								command.String("offset"),
							)
						},
					},
				},
			},
			{
				Name: "serve",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:     "df-amqp",
						Required: false,
					},
					&cli.BoolFlag{
						Name:     "monitoring-amqp",
						Required: false,
					},
					&cli.BoolFlag{
						Name:     "grpc",
						Required: false,
					},
					&cli.BoolFlag{
						Name:     "public",
						Required: false,
					},
					&cli.BoolFlag{
						Name:     "private",
						Required: false,
					},
					&cli.BoolFlag{
						Name:     "autoban",
						Required: false,
					},
					&cli.BoolFlag{
						Name:     attrsUpdateValuesAMQPFlag,
						Required: false,
					},
				},
				Action: func(ctx context.Context, cli *cli.Command) error {
					err := autowpApp.MigrateAutowp(ctx)
					if err != nil {
						return err
					}

					err = autowpApp.MigratePostgres(ctx)
					if err != nil {
						return err
					}

					quit := captureOsInterrupt()

					return autowpApp.Serve(ctx, goautowp.ServeOptions{
						DuplicateFinderAMQP:   cli.Bool("df-amqp"),
						MonitoringAMQP:        cli.Bool("monitoring-amqp"),
						GRPC:                  cli.Bool("grpc"),
						Public:                cli.Bool("public"),
						Private:               cli.Bool("private"),
						Autoban:               cli.Bool("autoban"),
						AttrsUpdateValuesAMQP: cli.Bool(attrsUpdateValuesAMQPFlag),
					}, quit)
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
			{
				Name: "scheduler-generate-index-cache",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.GenerateIndexCache(ctx)
				},
			},
			{
				Name: "specs-refresh-conflict-flags",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.SpecsRefreshConflictFlags(ctx)
				},
			},
			{
				Name: "specs-refresh-item-conflict-flags",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "item_id",
						Value:    0,
						Usage:    "item_id",
						Required: true,
					},
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					return autowpApp.SpecsRefreshItemConflictFlags(ctx, command.Int("item_id"))
				},
			},
			{
				Name: "specs-refresh-user-stat",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "user_id",
						Value:    0,
						Usage:    "user_id",
						Required: true,
					},
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					return autowpApp.SpecsRefreshUserConflicts(ctx, command.Int("user_id"))
				},
			},
			{
				Name: "specs-refresh-users-stat",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.SpecsRefreshUsersConflicts(ctx)
				},
			},
			{
				Name: "specs-refresh-actual-values",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.SpecsRefreshActualValues(ctx)
				},
			},
			{
				Name: "refresh-item-parent-language",
				Flags: []cli.Flag{
					&cli.UintFlag{
						Name:     "limit",
						Value:    0,
						Usage:    "limit",
						Required: true,
					},
					&cli.IntFlag{
						Name:     "parent_item_type_id",
						Value:    int64(schema.ItemTableItemTypeIDBrand),
						Usage:    "parent_item_type_id",
						Required: false,
					},
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					parentItemTypeID := command.Int("parent_item_type_id")

					return autowpApp.RefreshItemParentLanguage(ctx,
						schema.ItemTableItemTypeID(parentItemTypeID),
						uint(command.Uint("limit")),
					)
				},
			},
			{
				Name: "catalogue-refresh-brand-vehicle",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.RefreshItemParentAllAuto(ctx)
				},
			},
			{
				Name: "catalogue-rebuild-item-order-cache",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.RebuildItemOrderCache(ctx)
				},
			},
			{
				Name: "pictures-df-index",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.PicturesDfIndex(ctx)
				},
			},
			{
				Name: "pictures-fix-filenames",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.PicturesFixFilenames(ctx)
				},
			},
			{
				Name: "build-brands-sprite",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return autowpApp.BuildBrandsSprite(ctx)
				},
			},
			{
				Name: "telegram",
				Commands: []*cli.Command{
					{
						Name: "webhook-info",
						Action: func(_ context.Context, _ *cli.Command) error {
							return autowpApp.TelegramWebhookInfo()
						},
					},
					{
						Name: "register-webhook",
						Action: func(_ context.Context, _ *cli.Command) error {
							return autowpApp.TelegramRegisterWebhook()
						},
					},
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
