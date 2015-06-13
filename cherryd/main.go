/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package main

import (
	"flag"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/internal/controller"
	"golang.org/x/net/context"
	"log/syslog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func waitSignal(log *syslog.Writer, shutdown context.CancelFunc) {
	c := make(chan os.Signal, 5)
	// All incoming signals will be transferred to the channel
	signal.Notify(c)

	for {
		s := <-c
		if s == syscall.SIGTERM || s == syscall.SIGINT {
			// Graceful shutdown
			log.Info("Shutting down...")
			shutdown()
			time.Sleep(10 * time.Second) // let cancelation propagate
			log.Info("Halted")
			os.Exit(0)
		} else if s == syscall.SIGHUP {
			// XXX: Do something you need
			log.Debug("SIGHUP")
		}
	}
}

func listen(ctx context.Context, log *syslog.Writer, config *Config) {
	type KeepAliver interface {
		SetKeepAlive(keepalive bool) error
		SetKeepAlivePeriod(d time.Duration) error
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", config.Port))
	if err != nil {
		log.Err(fmt.Sprintf("Failed to listen on %v port: %v", config.Port, err))
		return
	}
	defer listener.Close()

	f := func(c chan<- net.Conn) {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Err(fmt.Sprintf("Failed to accept a new connection: %v", err))
				continue
			}
			c <- conn
		}
	}
	backlog := make(chan net.Conn)
	go f(backlog)

	topo := controller.NewTopology(log)
	// Infinite loop
	for {
		select {
		case conn := <-backlog:
			if v, ok := conn.(KeepAliver); ok {
				log.Debug("Trying to enable socket keepalive..")
				if err := v.SetKeepAlive(true); err == nil {
					log.Debug("Setting socket keepalive period...")
					v.SetKeepAlivePeriod(time.Duration(30) * time.Second)
				} else {
					log.Err(fmt.Sprintf("Failed to enable socket keepalive: %v", err))
				}
			}

			topo.AddDeviceConn(conn)
		case <-ctx.Done():
			return
		}
	}
}

//func enableApplications(config *Config) error {
//	for _, v := range config.Apps {
//		if err := application.Pool.Enable(v.Name, v.Priority); err != nil {
//			return err
//		}
//	}
//
//	return nil
//}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	conf := NewConfig()
	if err := conf.Read(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read configurations: %v\n", err)
		os.Exit(1)
	}

	//	if err := enableApplications(conf); err != nil {
	//		fmt.Fprintf(os.Stderr, "Failed to enable applications: %v\n", err)
	//		os.Exit(1)
	//	}

	log, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "cherryd")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init syslog: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go waitSignal(log, cancel)
	listen(ctx, log, conf)
}
