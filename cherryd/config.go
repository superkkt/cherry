/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
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
	"errors"
	"fmt"
	"github.com/dlintw/goconf"
	"github.com/superkkt/cherry/cherryd/internal/log"
	"strings"
)

type Config struct {
	conf     *goconf.ConfigFile
	Port     int
	LogLevel log.Level
	Apps     []string
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

func (c *Config) parseLogLevel(l string) error {
	switch strings.ToUpper(l) {
	case "DEBUG":
		c.LogLevel = log.Debug
	case "INFO":
		c.LogLevel = log.Info
	case "NOTICE":
		c.LogLevel = log.Notice
	case "WARNING":
		c.LogLevel = log.Warning
	case "ERROR":
		c.LogLevel = log.Error
	default:
		return fmt.Errorf("invalid log level: %v", l)
	}

	return nil
}

func (c *Config) readDefaultConfig(conf *goconf.ConfigFile) error {
	var err error

	c.Port, err = conf.GetInt("default", "port")
	if err != nil || c.Port <= 0 || c.Port > 0xFFFF {
		return errors.New("invalid port in the config file")
	}

	logLevel, err := conf.GetString("default", "log_level")
	if err != nil || len(logLevel) == 0 {
		return errors.New("invalid log level in the config file")
	}
	if err := c.parseLogLevel(logLevel); err != nil {
		return err
	}

	apps, err := conf.GetString("default", "applications")
	if err != nil || len(apps) == 0 {
		return errors.New("empty applications in the config file")
	}
	if err := c.parseApplications(apps); err != nil {
		return fmt.Errorf("invalid applications config: %v", err)
	}

	return nil
}
