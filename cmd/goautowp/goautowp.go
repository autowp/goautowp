package main

import (
	"github.com/autowp/goautowp"
	"github.com/getsentry/sentry-go"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func captureOsInterrupt() chan bool {
	quit := make(chan bool)
	go func() {
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		for sig := range c {
			log.Printf("captured %v, stopping and exiting.", sig)

			quit <- true
			close(quit)
			break
		}
	}()

	return quit
}

func main() {

	config := goautowp.LoadConfig()

	goautowp.ValidateConfig(config)

	err := sentry.Init(sentry.ClientOptions{
		Dsn:         config.Sentry.DSN,
		Environment: config.Sentry.Environment,
	})

	if err != nil {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
		return
	}

	var opts struct {
		Command string `short:"f" long:"file" description:"Input file" value-name:"FILE"`
	}

	args, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
		return
	}

	command := "usage"
	if len(args) > 1 {
		command = args[1]
	}

	app, err := goautowp.NewApplication(config)

	if err != nil {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
		return
	}

	log.Printf("Run %s\n", command)

	var cmdErr error

	switch command {
	case "autoban":
		quit := captureOsInterrupt()
		cmdErr = app.Autoban(quit)
	case "listen-df-amqp":
		quit := captureOsInterrupt()
		cmdErr = app.ListenDuplicateFinderAMQP(quit)
	case "listen-monitoring-amqp":
		quit := captureOsInterrupt()
		cmdErr = app.ListenMonitoringAMQP(quit)
	case "migrate-autowp":
		cmdErr = app.MigrateAutowp()
	case "migrate-traffic":
		cmdErr = app.MigrateTraffic()
	case "scheduler-hourly":
		cmdErr = app.SchedulerHourly()
	case "serve-public":
		cmdErr = app.MigrateAutowp()
		if cmdErr != nil {
			break
		}
		cmdErr = app.MigrateTraffic()
		if cmdErr != nil {
			break
		}
		quit := captureOsInterrupt()
		cmdErr = app.ServePublic(quit)
	case "serve-private":
		quit := captureOsInterrupt()
		cmdErr = app.ServePrivate(quit)
	case "serve-auth":
		cmdErr = app.MigrateAuth()
		if cmdErr != nil {
			break
		}
		quit := captureOsInterrupt()
		cmdErr = app.ServeAuth(quit)
	}

	exitCode := 0
	if cmdErr != nil {
		log.Printf("Error: %s\n", cmdErr.Error())
		sentry.CaptureException(cmdErr)
		exitCode = 1
	}

	err = app.Close()
	if err != nil {
		log.Println(err.Error())
	}

	sentry.Flush(time.Second * 5)
	os.Exit(exitCode)
}
