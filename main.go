package main

import (
	"log"
	"os"

	"github.com/scgolang/nsm"
	"github.com/scgolang/osc"
	"github.com/scgolang/sc"
)

const (
	// Name is the name of the application.
	Name = "sc-servers"

	// AddressStartServer is the OSC address
	// used for starting instances of scsynth.
	AddressStartServer = "/sc/server/start"
)

// App contains all the state for the application.
type App struct {
	nsm.SessionInfo
}

// Open opens a session or creates a new one.
func (app *App) Open(info nsm.SessionInfo) (string, nsm.Error) {
	return Name + " has opened a new session", nil
}

// Save saves the current session.
func (app *App) Save() (string, nsm.Error) {
	return Name + " has saved a session", nil
}

// Methods returns osc methods.
func (app *App) Methods() osc.Dispatcher {
	return osc.Dispatcher{
		AddressServerStart: func(msg *osc.Message) error {
			return nil
		},
	}
}

func main() {
	nc, err := nsm.NewClient(nsm.ClientConfig{
		Session: &App{},
		Name:    "sc-servers",
		PID:     os.Getpid(),
		Major:   1,
		Minor:   2,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := nc.Wait(); err != nil {
		log.Fatal(err)
	}
}
