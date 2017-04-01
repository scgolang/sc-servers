package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
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

	ctx          context.Context
	log          io.Writer
	nc           *nsm.Client
	servers      map[string]map[int32]*sc.Server // host => port => scsynth
	serversMutex sync.RWMutex
}

// NewApp creates a new app.
func NewApp(ctx context.Context, config Config) (*App, error) {
	// Initialize the app.
	app := &App{
		Config:  config,
		ctx:     ctx,
		log:     os.Stdout,
		servers: map[string]map[int32]*sc.Server{},
	}

	// Announce to nsm server.
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

	app.serversMutex.RLock()
	for _, portMap := range app.servers {
		for _, cmd := range portMap {
			if err := cmd.Stop(); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	app.serversMutex.RUnlock()

	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, " and "))
}

// Logf logs a message.
func (app *App) Logf(format string, args ...interface{}) error {
	s := fmt.Sprintf(format, args...)
	_, err := io.WriteString(app.log, s+"\n")
	return err
}

// Methods returns osc methods.
func (app *App) Methods() osc.Dispatcher {
	return osc.Dispatcher{
		AddressStartServer: app.StartServer,
		AddressStopServer:  app.StopServer,
	}
}

// Open opens a session or creates a new one.
func (app *App) Open(info nsm.SessionInfo) (string, nsm.Error) {
	msg := Name + " is listening at " + info.LocalAddr.String()
	if err := app.Log(msg); err != nil {
		log.Fatal(err)
	}
	return msg, nil
}

// Reply replies to a client with a success.
func (app *App) Reply(addr net.Addr, host string, port int32) error {
	return errors.Wrap(app.SendTo(addr, osc.Message{
		Address: nsm.AddressReply,
		Arguments: osc.Arguments{
			osc.String(host),
			osc.Int(port),
		},
	}), "sending reply")
}

// ReplyError replies to a client with an error message.
func (app *App) ReplyError(addr net.Addr, err error) error {
	return errors.Wrap(app.SendTo(addr, osc.Message{
		Address:   nsm.AddressError,
		Arguments: osc.Arguments{osc.String(err.Error())},
	}), "sending error")
}

// Log logs a message.
func (app *App) Log(msg string) error {
	_, err := io.WriteString(app.log, msg+"\n")
	return err
}

// Save saves the current session.
func (app *App) Save() (string, nsm.Error) {
	return Name + " has saved a session", nil
}

// SendTo sends an OSC message to a particular address.
func (app *App) SendTo(addr net.Addr, pkt osc.Packet) error {
	return app.nc.SendTo(addr, pkt)
}

// Spawn starts a new scsynth instance.
func (app *App) Spawn() error {
	server := &sc.Server{
		Network: "udp",
		Port:    int(app.Port),
	}

	// Spawn scsynth.
	if err := app.Logf("starting scsynth on port %d", app.Port); err != nil {
		log.Fatal(err)
	}
	if _, _, err := server.Start(5 * time.Second); err != nil {
		return errors.Wrap(err, "starting scsynth")
	}
	if err := app.Logf("started scsynth on port %d", app.Port); err != nil {
		log.Fatal(err)
	}

	// Add scsynth to map.
	app.serversMutex.Lock()
	portMap, ok := app.servers["127.0.0.1"]
	if !ok {
		portMap = map[int32]*sc.Server{}
		app.servers["127.0.0.1"] = portMap
	}
	portMap[app.Port] = server
	app.serversMutex.Unlock()

	return nil
}

// StartServer is an osc method that starts a new instance of scsynth.
func (app *App) StartServer(msg osc.Message) error {
	if err := app.Log(AddressStartServer); err != nil {
		return err
	}
	// TODO: how to support other hosts besides localhost?
	if err := app.Spawn(); err != nil {
		if err2 := app.ReplyError(msg.Sender, err); err2 != nil {
			_ = app.Log("could not send error reply: " + err2.Error())
		}
		_ = app.Log("could not spawn scsynth: " + err.Error())
	}
	if swapped := atomic.CompareAndSwapInt32(&app.Port, app.Port, app.Port+int32(1)); !swapped {
		return errors.New("atomically incrementing port")
	}
	return nil
}

// StopServer is an osc method that stops a running instance of scsynth.
func (app *App) StopServer(msg osc.Message) error {
	if l := len(msg.Arguments); l != 2 {
		return errors.Errorf("expected 2 arguments, got %d", l)
	}
	host, err := msg.Arguments[0].ReadString()
	if err != nil {
		return errors.Wrap(err, "reading string from first argument")
	}
	port, err := msg.Arguments[1].ReadInt32()
	if err != nil {
		return errors.Wrap(err, "reading int from second argument")
	}

	var server *sc.Server
	app.serversMutex.RLock()
	portMap, ok := app.servers[host]
	if !ok {
		app.serversMutex.RUnlock()
		return errors.New("no server for host: " + host)
	}
	server = portMap[port]
	app.serversMutex.RUnlock()

	if server == nil {
		return errors.Errorf("no server for port: %d", port)
	}

	return errors.Wrap(server.Stop(), "stopping scsynth")
}

// Wait returns when all the goroutines that are part of the application have returned
// nil or when one of them returns an error, whichever happens first.
func (app *App) Wait() error {
	return app.nc.Wait()
}
