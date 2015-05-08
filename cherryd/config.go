/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/dlintw/goconf"
	"strconv"
	"strings"
)

const (
	defaultConfigFile = "/usr/local/etc/cherryd.conf"
)

var (
	configFile = flag.String("config", defaultConfigFile, "Absolute path of the configuration file")
)

type Application struct {
	Name     string
	Priority uint
}

type Config struct {
	Port int
	Apps []Application
}

func NewConfig() *Config {
	return &Config{
		Apps: make([]Application, 0),
	}
}

func (c *Config) Read() error {
	conf, err := goconf.ReadConfigFile(*configFile)
	if err != nil {
		return fmt.Errorf("failed to read the config file: %v", err)
	}

	if err := c.readDefaultConfig(conf); err != nil {
		return err
	}

	return nil
}

func (c *Config) parseApplications(apps string) error {
	// NAME:PRIORITY,NAME:PRIORITY,...

	// Remove spaces, and then split it using comma
	token := strings.Split(strings.Replace(apps, " ", "", -1), ",")
	if len(token) == 0 {
		return errors.New("empty token")
	}

	for _, v := range token {
		// Application's name and priority are separated by colon.
		t := strings.Split(v, ":")
		if len(t) != 2 {
			return errors.New("priority is not specified")
		}
		priority, err := strconv.ParseUint(t[1], 10, 32)
		if err != nil {
			return fmt.Errorf("invalid priority: %v", err)
		}
		c.Apps = append(c.Apps, Application{Name: t[0], Priority: uint(priority)})
	}

	return nil
}

func (c *Config) readDefaultConfig(conf *goconf.ConfigFile) error {
	var err error

	c.Port, err = conf.GetInt("default", "port")
	if err != nil || c.Port <= 0 || c.Port > 0xFFFF {
		return errors.New("invalid port config")
	}

	apps, err := conf.GetString("default", "applications")
	if err != nil || len(apps) == 0 {
		return errors.New("empty applications config")
	}
	if err := c.parseApplications(apps); err != nil {
		return fmt.Errorf("invalid applications config: %v", err)
	}

	return nil
}
