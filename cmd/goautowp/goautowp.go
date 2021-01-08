package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/autowp/goautowp"
	"github.com/getsentry/sentry-go"
)

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

	command := "usage"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	wg := &sync.WaitGroup{}
	t, err := goautowp.NewService(wg, config)

	if err != nil {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
		return
	}

	log.Printf("Run %s\n", command)

	switch command {
	case "migrate-autowp":
		err = t.MigrateAutowp()
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(1)
			return
		}
		t.Close()
		wg.Wait()
		os.Exit(0)
		return
	case "serve-public":
		err = t.MigrateAutowp()
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(1)
			return
		}
		err = t.MigrateTraffic()
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(1)
			return
		}
		err = t.ServePublic()
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(1)
			return
		}

		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		for sig := range c {
			log.Printf("captured %v, stopping and exiting.", sig)

			sentry.Flush(time.Second * 5)

			t.Close()
			wg.Wait()
			os.Exit(0)
		}
	case "listen-df-amqp":
		quit := make(chan bool)
		err = t.ListenDuplicateFinderAMQP(quit)
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(1)
			return
		}

		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		for sig := range c {
			log.Printf("captured %v, stopping and exiting.", sig)

			quit <- true
			close(quit)
			t.Close()
			os.Exit(1)
		}
		return
	case "migrate-traffic":
		err = t.MigrateTraffic()
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(1)
			return
		}
		t.Close()
		os.Exit(0)
		return
	case "scheduler-hourly":
		err = t.SchedulerHourly()
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(1)
			return
		}
		t.Close()
		os.Exit(0)
		return
	case "autoban":
		quit := make(chan bool)
		err = t.Autoban(quit)
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(1)
			return
		}
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		for sig := range c {
			log.Printf("captured %v, stopping and exiting.", sig)

			quit <- true
			close(quit)
			t.Close()
			os.Exit(1)
		}
		return
	case "serve-private":
		err = t.ServePrivate()
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(1)
			return
		}

		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		for sig := range c {
			log.Printf("captured %v, stopping and exiting.", sig)

			t.Close()
			os.Exit(1)
		}
		return
	case "listen-monitoring-amqp":
		quit := make(chan bool)
		err = t.ListenMonitoringAMQP(quit)
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(1)
			return
		}

		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		for sig := range c {
			log.Printf("captured %v, stopping and exiting.", sig)

			quit <- true
			close(quit)
			t.Close()
			os.Exit(1)
		}
		return
	}

	sentry.Flush(time.Second * 5)

	t.Close()
	wg.Wait()
	os.Exit(0)
}
