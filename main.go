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
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/superkkt/cherry/database"
	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound"

	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

const (
	programName     = "cherry"
	programVersion  = "0.12.1"
	defaultLogLevel = logging.INFO
)

var (
	logger            = logging.MustGetLogger("main")
	showVersion       = flag.Bool("version", false, "Show program version and exit")
	defaultConfigFile = flag.String("config", fmt.Sprintf("/usr/local/etc/%v.conf", programName), "absolute path of the configuration file")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if *showVersion {
		fmt.Printf("Version: %v\n", programVersion)
		os.Exit(0)
	}

	conf := NewConfig()
	if err := conf.Read(); err != nil {
		logger.Fatalf("failed to read configurations: %v", err)
	}
	if err := initLog(getLogLevel(conf.LogLevel)); err != nil {
		logger.Fatalf("failed to init log: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	db, err := database.NewMySQL(conf.RawConfig())
	if err != nil {
		logger.Fatalf("failed to init MySQL database: %v", err)
	}
	controller := network.NewController(db, conf.RawConfig())
	manager, err := createAppManager(conf, db)
	if err != nil {
		logger.Fatalf("failed to create application manager: %v", err)
	}
	manager.AddEventSender(controller)

	// Signal handler
	go func() {
		c := make(chan os.Signal, 5)
		// All incoming signals will be transferred to the channel
		signal.Notify(c)

		for {
			s := <-c
			if s == syscall.SIGTERM || s == syscall.SIGINT {
				// Graceful shutdown
				logger.Info("Shutting down...")
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

	listen(ctx, conf.Port, controller)
}

func initLog(level logging.Level) error {
	backend, err := newSyslog(programName)
	if err != nil {
		return err
	}
	backend = logging.NewBackendFormatter(backend, logging.MustStringFormatter(`%{level}: %{shortpkg}.%{shortfunc}: %{message}`))

	leveled := logging.AddModuleLevel(backend)
	// Set log level for all modules
	leveled.SetLevel(level, "")
	logging.SetBackend(leveled)

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

func listen(ctx context.Context, port int, controller *network.Controller) {
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

	f := func(c chan<- net.Conn) {
		for {
			conn, err := listener.Accept()

			// Check shutdown signal
			select {
			case <-ctx.Done():
				logger.Info("socket listener is finished by the shutdown signal")
				return
			default:
			}

			if err != nil {
				logger.Errorf("failed to accept a new connection: %v", err)
				continue
			}
			c <- conn
			logger.Infof("new device is connected from %v", conn.RemoteAddr())
		}
	}
	backlog := make(chan net.Conn, 32)
	go f(backlog)

	// Infinite loop
	for {
		select {
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
		case <-ctx.Done():
			return
		}
	}
}

func createAppManager(config *Config, db *database.MySQL) (*northbound.Manager, error) {
	manager, err := northbound.NewManager(config.RawConfig(), db)
	if err != nil {
		return nil, err
	}

	for _, v := range config.Apps {
		if err := manager.Enable(v); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("enabling %v", v))
		}
	}

	return manager, nil
}
