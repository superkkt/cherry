/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package main

import (
	"errors"
	"fmt"
	"github.com/dlintw/goconf"
	"strings"
)

type Config struct {
	conf *goconf.ConfigFile
	Port int
	Apps []string
}

func NewConfig() *Config {
	return &Config{
		Apps: make([]string, 0),
	}
}

func (c *Config) Read() error {
	conf, err := goconf.ReadConfigFile(*configFile)
	if err != nil {
		return err
	}
	c.conf = conf

	if err := c.readDefaultConfig(conf); err != nil {
		return err
	}

	return nil
}

func (c *Config) RawConfig() *goconf.ConfigFile {
	return c.conf
}

func (c *Config) parseApplications(apps string) error {
	// Remove spaces, and then split it using comma
	tokens := strings.Split(strings.Replace(apps, " ", "", -1), ",")
	if len(tokens) == 0 {
		return errors.New("empty token")
	}

	for _, v := range tokens {
		c.Apps = append(c.Apps, v)
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
