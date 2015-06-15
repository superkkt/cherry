/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package l2switch

import (
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/internal/network"
	"git.sds.co.kr/cherry.git/cherryd/protocol"
)

type L2Switch struct {
}

func New() *L2Switch {
	return &L2Switch{}
}

func (r *L2Switch) Name() string {
	return "L2Switch"
}

func (r *L2Switch) Process(eth *protocol.Ethernet, ingress *network.Port, log log.Logger) (drop bool, err error) {
	log.Debug("L2Switch is executed..")
	return false, nil
}
