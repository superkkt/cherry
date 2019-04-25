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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

type coreSDK struct {
	baseURL string
	client  *http.Client
}

func newCoreSDK(baseURL string) (*coreSDK, error) {
	if _, err := url.Parse(baseURL); err != nil {
		return nil, err
	}

	return &coreSDK{
		baseURL: baseURL,
		client: &http.Client{
			Transport: &http.Transport{
				TLSHandshakeTimeout: 10 * time.Second,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (r *coreSDK) Announce(ip net.IP, mac net.HardwareAddr) error {
	arg := &struct {
		IP  string `json:"ip"`
		MAC string `json:"mac"`
	}{
		IP:  ip.String(),
		MAC: mac.String(),
	}

	return r.call("POST", "/api/v1/announce", arg, nil)
}

func (r *coreSDK) RemoveFlows() error {
	return r.call("POST", "/api/v1/remove", nil, nil)
}

func (r *coreSDK) RemoveFlowsByMAC(mac net.HardwareAddr) error {
	arg := &struct {
		MAC string `json:"mac"`
	}{mac.String()}

	return r.call("POST", "/api/v1/remove", arg, nil)
}

func (r *coreSDK) IsMaster() bool {
	res := new(struct {
		Master bool `json:"master"`
	})
	if err := r.call("POST", "/api/v1/status", nil, res); err != nil {
		return false
	}

	return res.Master
}

func (r *coreSDK) call(method, command string, param interface{}, result interface{}) error {
	payload, err := json.Marshal(param)
	if err != nil {
		return err
	}
	logger.Debugf("REST request: %v", string(payload))

	req, err := http.NewRequest(method, r.baseURL+command, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected HTTP status code: %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	logger.Debugf("REST response: %v", string(body))

	res := new(struct {
		Status  int             `json:"status"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	})
	if err := json.Unmarshal(body, res); err != nil {
		return err
	}
	if res.Status != 200 {
		return fmt.Errorf("unexpected response status: status=%v, message=%v", res.Status, res.Message)
	}

	if result == nil {
		// Ignore the query result.
		return nil
	}

	return json.Unmarshal(res.Data, &result)
}
