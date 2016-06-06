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
	"github.com/superkkt/cherry/cherryd/database"
	"github.com/superkkt/cherry/cherryd/log"
	"github.com/superkkt/cherry/cherryd/network"
	"github.com/superkkt/cherry/cherryd/northbound"
	"golang.org/x/net/context"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	Version           = "0.11-RC3"
	defaultConfigFile = "/usr/local/etc/cherryd.conf"
)

var (
	showVersion = flag.Bool("version", false, "Show program version and exit")
	configFile  = flag.String("config", defaultConfigFile, "Absolute path of the configuration file")
)

func listen(ctx context.Context, log *log.Syslog, port int, controller *network.Controller) {
	type KeepAliver interface {
		SetKeepAlive(keepalive bool) error
		SetKeepAlivePeriod(d time.Duration) error
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		log.Err(fmt.Sprintf("Failed to listen on %v port: %v", port, err))
		return
	}
	defer listener.Close()

	f := func(c chan<- net.Conn) {
		for {
			conn, err := listener.Accept()

			// Check shutdown signal
			select {
			case <-ctx.Done():
				log.Info("Socket listener is finished by the shutdown signal")
				return
			default:
			}

			if err != nil {
				log.Err(fmt.Sprintf("Failed to accept a new connection: %v", err))
				continue
			}
			c <- conn
			log.Info(fmt.Sprintf("New device is connected from %v", conn.RemoteAddr()))
		}
	}
	backlog := make(chan net.Conn, 32)
	go f(backlog)

	// Infinite loop
	for {
		select {
		case conn := <-backlog:
			log.Debug("Fetching a new connection from the backlog..")
			if v, ok := conn.(KeepAliver); ok {
				log.Debug("Trying to enable socket keepalive..")
				if err := v.SetKeepAlive(true); err == nil {
					log.Debug("Setting socket keepalive period...")
					// Makes a broken connection will be disconnected within 45 seconds.
					// http://felixge.de/2014/08/26/tcp-keepalive-with-golang.html
					v.SetKeepAlivePeriod(time.Duration(5) * time.Second)
				} else {
					log.Err(fmt.Sprintf("Failed to enable socket keepalive: %v", err))
				}
			}
			controller.AddConnection(ctx, conn)
		case <-ctx.Done():
			return
		}
	}
}

func createAppManager(config *Config, log *log.Syslog, db *database.MySQL) (*northbound.Manager, error) {
	manager, err := northbound.NewManager(config.RawConfig(), log, db)
	if err != nil {
		return nil, err
	}

	for _, v := range config.Apps {
		if err := manager.Enable(v); err != nil {
			return nil, fmt.Errorf("enabling %v: %v", v, err)
		}
	}

	return manager, nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	if *showVersion {
		fmt.Printf("Version: %v\n", Version)
		os.Exit(0)
	}

	conf := NewConfig()
	if err := conf.Read(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read configurations: %v\n", err)
		os.Exit(1)
	}

	log, err := log.NewSyslog(conf.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	db, err := database.NewMySQL(conf.RawConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init MySQL database: %v\n", err)
		os.Exit(1)
	}

	controller := network.NewController(log, db, conf.RawConfig())
	manager, err := createAppManager(conf, log, db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create application manager: %v\n", err)
		os.Exit(1)
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
				log.Info("Shutting down...")
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

	listen(ctx, log, conf.Port, controller)
}
