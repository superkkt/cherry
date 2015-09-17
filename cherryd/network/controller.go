/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service, Inc. All rights reserved.
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

package network

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/dlintw/goconf"
	"github.com/superkkt/cherry/cherryd/log"
	"github.com/superkkt/cherry/cherryd/openflow"
	"github.com/superkkt/cherry/cherryd/protocol"
	"golang.org/x/net/context"
)

type database interface {
	AddHost(Host) (hostID uint64, err error)
	AddNetwork(net.IP, net.IPMask) (netID uint64, err error)
	AddSwitch(Switch) (swID uint64, err error)
	AddVIP(VIP) (id uint64, cidr string, err error)
	Host(hostID uint64) (host RegisteredHost, ok bool, err error)
	Hosts() ([]RegisteredHost, error)
	IPAddrs(networkID uint64) ([]IP, error)
	Location(mac net.HardwareAddr) (dpid string, port uint32, ok bool, err error)
	Network(net.IP) (n RegisteredNetwork, ok bool, err error)
	Networks() ([]RegisteredNetwork, error)
	RemoveHost(id uint64) (ok bool, err error)
	RemoveNetwork(id uint64) (ok bool, err error)
	RemoveSwitch(id uint64) (ok bool, err error)
	RemoveVIP(id uint64) (ok bool, err error)
	Switch(dpid uint64) (sw RegisteredSwitch, ok bool, err error)
	Switches() ([]RegisteredSwitch, error)
	SwitchPorts(switchID uint64) ([]SwitchPort, error)
	VIPActiveMAC(id uint64) (mac net.HardwareAddr, ok bool, err error)
	VIPs() ([]RegisteredVIP, error)
}

type EventListener interface {
	ControllerEventListener
	TopologyEventListener
}

type ControllerEventListener interface {
	OnPacketIn(Finder, *Port, *protocol.Ethernet) error
	OnPortUp(Finder, *Port) error
	OnPortDown(Finder, *Port) error
	OnDeviceUp(Finder, *Device) error
	OnDeviceDown(Finder, *Device) error
}

type TopologyEventListener interface {
	OnTopologyChange(Finder) error
}

type Controller struct {
	log      log.Logger
	topo     *topology
	listener EventListener
	db       database
}

func NewController(log log.Logger, db database, conf *goconf.ConfigFile) *Controller {
	if log == nil {
		panic("Logger is nil")
	}

	v := &Controller{
		log:  log,
		topo: newTopology(log, db),
		db:   db,
	}
	go v.serveREST(conf)

	return v
}

func (r *Controller) serveREST(conf *goconf.ConfigFile) {
	c, err := parseRESTConfig(conf)
	if err != nil {
		r.log.Err(fmt.Sprintf("Controller: parsing REST configurations: %v", err))
		return
	}

	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Get("/api/v1/switch", r.listSwitch),
		rest.Post("/api/v1/switch", r.addSwitch),
		rest.Delete("/api/v1/switch/:id", r.removeSwitch),
		rest.Get("/api/v1/port/:switchID", r.listPort),
		rest.Get("/api/v1/network", r.listNetwork),
		rest.Post("/api/v1/network", r.addNetwork),
		rest.Delete("/api/v1/network/:id", r.removeNetwork),
		rest.Get("/api/v1/ip/:networkID", r.listIP),
		rest.Get("/api/v1/host", r.listHost),
		rest.Post("/api/v1/host", r.addHost),
		rest.Delete("/api/v1/host/:id", r.removeHost),
		rest.Get("/api/v1/vip", r.listVIP),
		rest.Post("/api/v1/vip", r.addVIP),
		rest.Delete("/api/v1/vip/:id", r.removeVIP),
	)
	if err != nil {
		r.log.Err(fmt.Sprintf("Controller: making a REST router: %v", err))
		return
	}
	api.SetApp(router)

	addr := fmt.Sprintf(":%v", c.port)
	if c.tls.enable {
		err = http.ListenAndServeTLS(addr, c.tls.certFile, c.tls.keyFile, api.MakeHandler())
	} else {
		err = http.ListenAndServe(addr, api.MakeHandler())
	}

	if err != nil {
		r.log.Err(fmt.Sprintf("Controller: listening on HTTP: %v", err))
		return
	}
}

type restConfig struct {
	port uint16
	tls  struct {
		enable   bool
		certFile string
		keyFile  string
	}
}

func parseRESTConfig(conf *goconf.ConfigFile) (*restConfig, error) {
	var err error
	c := &restConfig{}

	c.tls.enable, err = conf.GetBool("rest", "tls")
	if err != nil {
		return nil, errors.New("invalid rest/tls value")
	}

	port, err := conf.GetInt("rest", "port")
	if err != nil || port <= 0 || port > 65535 {
		return nil, errors.New("empty or invalid rest/port value")
	}
	c.port = uint16(port)

	c.tls.certFile, err = conf.GetString("rest", "cert_file")
	if err != nil || len(c.tls.certFile) == 0 {
		return nil, errors.New("empty rest/cert_file value")
	}
	if c.tls.certFile[0] != '/' {
		return nil, errors.New("rest/cert_file should be specified as an absolute path")
	}

	c.tls.keyFile, err = conf.GetString("rest", "key_file")
	if err != nil || len(c.tls.keyFile) == 0 {
		return nil, errors.New("empty rest/key_file value")
	}
	if c.tls.keyFile[0] != '/' {
		return nil, errors.New("rest/key_file should be specified as an absolute path")
	}

	return c, nil
}

type Switch struct {
	DPID        uint64 `json:"dpid"`
	NumPorts    uint16 `json:"n_ports"`
	FirstPort   uint16 `json:"first_port"`
	Description string `json:"description"`
}

func (r *Switch) validate() error {
	if r.NumPorts > 512 {
		return errors.New("too many ports")
	}
	if uint32(r.FirstPort)+uint32(r.NumPorts) > 0xFFFF {
		return errors.New("too high first port number")
	}

	return nil
}

type RegisteredSwitch struct {
	ID uint64 `json:"id"`
	Switch
}

func (r *Controller) listSwitch(w rest.ResponseWriter, req *rest.Request) {
	sw, err := r.db.Switches()
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteJson(&struct {
		Switches []RegisteredSwitch `json:"switches"`
	}{sw})
}

func (r *Controller) addSwitch(w rest.ResponseWriter, req *rest.Request) {
	sw := Switch{}
	if err := req.DecodeJsonPayload(&sw); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := sw.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	r.log.Info(fmt.Sprintf("Controller: REST: adding a new switch whose DPID is %v", sw.DPID))
	_, ok, err := r.db.Switch(sw.DPID)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if ok {
		r.log.Info(fmt.Sprintf("Controller: REST: duplicated switch DPID: %v", sw.DPID))
		writeError(w, http.StatusConflict, errors.New("duplicated switch DPID"))
		return
	}
	swID, err := r.db.AddSwitch(sw)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	r.log.Info(fmt.Sprintf("Controller: REST: added the new switch whose DPID is %v", sw.DPID))

	w.WriteJson(&struct {
		SwitchID uint64 `json:"switch_id"`
	}{swID})
}

func (r *Controller) removeSwitch(w rest.ResponseWriter, req *rest.Request) {
	id, err := strconv.ParseUint(req.PathParam("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	r.log.Info(fmt.Sprintf("Controller: REST: removing a switch whose id is %v", id))
	ok, err := r.db.RemoveSwitch(id)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("unknown switch ID"))
		return
	}
	r.log.Info(fmt.Sprintf("Controller: REST: removed the switch whose id is %v", id))

	for _, sw := range r.topo.Devices() {
		r.log.Info(fmt.Sprintf("Controller: REST: removing all flows from %v", sw.ID()))
		if err := sw.RemoveAllFlows(); err != nil {
			r.log.Warning(fmt.Sprintf("Controller: REST: failed to remove all flows on %v device: %v", sw.ID(), err))
			continue
		}
	}

	w.WriteHeader(http.StatusOK)
}

type SwitchPort struct {
	ID     uint64 `json:"id"`
	Number uint   `json:"number"`
}

func (r *Controller) listPort(w rest.ResponseWriter, req *rest.Request) {
	swID, err := strconv.ParseUint(req.PathParam("switchID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	ports, err := r.db.SwitchPorts(swID)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteJson(&struct {
		Ports []SwitchPort `json:"ports"`
	}{ports})
}

type Network struct {
	Address string `json:"address"`
	Mask    uint8  `json:"mask"`
}

func (r *Network) validate() error {
	if net.ParseIP(r.Address) == nil {
		return errors.New("invalid network address")
	}
	if r.Mask < 24 || r.Mask > 30 {
		return errors.New("invalid network mask")
	}

	return nil
}

type RegisteredNetwork struct {
	ID uint64 `json:"id"`
	Network
}

func (r *Controller) listNetwork(w rest.ResponseWriter, req *rest.Request) {
	networks, err := r.db.Networks()
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteJson(&struct {
		Networks []RegisteredNetwork `json:"networks"`
	}{networks})
}

func (r *Controller) addNetwork(w rest.ResponseWriter, req *rest.Request) {
	network := Network{}
	if err := req.DecodeJsonPayload(&network); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := network.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	netMask := net.CIDRMask(int(network.Mask), 32)
	netAddr := net.ParseIP(network.Address)
	if netAddr == nil {
		panic("network.Address should be valid")
	}
	netAddr = netAddr.Mask(netMask)

	r.log.Info(fmt.Sprintf("Controller: REST: adding new network address: %v/%v", network.Address, network.Mask))
	_, ok, err := r.db.Network(netAddr)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if ok {
		r.log.Info(fmt.Sprintf("Controller: REST: duplicated network address: %v/%v", network.Address, network.Mask))
		writeError(w, http.StatusConflict, errors.New("duplicated network address"))
		return
	}
	netID, err := r.db.AddNetwork(netAddr, netMask)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	r.log.Info(fmt.Sprintf("Controller: REST: added new network address: %v/%v", network.Address, network.Mask))

	w.WriteJson(&struct {
		NetworkID uint64 `json:"network_id"`
	}{netID})
}

func (r *Controller) removeNetwork(w rest.ResponseWriter, req *rest.Request) {
	id, err := strconv.ParseUint(req.PathParam("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	r.log.Info(fmt.Sprintf("Controller: REST: removing network address whose id is %v", id))
	ok, err := r.db.RemoveNetwork(id)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("unknown network ID"))
		return
	}
	r.log.Info(fmt.Sprintf("Controller: REST: removed network address whose id is %v", id))

	for _, sw := range r.topo.Devices() {
		r.log.Info(fmt.Sprintf("Controller: REST: removing all flows from %v", sw.ID()))
		if err := sw.RemoveAllFlows(); err != nil {
			r.log.Warning(fmt.Sprintf("Controller: REST: failed to remove all flows on %v device: %v", sw.ID(), err))
			continue
		}
	}

	w.WriteHeader(http.StatusOK)
}

type IP struct {
	ID      uint64 `json:"id"`
	Address string `json:"address"`
	Used    bool   `json:"used"`
	Port    string `json:"port"`
	Host    string `json:"host"`
}

func (r *Controller) listIP(w rest.ResponseWriter, req *rest.Request) {
	networkID, err := strconv.ParseUint(req.PathParam("networkID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	addresses, err := r.db.IPAddrs(networkID)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteJson(&struct {
		Addresses []IP `json:"addresses"`
	}{addresses})
}

type Host struct {
	IPID        uint64 `json:"ip_id"`
	PortID      uint64 `json:"port_id"`
	MAC         string `json:"mac"`
	Description string `json:"description"`
}

func (r *Host) validate() error {
	_, err := net.ParseMAC(r.MAC)
	if err != nil {
		return err
	}

	return nil
}

type RegisteredHost struct {
	ID          string `json:"id"`
	IP          string `json:"ip"`
	Port        string `json:"port"`
	MAC         string `json:"mac"`
	Description string `json:"description"`
}

func (r *Controller) listHost(w rest.ResponseWriter, req *rest.Request) {
	hosts, err := r.db.Hosts()
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteJson(&struct {
		Hosts []RegisteredHost `json:"hosts"`
	}{hosts})
}

func (r *Controller) addHost(w rest.ResponseWriter, req *rest.Request) {
	host := Host{}
	if err := req.DecodeJsonPayload(&host); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := host.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	r.log.Info(fmt.Sprintf("Controller: REST: adding a new host (%+v)", host))
	hostID, err := r.db.AddHost(host)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	r.log.Info(fmt.Sprintf("Controller: REST: added the new host (%+v)", host))

	w.WriteJson(&struct {
		HostID uint64 `json:"host_id"`
	}{hostID})

	regHost, ok, err := r.db.Host(hostID)
	if err != nil {
		r.log.Err(fmt.Sprintf("Controller: REST: failed to query a registered host: %v", err))
		return
	}
	if !ok {
		r.log.Err(fmt.Sprintf("Controller: REST: registered host (ID=%v) is vanished", hostID))
		return
	}

	// Sends ARP announcement to all hosts to update their ARP caches
	if err := r.sendARPAnnouncement(regHost.IP, regHost.MAC); err != nil {
		r.log.Err(fmt.Sprintf("Controller: REST: failed to send ARP announcement for newly added host (ID=%v): %v", hostID, err))
		return
	}
}

func (r *Controller) sendARPAnnouncement(cidr string, mac string) error {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid IP address: %v", cidr)
	}
	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return err
	}

	for _, sw := range r.topo.Devices() {
		r.log.Info(fmt.Sprintf("Controller: REST: sending ARP announcement for a host (IP: %v, MAC: %v) via %v", ip, hwAddr, sw.ID()))
		if err := sw.SendARPAnnouncement(ip, hwAddr); err != nil {
			r.log.Err(fmt.Sprintf("Controller: REST: failed to send ARP announcement via %v: %v", sw.ID(), err))
			continue
		}
	}

	return nil
}

func (r *Controller) removeHost(w rest.ResponseWriter, req *rest.Request) {
	id, err := strconv.ParseUint(req.PathParam("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	host, ok, err := r.db.Host(id)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("unknown host ID"))
		return
	}
	mac, err := net.ParseMAC(host.MAC)
	if err != nil {
		panic("host.MAC should be valid")
	}

	r.log.Debug(fmt.Sprintf("Controller: REST: removing a host whose MAC address is %v", mac))
	_, err = r.db.RemoveHost(id)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	r.log.Debug(fmt.Sprintf("Controller: REST: removed the host whose MAC address is %v", mac))
	// Remove flows whose destination MAC is one we are removing when we remove a host
	r.removeFlows(mac)

	w.WriteHeader(http.StatusOK)
}

type VIP struct {
	IPID          uint64 `json:"ip_id"`
	ActiveHostID  uint64 `json:"active_host_id"`
	StandbyHostID uint64 `json:"standby_host_id"`
	Description   string `json:"description"`
}

type RegisteredVIP struct {
	ID          uint64         `json:"id"`
	IP          string         `json:"ip"`
	ActiveHost  RegisteredHost `json:"active_host"`
	StandbyHost RegisteredHost `json:"standby_host"`
	Description string         `json:"description"`
}

func (r *Controller) listVIP(w rest.ResponseWriter, req *rest.Request) {
	vip, err := r.db.VIPs()
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteJson(&struct {
		VIP []RegisteredVIP `json:"vip"`
	}{vip})
}

func (r *Controller) addVIP(w rest.ResponseWriter, req *rest.Request) {
	vip := VIP{}
	if err := req.DecodeJsonPayload(&vip); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	r.log.Info(fmt.Sprintf("Controller: REST: adding a new VIP (%+v)", vip))
	id, cidr, err := r.db.AddVIP(vip)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	r.log.Info(fmt.Sprintf("Controller: REST: added the new VIP (%+v)", vip))

	w.WriteJson(&struct {
		ID uint64 `json:"vip_id"`
	}{id})

	active, ok, err := r.db.Host(vip.ActiveHostID)
	if err != nil {
		r.log.Err(fmt.Sprintf("Controller: REST: failed to query active VIP host: %v", err))
		return
	}
	if !ok {
		r.log.Err(fmt.Sprintf("Controller: REST: unknown active VIP host (ID=%v)", vip.ActiveHostID))
		return
	}

	// Sends ARP announcement to all hosts to update their ARP caches (IP = VIP, MAC = Active's MAC)
	if err := r.sendARPAnnouncement(cidr, active.MAC); err != nil {
		r.log.Err(fmt.Sprintf("Controller: REST: failed to send ARP announcement for newly added VIP (ID=%v): %v", id, err))
		return
	}
}

func (r *Controller) removeVIP(w rest.ResponseWriter, req *rest.Request) {
	id, err := strconv.ParseUint(req.PathParam("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	mac, ok, err := r.db.VIPActiveMAC(id)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("unknown VIP active host"))
		return
	}

	r.log.Debug(fmt.Sprintf("Controller: REST: removing a VIP (ID=%v)", id))
	_, err = r.db.RemoveVIP(id)
	if err != nil {
		r.log.Info(fmt.Sprintf("Controller: REST: failed to query database: %v", err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	r.log.Debug(fmt.Sprintf("Controller: REST: removed the VIP (ID=%v)", id))
	// Remove flows whose destination MAC address is same with the active VIP host's one
	r.removeFlows(mac)

	w.WriteHeader(http.StatusOK)
}

func (r *Controller) removeFlows(mac net.HardwareAddr) {
	for _, sw := range r.topo.Devices() {
		f := sw.Factory()
		match, err := f.NewMatch()
		if err != nil {
			r.log.Err(fmt.Sprintf("Controller: REST: failed to create an OpenFlow match: %v", err))
			continue
		}
		match.SetDstMAC(mac)
		outPort := openflow.NewOutPort()
		outPort.SetNone()

		r.log.Debug(fmt.Sprintf("Controller: REST: removing flows whose destinatcion MAC address is %v on %v", mac, sw.ID()))
		if err := sw.RemoveFlow(match, outPort); err != nil {
			r.log.Err(fmt.Sprintf("Controller: REST: failed to remove a flow from %v: %v", sw.ID(), err))
			continue
		}
	}
}

func writeError(w rest.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	w.WriteJson(&struct {
		Error string `json:"error"`
	}{err.Error()})
}

func (r *Controller) AddConnection(ctx context.Context, c net.Conn) {
	conf := sessionConfig{
		conn:     c,
		logger:   r.log,
		watcher:  r.topo,
		finder:   r.topo,
		listener: r.listener,
	}
	session := newSession(conf)
	go session.Run(ctx)
}

func (r *Controller) SetEventListener(l EventListener) {
	r.listener = l
	r.topo.setEventListener(l)
}

func (r *Controller) String() string {
	return r.topo.String()
}
