/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service, Inc. All rights reserved.
 * Kitae Kim <superkkt@sds.co.kr>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along
 * with this program; if not, write to the Free Software Foundation, Inc.,
 * 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/superkkt/cherry/api"
	"github.com/superkkt/cherry/database"
	"github.com/superkkt/cherry/election"
	"github.com/superkkt/cherry/log"
	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/superkkt/go-logging"
	"github.com/superkkt/viper"
)

const (
	programName     = "cherry"
	programVersion  = "0.13.5"
	defaultLogLevel = logging.INFO
)

var (
	logger            = logging.MustGetLogger("main")
	loggerLeveled     logging.LeveledBackend
	showVersion       = flag.Bool("version", false, "Show program version and exit")
	defaultConfigFile = flag.String("config", fmt.Sprintf("/usr/local/etc/%v.yaml", programName), "absolute path of the configuration file")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if *showVersion {
		fmt.Printf("Version: %v\n", programVersion)
		os.Exit(0)
	}

	initConfig()
	if err := initLog(getLogLevel(viper.GetString("default.log_level"))); err != nil {
		logger.Fatalf("failed to init log: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	db, err := database.NewMySQL()
	if err != nil {
		logger.Fatalf("failed to init MySQL database: %v", err)
	}

	observer := initElectionObserver(ctx, db)
	controller := network.NewController(db)
	initAPIServer(observer, controller)
	manager, err := createAppManager(db)
	if err != nil {
		logger.Fatalf("failed to create application manager: %v", err)
	}
	manager.AddEventSender(controller)

	initSignalHandler(controller, manager, cancel)

	listen(ctx, viper.GetInt("default.port"), controller, observer)
}

func initConfig() {
	viper.SetConfigFile(*defaultConfigFile)
	// Read the config file.
	if err := viper.ReadInConfig(); err != nil {
		logger.Fatalf("failed to read the config file: %v", err)
	}
	// Watching and re-reading config file whenever it changes.
	viper.OnConfigChange(func(e fsnotify.Event) {
		// Ignore the WRITE operation to avoid reading empty config.
		if e.Op != fsnotify.Write {
			return
		}

		if loggerLeveled != nil {
			// Set log level for all modules
			loggerLeveled.SetLevel(getLogLevel(viper.GetString("default.log_level")), "")
		}
	})
	viper.WatchConfig()
	if err := validateConfig(); err != nil {
		logger.Fatalf("failed to validate the configuration: %v", err)
	}
}

func validateConfig() error {
	if port := viper.GetInt("default.port"); port <= 0 || port > 0xFFFF {
		return errors.New("invalid default.port")
	}
	if len(viper.GetString("default.log_level")) == 0 {
		return errors.New("invalid default.log_level")
	}
	if len(viper.GetString("default.applications")) == 0 {
		return errors.New("invalid default.applications")
	}
	if len(viper.GetString("default.admin_email")) == 0 {
		return errors.New("invalid default.admin_email")
	}
	vlanID := viper.GetInt("default.vlan_id")
	if vlanID < 0 || vlanID > 4095 {
		return errors.New("invalid default.vlan_id in the config file")
	}

	return nil
}

func initElectionObserver(ctx context.Context, db *database.MySQL) *election.Observer {
	observer := election.New(db)
	go func() {
		if err := observer.Run(ctx); err != nil {
			logger.Fatalf("failed to run the election observer: %v", err)
		}
		logger.Debugf("election observer terminated")
	}()

	return observer
}

func initAPIServer(observer *election.Observer, controller *network.Controller) {
	go func() {
		conf := api.Config{}
		conf.Port = uint16(viper.GetInt("rest.port"))
		if viper.GetBool("rest.tls") == true {
			conf.TLS.Cert = viper.GetString("rest.cert_file")
			conf.TLS.Key = viper.GetString("rest.key_file")
		}
		conf.Observer = observer
		conf.Controller = controller

		srv := &api.Core{Config: conf}
		if err := srv.Serve(); err != nil {
			logger.Fatalf("failed to run the API server: %v", err)
		}
	}()
}

func initSignalHandler(controller *network.Controller, manager *northbound.Manager, cancel context.CancelFunc) {
	go func() {
		c := make(chan os.Signal, 5)
		// All incoming signals will be transferred to the channel
		signal.Notify(c)

		// Infinte loop.
		for {
			s := <-c
			if s == syscall.SIGTERM || s == syscall.SIGINT {
				// Graceful shutdown
				logger.Warning("Shutting down...")
				cancel()
				// Timeout for cancelation
				time.Sleep(5 * time.Second)
				os.Exit(0)
			} else if s == syscall.SIGHUP {
				fmt.Println("* Controller status:")
				fmt.Println(controller.String())
				fmt.Printf("\n* Manager status:\n")
				fmt.Println(manager.String())
			}
		}
	}()
}

func initLog(level logging.Level) error {
	backend, err := log.NewSyslog(programName)
	if err != nil {
		return err
	}
	backend = logging.NewBackendFormatter(backend, logging.MustStringFormatter(`%{level}: %{shortpkg}.%{shortfunc}: %{message}`))

	loggerLeveled = logging.AddModuleLevel(backend)
	// Set log level for all modules
	loggerLeveled.SetLevel(level, "")
	logging.SetBackend(loggerLeveled)

	return nil
}

func getLogLevel(level string) logging.Level {
	level = strings.ToUpper(level)
	ret, err := logging.LogLevel(level)
	if err != nil {
		logger.Infof("invalid log level=%v, defaulting to %v..", level, defaultLogLevel)
		return defaultLogLevel
	}

	return ret
}

func listen(ctx context.Context, port int, controller *network.Controller, observer *election.Observer) {
	type KeepAliver interface {
		SetKeepAlive(keepalive bool) error
		SetKeepAlivePeriod(d time.Duration) error
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		logger.Errorf("failed to listen on %v port: %v", port, err)
		return
	}
	defer listener.Close()

	// Connection dispatcher.
	f := func(c chan<- net.Conn) {
		for {
			conn, err := listener.Accept()
			if err != nil {
				logger.Errorf("failed to accept a new connection: %v", err)
				continue
			}
			logger.Infof("new device is connected from %v", conn.RemoteAddr())

			// Only the master controller can serve the connections!
			if observer.IsMaster() == false {
				logger.Warningf("disconnecting the newly connected device (%v) because we are not the master controller!", conn.RemoteAddr())
				conn.Close()
				continue
			}

			// Pass the new connection into the backlog queue.
			c <- conn
		}
	}
	backlog := make(chan net.Conn, 32)
	go f(backlog)

	// Infinite loop
	for {
		select {
		case <-ctx.Done():
			logger.Debug("terminating the main listener loop...")
			return
		case conn := <-backlog:
			logger.Debug("fetching a new connection from the backlog..")
			if v, ok := conn.(KeepAliver); ok {
				logger.Debug("trying to enable socket keepalive..")
				if err := v.SetKeepAlive(true); err == nil {
					logger.Debug("setting socket keepalive period...")
					// Makes a broken connection will be disconnected within 45 seconds.
					// http://felixge.de/2014/08/26/tcp-keepalive-with-golang.html
					v.SetKeepAlivePeriod(time.Duration(5) * time.Second)
				} else {
					logger.Errorf("failed to enable socket keepalive: %v", err)
				}
			}
			controller.AddConnection(ctx, conn)
		}
	}
}

func createAppManager(db *database.MySQL) (*northbound.Manager, error) {
	manager, err := northbound.NewManager(db)
	if err != nil {
		return nil, err
	}

	apps, err := parseApplications()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse applications")
	}
	for _, v := range apps {
		if err := manager.Enable(v); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("enabling %v", v))
		}
	}

	return manager, nil
}

func parseApplications() ([]string, error) {
	// Remove spaces, and then split it using comma
	tokens := strings.Split(strings.Replace(viper.GetString("default.applications"), " ", "", -1), ",")
	if len(tokens) == 0 {
		return nil, errors.New("empty application")
	}

	return tokens, nil
}
