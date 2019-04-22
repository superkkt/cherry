/*
 * Copyright 2019 Samjung Data Service, Inc. All Rights Reserved.
 *
 * Authors:
 * 	Kitae Kim <superkkt@sds.co.kr>
 * 	Donam Kim <donam.kim@sds.co.kr>
 *	Jooyoung Kang <jooyoung.kang@sds.co.kr>
 *	Changjin Choi <ccj9707@sds.co.kr>
 */

package main

import (
	"net"
)

func main() {
	/* FYI.

	contoller := new(Controller)

	go func() {
		conf := api.Config{}
		conf.Port = uint16(viper.GetInt("rest.port"))
		if viper.GetBool("rest.tls") == true {
			conf.TLS.Cert = viper.GetString("rest.cert_file")
			conf.TLS.Key = viper.GetString("rest.key_file")
		}
		conf.Observer = observer
		conf.Controller = controller

		srv := &api.UI{Config: conf}
		if err := srv.Serve(); err != nil {
			logger.Fatalf("failed to run the API server: %v", err)
		}
	}()
	*/
}

type Controller struct {
	// TODO
}

func (r *Controller) Announce(net.IP, net.HardwareAddr) error {
	// TODO: Call the Core API via HTTP REST.
	return nil
}

func (r *Controller) RemoveFlows() error {
	// TODO: Call the Core API via HTTP REST.
	return nil
}

func (r *Controller) RemoveFlowsByMAC(net.HardwareAddr) error {
	// TODO: Call the Core API via HTTP REST.
	return nil
}
