package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/autowp/goautowp"
)

func main() {

	config := goautowp.LoadConfig()

	goautowp.ValidateConfig(config)

	t, err := goautowp.NewService(config)

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

	t.Close()
	os.Exit(0)
}
