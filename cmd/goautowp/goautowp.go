package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/autowp/goautowp"
	"github.com/autowp/goautowp/config"
	"github.com/autowp/goautowp/util"
	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/gographics/imagick.v2/imagick"
)

const sentryFlushTime = time.Second * 5

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

func createCmd(autowpApp *goautowp.Application) (*cobra.Command, error) {
	imageStorageCmd := &cobra.Command{
		Use: "image-storage",
	}

	imageStorageGetImageCmd := &cobra.Command{
		Use: "get-image",
		RunE: func(cmd *cobra.Command, args []string) error {
			imageID, err := cmd.Flags().GetInt("image_id")
			if err != nil {
				return err
			}
			i, err := autowpApp.ImageStorageGetImage(cmd.Context(), imageID)
			if err != nil {
				return err
			}

			logrus.Infof("%v", i)

			return nil
		},
	}
	imageStorageGetImageCmd.Flags().Int("image_id", 0, "Image ID")

	err := imageStorageGetImageCmd.MarkFlagRequired("image_id")
	if err != nil {
		return nil, err
	}

	getFormattedImageCmd := &cobra.Command{
		Use: "get-formatted-image",
		RunE: func(cmd *cobra.Command, args []string) error {
			imageID, err := cmd.Flags().GetInt("image_id")
			if err != nil {
				return err
			}

			format, err := cmd.Flags().GetString("format")
			if err != nil {
				return err
			}

			i, err := autowpApp.ImageStorageGetFormattedImage(cmd.Context(), imageID, format)
			if err != nil {
				return err
			}

			logrus.Infof("%v", i)

			return nil
		},
	}
	getFormattedImageCmd.Flags().Int("image_id", 0, "Image ID")
	getFormattedImageCmd.Flags().String("format", "", "Format")

	err = getFormattedImageCmd.MarkFlagRequired("image_id")
	if err != nil {
		return nil, err
	}

	err = getFormattedImageCmd.MarkFlagRequired("format")
	if err != nil {
		return nil, err
	}

	imageStorageCmd.AddCommand(
		imageStorageGetImageCmd,
		getFormattedImageCmd,
	)

	rootCmd := &cobra.Command{
		Use:   "goautowp",
		Short: "autowp cli interface",
	}

	rootCmd.AddCommand(
		imageStorageCmd,
		&cobra.Command{
			Use: "autoban",
			RunE: func(cmd *cobra.Command, args []string) error {
				quit := captureOsInterrupt()

				return autowpApp.Autoban(quit)
			},
		},
		&cobra.Command{
			Use: "listen-df-amqp",
			RunE: func(cmd *cobra.Command, args []string) error {
				quit := captureOsInterrupt()

				return autowpApp.ListenDuplicateFinderAMQP(quit)
			},
		},
		&cobra.Command{
			Use: "listen-monitoring-amqp",
			RunE: func(cmd *cobra.Command, args []string) error {
				quit := captureOsInterrupt()

				return autowpApp.ListenMonitoringAMQP(quit)
			},
		},
		&cobra.Command{
			Use: "serve-grpc",
			RunE: func(cmd *cobra.Command, args []string) error {
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
		&cobra.Command{
			Use: "serve-public",
			RunE: func(cmd *cobra.Command, args []string) error {
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
		&cobra.Command{
			Use: "serve-private",
			RunE: func(cmd *cobra.Command, args []string) error {
				quit := captureOsInterrupt()

				return autowpApp.ServePrivate(quit)
			},
		},
		&cobra.Command{
			Use: "migrate-autowp",
			RunE: func(cmd *cobra.Command, args []string) error {
				return autowpApp.MigrateAutowp()
			},
		},
		&cobra.Command{
			Use: "migrate-postgres",
			RunE: func(cmd *cobra.Command, args []string) error {
				return autowpApp.MigratePostgres()
			},
		},
		&cobra.Command{
			Use: "scheduler-hourly",
			RunE: func(cmd *cobra.Command, args []string) error {
				return autowpApp.SchedulerHourly(cmd.Context())
			},
		},
		&cobra.Command{
			Use: "scheduler-daily",
			RunE: func(cmd *cobra.Command, args []string) error {
				return autowpApp.SchedulerDaily(cmd.Context())
			},
		},
		&cobra.Command{
			Use: "scheduler-midnight",
			RunE: func(cmd *cobra.Command, args []string) error {
				return autowpApp.SchedulerMidnight(cmd.Context())
			},
		},
		&cobra.Command{
			Use: "export-users-to-keycloak",
			RunE: func(cmd *cobra.Command, args []string) error {
				return autowpApp.ExportUsersToKeycloak(cmd.Context())
			},
		},
	)

	return rootCmd, nil
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

	autowpApp := goautowp.NewApplication(cfg)
	defer util.Close(autowpApp)

	rootCmd, err := createCmd(autowpApp)
	if err != nil {
		logrus.Fatal(err)

		return 1
	}

	err = rootCmd.Execute()
	if err != nil {
		logrus.Fatal(err)
		sentry.CaptureException(err)

		return 1
	}

	return 0
}
