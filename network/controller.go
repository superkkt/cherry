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

	"github.com/superkkt/cherry/protocol"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/dlintw/goconf"
	"github.com/op/go-logging"
	"golang.org/x/net/context"
)

var (
	logger = logging.MustGetLogger("network")
)

type database interface {
	AddHost(HostParam) (hostID uint64, err error)
	AddNetwork(net.IP, net.IPMask) (netID uint64, err error)
	AddSwitch(SwitchParam) (swID uint64, err error)
	AddVIP(VIPParam) (id uint64, cidr string, err error)
	Host(hostID uint64) (host Host, ok bool, err error)
	Hosts() ([]Host, error)
	IPAddrs(networkID uint64) ([]IP, error)
	Location(mac net.HardwareAddr) (dpid string, port uint32, status LocationStatus, err error)
	Network(net.IP) (n Network, ok bool, err error)
	Networks() ([]Network, error)
	RemoveHost(id uint64) (ok bool, err error)
	RemoveNetwork(id uint64) (ok bool, err error)
	RemoveSwitch(id uint64) (ok bool, err error)
	RemoveVIP(id uint64) (ok bool, err error)
	Switch(dpid uint64) (sw Switch, ok bool, err error)
	Switches() ([]Switch, error)
	SwitchPorts(switchID uint64) ([]SwitchPort, error)
	ToggleVIP(id uint64) (net.IP, net.HardwareAddr, error)
	VIPs() ([]VIP, error)
}

type LocationStatus int

const (
	// Unregistered MAC address.
	LocationUnregistered LocationStatus = iota
	// Registered MAC address, but we don't know its physical location yet.
	LocationUndiscovered
	// Registered MAC address, and we know its physical location.
	LocationDiscovered
)

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
	topo     *topology
	listener EventListener
	db       database
}

func NewController(db database, conf *goconf.ConfigFile) *Controller {
	v := &Controller{
		topo: newTopology(db),
		db:   db,
	}
	go v.serveREST(conf)

	return v
}

func (r *Controller) serveREST(conf *goconf.ConfigFile) {
	c, err := parseRESTConfig(conf)
	if err != nil {
		logger.Errorf("failed to parse REST configurations: %v", err)
		return
	}

	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Get("/api/v1/switch", r.listSwitch),
		rest.Post("/api/v1/switch", r.addSwitch),
		rest.Delete("/api/v1/switch/:id", r.removeSwitch),
		rest.Options("/api/v1/switch/:id", r.allowOrigin),
		rest.Get("/api/v1/port/:switchID", r.listPort),
		rest.Get("/api/v1/network", r.listNetwork),
		rest.Post("/api/v1/network", r.addNetwork),
		rest.Delete("/api/v1/network/:id", r.removeNetwork),
		rest.Options("/api/v1/network/:id", r.allowOrigin),
		rest.Get("/api/v1/ip/:networkID", r.listIP),
		rest.Get("/api/v1/host", r.listHost),
		rest.Post("/api/v1/host", r.addHost),
		rest.Delete("/api/v1/host/:id", r.removeHost),
		rest.Options("/api/v1/host/:id", r.allowOrigin),
		rest.Get("/api/v1/vip", r.listVIP),
		rest.Post("/api/v1/vip", r.addVIP),
		rest.Delete("/api/v1/vip/:id", r.removeVIP),
		rest.Options("/api/v1/vip/:id", r.allowOrigin),
		rest.Put("/api/v1/vip/:id", r.toggleVIP),
	)
	if err != nil {
		logger.Errorf("failed to make a REST router: %v", err)
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
		logger.Errorf("failed to listen on HTTP(S): %v", err)
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

func (r *Controller) allowOrigin(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, PUT")
}

type SwitchParam struct {
	DPID        uint64 `json:"dpid"`
	NumPorts    uint16 `json:"n_ports"`
	FirstPort   uint16 `json:"first_port"`
	Description string `json:"description"`
}

func (r *SwitchParam) validate() error {
	if r.NumPorts > 512 {
		return errors.New("too many ports")
	}
	if uint32(r.FirstPort)+uint32(r.NumPorts) > 0xFFFF {
		return errors.New("too high first port number")
	}

	return nil
}

type Switch struct {
	ID uint64 `json:"id"`
	SwitchParam
}

func (r *Controller) listSwitch(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	logger.Debug("listing all switches..")
	sw, err := r.db.Switches()
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteJson(&struct {
		Switches []Switch `json:"switches"`
	}{sw})
}

func (r *Controller) addSwitch(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	sw := SwitchParam{}
	if err := req.DecodeJsonPayload(&sw); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := sw.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	logger.Debugf("adding a new switch whose DPID is %v", sw.DPID)
	_, ok, err := r.db.Switch(sw.DPID)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if ok {
		logger.Infof("duplicated switch DPID: %v", sw.DPID)
		writeError(w, http.StatusConflict, errors.New("duplicated switch DPID"))
		return
	}
	swID, err := r.db.AddSwitch(sw)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logger.Infof("added the new switch whose DPID is %v", sw.DPID)

	w.WriteJson(&struct {
		SwitchID uint64 `json:"switch_id"`
	}{swID})
}

func (r *Controller) removeSwitch(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	id, err := strconv.ParseUint(req.PathParam("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	logger.Debugf("removing a switch whose id is %v", id)
	ok, err := r.db.RemoveSwitch(id)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("unknown switch ID"))
		return
	}
	logger.Infof("removed the switch whose id is %v", id)

	for _, sw := range r.topo.Devices() {
		logger.Infof("removing all flows from %v", sw.ID())
		if err := sw.RemoveAllFlows(); err != nil {
			logger.Warningf("failed to remove all flows on %v device: %v", sw.ID(), err)
			continue
		}
	}

	w.WriteJson(&struct{}{})
}

type SwitchPort struct {
	ID     uint64 `json:"id"`
	Number uint   `json:"number"`
}

func (r *Controller) listPort(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	swID, err := strconv.ParseUint(req.PathParam("switchID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	ports, err := r.db.SwitchPorts(swID)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteJson(&struct {
		Ports []SwitchPort `json:"ports"`
	}{ports})
}

type NetworkParam struct {
	Address string `json:"address"`
	Mask    uint8  `json:"mask"`
}

func (r *NetworkParam) validate() error {
	if net.ParseIP(r.Address) == nil {
		return errors.New("invalid network address")
	}
	if r.Mask < 24 || r.Mask > 30 {
		return errors.New("invalid network mask")
	}

	return nil
}

type Network struct {
	ID uint64 `json:"id"`
	NetworkParam
}

func (r *Controller) listNetwork(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	logger.Debug("listing all networks..")
	networks, err := r.db.Networks()
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteJson(&struct {
		Networks []Network `json:"networks"`
	}{networks})
}

func (r *Controller) addNetwork(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	network := NetworkParam{}
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

	logger.Debugf("adding new network address: %v/%v", network.Address, network.Mask)
	_, ok, err := r.db.Network(netAddr)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if ok {
		logger.Infof("duplicated network address: %v/%v", network.Address, network.Mask)
		writeError(w, http.StatusConflict, errors.New("duplicated network address"))
		return
	}
	netID, err := r.db.AddNetwork(netAddr, netMask)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logger.Infof("added new network address: %v/%v", network.Address, network.Mask)

	w.WriteJson(&struct {
		NetworkID uint64 `json:"network_id"`
	}{netID})
}

func (r *Controller) removeNetwork(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	id, err := strconv.ParseUint(req.PathParam("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	logger.Debugf("removing network address whose id is %v", id)
	ok, err := r.db.RemoveNetwork(id)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("unknown network ID"))
		return
	}
	logger.Infof("removed network address whose id is %v", id)

	for _, sw := range r.topo.Devices() {
		logger.Infof("removing all flows from %v", sw.ID())
		if err := sw.RemoveAllFlows(); err != nil {
			logger.Warningf("failed to remove all flows on %v device: %v", sw.ID(), err)
			continue
		}
	}

	w.WriteJson(&struct{}{})
}

type IP struct {
	ID      uint64 `json:"id"`
	Address string `json:"address"`
	Used    bool   `json:"used"`
	Port    string `json:"port"`
	Host    string `json:"host"`
}

func (r *Controller) listIP(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	networkID, err := strconv.ParseUint(req.PathParam("networkID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	addresses, err := r.db.IPAddrs(networkID)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteJson(&struct {
		Addresses []IP `json:"addresses"`
	}{addresses})
}

type HostParam struct {
	IPID        uint64 `json:"ip_id"`
	MAC         string `json:"mac"`
	Description string `json:"description"`
}

func (r *HostParam) validate() error {
	_, err := net.ParseMAC(r.MAC)
	if err != nil {
		return err
	}

	return nil
}

type Host struct {
	ID          string `json:"id"`
	IP          string `json:"ip"`
	Port        string `json:"port"`
	MAC         string `json:"mac"`
	Description string `json:"description"`
	Stale       bool   `json:"stale"`
}

func (r *Controller) listHost(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	logger.Debug("listing all hosts..")
	hosts, err := r.db.Hosts()
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteJson(&struct {
		Hosts []Host `json:"hosts"`
	}{hosts})
}

func (r *Controller) addHost(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	host := HostParam{}
	if err := req.DecodeJsonPayload(&host); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := host.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	logger.Debugf("adding a new host (%+v)", host)
	hostID, err := r.db.AddHost(host)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logger.Infof("added the new host (%+v)", host)

	w.WriteJson(&struct {
		HostID uint64 `json:"host_id"`
	}{hostID})

	regHost, ok, err := r.db.Host(hostID)
	if err != nil {
		logger.Errorf("failed to query a registered host: %v", err)
		return
	}
	if !ok {
		logger.Errorf("registered host (ID=%v) is vanished", hostID)
		return
	}

	// Sends ARP announcement to all hosts to update their ARP caches
	if err := r.sendARPAnnouncement(regHost.IP, regHost.MAC); err != nil {
		logger.Errorf("failed to send ARP announcement for newly added host (ID=%v): %v", hostID, err)
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
		logger.Infof("sending ARP announcement for a host (IP: %v, MAC: %v) via %v", ip, hwAddr, sw.ID())
		if err := sw.SendARPAnnouncement(ip, hwAddr); err != nil {
			logger.Errorf("failed to send ARP announcement via %v: %v", sw.ID(), err)
			continue
		}
	}

	return nil
}

func (r *Controller) removeHost(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	id, err := strconv.ParseUint(req.PathParam("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	host, ok, err := r.db.Host(id)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
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

	logger.Debugf("removing a host whose MAC address is %v", mac)
	_, err = r.db.RemoveHost(id)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logger.Infof("removed the host whose MAC address is %v", mac)
	// Remove flows whose destination MAC is one we are removing when we remove a host
	r.removeFlows(mac)

	w.WriteJson(&struct{}{})
}

type VIPParam struct {
	IPID          uint64 `json:"ip_id"`
	ActiveHostID  uint64 `json:"active_host_id"`
	StandbyHostID uint64 `json:"standby_host_id"`
	Description   string `json:"description"`
}

func (r *VIPParam) validate() error {
	if r.ActiveHostID == r.StandbyHostID {
		return errors.New("same host for the active and standby")
	}

	return nil
}

type VIP struct {
	ID          uint64 `json:"id"`
	IP          string `json:"ip"`
	ActiveHost  Host   `json:"active_host"`
	StandbyHost Host   `json:"standby_host"`
	Description string `json:"description"`
}

func (r *Controller) listVIP(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vip, err := r.db.VIPs()
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteJson(&struct {
		VIP []VIP `json:"vip"`
	}{vip})
}

func (r *Controller) addVIP(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vip := VIPParam{}
	if err := req.DecodeJsonPayload(&vip); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := vip.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	logger.Debugf("adding a new VIP (%+v)", vip)
	id, cidr, err := r.db.AddVIP(vip)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logger.Infof("added the new VIP (%+v)", vip)

	w.WriteJson(&struct {
		ID uint64 `json:"vip_id"`
	}{id})

	active, ok, err := r.db.Host(vip.ActiveHostID)
	if err != nil {
		logger.Errorf("failed to query active VIP host: %v", err)
		return
	}
	if !ok {
		logger.Errorf("unknown active VIP host (ID=%v)", vip.ActiveHostID)
		return
	}

	// Sends ARP announcement to all hosts to update their ARP caches (IP = VIP, MAC = Active's MAC)
	if err := r.sendARPAnnouncement(cidr, active.MAC); err != nil {
		logger.Errorf("failed to send ARP announcement for newly added VIP (ID=%v): %v", id, err)
		return
	}
}

func (r *Controller) removeVIP(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	id, err := strconv.ParseUint(req.PathParam("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	logger.Debugf("removing a VIP (ID=%v)..", id)
	_, err = r.db.RemoveVIP(id)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logger.Infof("removed the VIP (ID=%v)", id)

	w.WriteJson(&struct{}{})
}

func (r *Controller) removeFlows(mac net.HardwareAddr) {
	for _, device := range r.topo.Devices() {
		if err := device.RemoveFlowByMAC(mac); err != nil {
			logger.Errorf("failed to remove flows from %v: %v", device.ID(), err)
			continue
		}
		logger.Debugf("removed flows whose destination MAC address is %v on %v", mac, device.ID())
	}
}

func (r *Controller) toggleVIP(w rest.ResponseWriter, req *rest.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	id, err := strconv.ParseUint(req.PathParam("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	logger.Debugf("toggling a VIP (ID=%v)..", id)
	// Toggle VIP and get active server's IP and MAC addresses
	ip, mac, err := r.db.ToggleVIP(id)
	if err != nil {
		logger.Errorf("failed to query database: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logger.Infof("toggled the VIP (ID=%v)", id)

	for _, sw := range r.topo.Devices() {
		logger.Infof("sending ARP announcement for a host (IP: %v, MAC: %v) via %v", ip, mac, sw.ID())
		if err := sw.SendARPAnnouncement(ip, mac); err != nil {
			logger.Errorf("failed to send ARP announcement via %v: %v", sw.ID(), err)
			continue
		}
	}

	w.WriteJson(&struct{}{})
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
