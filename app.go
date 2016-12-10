package main

import (
	"context"
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

// App contains all the state for the application.
type App struct {
	Config
	nsm.SessionInfo

	ctx     context.Context
	log     io.Writer
	nc      *nsm.Client
	servers []*sc.Server
}

// NewApp creates a new app.
func NewApp(ctx context.Context, config Config) (*App, error) {
	app := &App{
		Config: config,
		ctx:    ctx,
	}
	nc, err := nsm.NewClient(app.ctx, nsm.ClientConfig{
		Session: app,
		Name:    "sc-servers",
		PID:     os.Getpid(),
		Major:   1,
		Minor:   2,
	})
	if err != nil {
		return nil, errors.Wrap(err, "initializing nsm client")
	}
	app.nc = nc
	return app, nil
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

// Methods returns osc methods.
func (app *App) Methods() osc.Dispatcher {
	return osc.Dispatcher{
		AddressStartServer: func(msg osc.Message) error {
			if err := app.Log(AddressStartServer); err != nil {
				log.Fatal(err)
			}
			if err := app.Spawn(); err != nil {
				return errors.Wrap(err, "could not spawn scsynth")
			}
			app.Port = atomic.AddInt32(&app.Port, int32(1))
			return nil
		},
	}
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

// Spawn starts a new scsynth instance.
func (app *App) Spawn() error {
	server := &sc.Server{
		Network: "udp",
		Port:    int(app.Port),
	}
	if err := app.Logf("starting scsynth on port %d", app.Port); err != nil {
		log.Fatal(err)
	}
	if _, _, err := server.Start(5 * time.Second); err != nil {
		errors.Wrap(err, "could not start scsynth")
	}
	if err := app.Logf("started scsynth on port %d", app.Port); err != nil {
		log.Fatal(err)
	}
	app.servers = append(app.servers, server)
	return nil
}

func (app *App) Wait() error {
	return app.nc.Wait()
}
