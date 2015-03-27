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
)

var (
	configFile = flag.String("config", "/usr/local/etc/cherryd.conf",
		"Absolute path of the configuration file")
)

type Config struct {
	ServerPort int
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Read() error {
	readConfigData, err := goconf.ReadConfigFile(*configFile)
	if err != nil {
		return fmt.Errorf("failed to read the config file: %v", err)
	}

	if err := c.readDefaultConfig(readConfigData); err != nil {
		return err
	}

	return nil
}

func (c *Config) readDefaultConfig(readConfigData *goconf.ConfigFile) error {
	var err error

	c.ServerPort, err = readConfigData.GetInt("default", "server_port")
	if err != nil || c.ServerPort <= 0 || c.ServerPort > 65535 {
		return errors.New("invalid server port value")
	}

	return nil
}
