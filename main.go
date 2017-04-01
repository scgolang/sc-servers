package main

import (
	"context"
	"log"
)

const (
	// Name is the name of the application.
	Name = "sc-servers"

	// AddressStartServer is the OSC address used for starting instances of scsynth.
	AddressStartServer = "/sc/server/start"

	// AddressStopServer is the OSC address used for stopping instances of scsynth.
	AddressStopServer = "/sc/server/stop"
)

func main() {
	config, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	app, err := NewApp(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = app.Close() }()

	if err := app.Wait(); err != nil {
		log.Fatal(err)
	}
}
