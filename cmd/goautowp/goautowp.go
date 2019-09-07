package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/autowp/goautowp"
)

func main() {

	config := goautowp.LoadConfig()

	goautowp.ValidateConfig(config)
	wg := &sync.WaitGroup{}
	t, err := goautowp.NewService(wg, config)

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
		wg.Wait()
		os.Exit(0)
	}

	t.Close()
	wg.Wait()
	os.Exit(0)
}
