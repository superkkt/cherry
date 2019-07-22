/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015-2019 Samjung Data Service, Inc. All rights reserved.
 *
 *  Kitae Kim <superkkt@sds.co.kr>
 *  Donam Kim <donam.kim@sds.co.kr>
 *  Jooyoung Kang <jooyoung.kang@sds.co.kr>
 *  Changjin Choi <ccj9707@sds.co.kr>
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
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/superkkt/cherry"
	"github.com/superkkt/cherry/api"
	"github.com/superkkt/cherry/api/ui"
	"github.com/superkkt/cherry/database"
	"github.com/superkkt/cherry/ldap"
	"github.com/superkkt/cherry/log"

	"github.com/fsnotify/fsnotify"
	"github.com/superkkt/go-logging"
	"github.com/superkkt/viper"
)

const (
	programName     = "walnut"
	programVersion  = cherry.Version
	defaultLogLevel = logging.INFO
)

var (
	logger        = logging.MustGetLogger("main")
	loggerLeveled logging.LeveledBackend

	showHelp          = flag.Bool("help", false, "show this help and exit")
	showVersion       = flag.Bool("version", false, "show program version and exit")
	defaultConfigFile = flag.String("config", fmt.Sprintf("/usr/local/etc/%v.yaml", programName), "absolute path of the configuration file")
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().Unix())
}

func main() {
	parseCmdLines()
	initConfig()
	initLog()
	initAPIServer()
	waitSignal()
	logger.Infof("%v (version %v) shutdown complete!", programName, programVersion)
}

// Handle the command-line arguments.
func parseCmdLines() {
	flag.Parse()
	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}
	if *showVersion {
		fmt.Printf("%v v%v\n", programName, programVersion)
		os.Exit(0)
	}
}

func initConfig() {
	viper.SetConfigFile(*defaultConfigFile)

	// Read the config file.
	if err := viper.ReadInConfig(); err != nil {
		logger.Fatalf("failed to read the config file: %v", err)
	}

	// Watching and re-reading config file whenever it changes.
	viper.OnConfigChange(func(e fsnotify.Event) {
		// Ignore all the fsnotify operations except WRITE to avoid reading empty config.
		if e.Op != fsnotify.Write {
			return
		}
		logger.Infof("config file changed: %v", e.Name)
		// Set log level for all modules
		loggerLeveled.SetLevel(getLogLevel(), "")
	})
	viper.WatchConfig()

	if err := validateConfig(); err != nil {
		logger.Fatalf("invalid configuration: %v", err)
	}
}

// validateConfig validates essential configurations.
func validateConfig() error {
	// TODO: Add validation you want.
	return nil
}

func initLog() {
	logDriver := viper.GetString("log.driver")

	var err error
	var backend logging.Backend
	switch strings.ToLower(logDriver) {
	case "stderr":
		backend = logging.NewLogBackend(os.Stderr, "", 0)
		backend = logging.NewBackendFormatter(backend, logging.MustStringFormatter(`%{time} [%{pid}] %{level}: %{shortpkg}.%{longfunc}: %{message}`))
	case "syslog":
		backend, err = log.NewSyslog(programName)
		if err != nil {
			logger.Fatalf("failed to init log: %v", err)
		}
		backend = logging.NewBackendFormatter(backend, logging.MustStringFormatter(`%{level}: %{shortpkg}.%{longfunc}: %{message}`))
	default:
		logger.Fatalf("unsupported log driver: %v", logDriver)
	}

	loggerLeveled = logging.AddModuleLevel(backend)
	// Set log level for all modules
	loggerLeveled.SetLevel(getLogLevel(), "")
	logging.SetBackend(loggerLeveled)
}

func getLogLevel() logging.Level {
	level := strings.ToUpper(viper.GetString("log.level"))
	ret, err := logging.LogLevel(level)
	if err != nil {
		logger.Errorf("invalid log.level=%v, defaulting to %v..", level, defaultLogLevel)
		return defaultLogLevel
	}

	return ret
}

func initDatabase() *database.MySQL {
	db, err := database.NewMySQL()
	if err != nil {
		logger.Fatalf("failed to init MySQL database: %v", err)
	}

	return db
}

func initCoreSDK() *coreSDK {
	client, err := newCoreSDK(viper.GetString("core_api_url"))
	if err != nil {
		logger.Fatalf("failed to init the core API's SDK: %v", err)
	}

	return client
}

func initLDAPClient() *ldap.Client {
	return ldap.New(viper.Sub("ldap"), 5)
}

func initAPIServer() {
	go func() {
		s := api.Server{}
		s.Port = uint16(viper.GetInt("rest.port"))
		if viper.GetBool("rest.tls") == true {
			s.TLS.Cert = viper.GetString("rest.cert_file")
			s.TLS.Key = viper.GetString("rest.key_file")
		}
		sdk := initCoreSDK()
		s.Observer = sdk
		s.Controller = sdk

		srv := &ui.API{Server: s, DB: initDatabase(), LDAP: initLDAPClient()}
		if err := srv.Serve(); err != nil {
			logger.Fatalf("failed to run the API server: %v", err)
		}
	}()
}

// waitSignal waits until we receive SIGTERM or SIGINT signals.
func waitSignal() {
	c := make(chan os.Signal, 1)
	// Following signals will be transferred to the channel c.
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGPIPE)

	// Infinite loop.
	for {
		s := <-c
		switch s {
		case syscall.SIGTERM, syscall.SIGINT:
			logger.Infof("caught %v signal: shutting down...", s)
			return
		default:
			logger.Infof("caught %v signal: ignored!", s)
		}
	}
}
