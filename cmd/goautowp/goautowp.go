package main

import (
	"github.com/casbin/casbin"
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

	enforcer := casbin.NewEnforcer("model.conf", "policy.csv")

	command := "start"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	wg := &sync.WaitGroup{}
	t, err := goautowp.NewService(wg, config, enforcer)

	if err != nil {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
		return
	}

	if command == "migrate" {
		t.Close()
		wg.Wait()
		os.Exit(0)
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

	sentry.Flush(time.Second * 5)

	t.Close()
	wg.Wait()
	os.Exit(0)
}
