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
	"git.sds.co.kr/cherry.git/cherryd/internal/application"
	"git.sds.co.kr/cherry.git/cherryd/internal/device"
	"golang.org/x/net/context"
	"log"
	"log/syslog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func initSyslog() (*log.Logger, error) {
	log, err := syslog.NewLogger(syslog.LOG_ERR|syslog.LOG_DAEMON, log.Lshortfile)
	if err != nil {
		return nil, err
	}

	return log, nil
}

func waitSignal(log *log.Logger, shutdown context.CancelFunc) {
	c := make(chan os.Signal, 5)
	// All incoming signals will be transferred to the channel
	signal.Notify(c)

	for {
		s := <-c
		if s == syscall.SIGTERM || s == syscall.SIGINT {
			// Graceful shutdown
			log.Print("Shutting down...")
			shutdown()
			time.Sleep(10 * time.Second) // let cancelation propagate
			log.Print("Halted")
			os.Exit(0)
		} else if s == syscall.SIGHUP {
			// XXX: Do something you need
			log.Print("SIGHUP")
		}
	}
}

func listen(ctx context.Context, log *log.Logger, config *Config) {
	type KeepAliver interface {
		SetKeepAlive(keepalive bool) error
		SetKeepAlivePeriod(d time.Duration) error
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", config.Port))
	if err != nil {
		log.Printf("Failed to listen on %v port: %v", config.Port, err)
		return
	}
	defer listener.Close()

	f := func(c chan<- net.Conn) {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Failed to accept a new connection: %v", err)
				continue
			}
			c <- conn
		}
	}
	backlog := make(chan net.Conn)
	go f(backlog)

	// Infinite loop
	for {
		select {
		case conn := <-backlog:
			if v, ok := conn.(KeepAliver); ok {
				log.Print("Trying to enable socket keepalive..")
				if err := v.SetKeepAlive(true); err == nil {
					log.Print("Setting socket keepalive period...")
					v.SetKeepAlivePeriod(time.Duration(30) * time.Second)
				} else {
					log.Printf("Failed to enable socket keepalive: %v", err)
				}
			}

			go func() {
				defer conn.Close()
				transceiver, err := device.NewTransceiver(conn, log, application.Pool)
				if err != nil {
					log.Printf("Failed to create a new transceiver: %v", err)
					return
				}
				transceiver.Run(ctx)
			}()
		case <-ctx.Done():
			return
		}
	}
}

func enableApplications(config *Config) error {
	for _, v := range config.Apps {
		if err := application.Pool.Enable(v.Name, v.Priority); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	flag.Parse()

	conf := NewConfig()
	if err := conf.Read(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read configurations: %v\n", err)
		os.Exit(1)
	}

	if err := enableApplications(conf); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enable applications: %v\n", err)
		os.Exit(1)
	}

	log, err := initSyslog()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init syslog: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go waitSignal(log, cancel)
	listen(ctx, log, conf)
}
