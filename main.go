package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
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

	log     io.Writer
	port    int32
	servers []*sc.Server
}

// Open opens a session or creates a new one.
func (app *App) Open(info nsm.SessionInfo) (string, nsm.Error) {
	f, err := os.Create(info.ProjectPath)
	if err != nil {
		return "", nsm.NewError(nsm.ErrLaunchFailed, err.Error())
	}
	app.log = f

	msg := Name + " is listening at " + info.LocalAddr.String()
	if err := app.Log(msg); err != nil {
		log.Fatal(err)
	}
	return msg, nil
}

// Save saves the current session.
func (app *App) Save() (string, nsm.Error) {
	return Name + " has saved a session", nil
}

// Methods returns osc methods.
func (app *App) Methods() osc.Dispatcher {
	return osc.Dispatcher{
		AddressStartServer: func(msg *osc.Message) error {
			if err := app.Log(AddressStartServer); err != nil {
				log.Fatal(err)
			}
			if err := app.Spawn(); err != nil {
				return errors.Wrap(err, "could not spawn scsynth")
			}
			app.port = atomic.AddInt32(&app.port, int32(1))
			return nil
		},
	}
}

// Log logs a message to a file.
func (app *App) Log(msg string) error {
	_, err := io.WriteString(app.log, msg+"\n")
	return err
}

// Logf logs a message to a file.
func (app *App) Logf(format string, args ...interface{}) error {
	s := fmt.Sprintf(format, args...)
	_, err := io.WriteString(app.log, s+"\n")
	return err
}

// Spawn starts a new scsynth instance.
func (app *App) Spawn() error {
	server := &sc.Server{
		Network: "udp",
		Port:    int(app.port),
	}
	if err := app.Logf("starting scsynth on port %d", app.port); err != nil {
		log.Fatal(err)
	}
	if _, _, err := server.Start(5 * time.Second); err != nil {
		errors.Wrap(err, "could not start scsynth")
	}
	if err := app.Logf("started scsynth on port %d", app.port); err != nil {
		log.Fatal(err)
	}
	app.servers = append(app.servers, server)
	return nil
}

// Close closes the application.
func (app *App) Close() error {
	var errs []string
	for _, cmd := range app.servers {
		if err := cmd.Stop(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, " and "))
}

func main() {
	app := &App{port: 57120}
	defer app.Close()

	nc, err := nsm.NewClient(nsm.ClientConfig{
		Session: app,
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
