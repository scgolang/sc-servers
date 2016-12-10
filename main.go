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
	defer app.Close()

	if err := app.Wait(); err != nil {
		log.Fatal(err)
	}
}
