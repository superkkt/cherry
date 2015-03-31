/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

// ofp_type
type PacketType uint8

const (
	OFPT_HELLO PacketType = iota
	OFPT_ERROR
	OFPT_ECHO_REQUEST
	OFPT_ECHO_REPLY
	OFPT_VENDOR
	OFPT_FEATURES_REQUEST
	OFPT_FEATURES_REPLY
	OFPT_GET_CONFIG_REQUEST
	OFPT_GET_CONFIG_REPLY
	OFPT_SET_CONFIG
	OFPT_PACKET_IN
	OFPT_FLOW_REMOVED
	OFPT_PORT_STATUS
	OFPT_PACKET_OUT
	OFPT_FLOW_MOD
	OFPT_PORT_MOD
	OFPT_STATS_REQUEST
	OFPT_STATS_REPLY
	OFPT_BARRIER_REQUEST
	OFPT_BARRIER_REPLY
	OFPT_QUEUE_GET_CONFIG_REQUEST
	OFPT_QUEUE_GET_CONFIG_REPLY
)

// ofp_error_type
type ErrorType uint16

const (
	OFPET_HELLO_FAILED ErrorType = iota
	OFPET_BAD_REQUEST
	OFPET_BAD_ACTION
	OFPET_FLOW_MOD_FAILED
	OFPET_PORT_MOD_FAILED
	OFPET_QUEUE_OP_FAILED
)

type ErrorCode uint16

// ofp_hello_failed_code
const (
	OFPHFC_INCOMPATIBLE ErrorCode = iota
	OFPHFC_EPERM
)

// ofp_bad_request_code
const (
	OFPBRC_BAD_VERSION ErrorCode = iota
	OFPBRC_BAD_TYPE
	OFPBRC_BAD_STAT
	OFPBRC_BAD_VENDOR
	OFPBRC_BAD_SUBTYPE
	OFPBRC_EPERM
	OFPBRC_BAD_LEN
	OFPBRC_BUFFER_EMPTY
	OFPBRC_BUFFER_UNKNOWN
)

// ofp_bad_action_code
const (
	OFPBAC_BAD_TYPE ErrorCode = iota
	OFPBAC_BAD_LEN
	OFPBAC_BAD_VENDOR
	OFPBAC_BAD_VENDOR_TYPE
	OFPBAC_BAD_OUT_PORT
	OFPBAC_BAD_ARGUMENT
	OFPBAC_EPERM
	OFPBAC_TOO_MANY
	OFPBAC_BAD_QUEUE
)

// ofp_flow_mod_failed_code
const (
	OFPFMFC_ALL_TABLES_FULL ErrorCode = iota
	OFPFMFC_OVERLAP
	OFPFMFC_EPERM
	OFPFMFC_BAD_EMERG_TIMEOUT
	OFPFMFC_BAD_COMMAND
	OFPFMFC_UNSUPPORTED
)

// ofp_port_mod_failed_code
const (
	FPPMFC_BAD_PORT ErrorCode = iota
	OFPPMFC_BAD_HW_ADDR
)

// ofp_queue_op_failed_code
const (
	OFPQOFC_BAD_PORT ErrorCode = iota
	OFPQOFC_BAD_QUEUE
	OFPQOFC_EPERM
)

// ofp_port_config
type PortConfig uint32

const (
	OFPPC_PORT_DOWN    PortConfig = 1 << 0
	OFPPC_NO_STP                  = 1 << 1
	OFPPC_NO_RECV                 = 1 << 2
	OFPPC_NO_RECV_STP             = 1 << 3
	OFPPC_NO_FLOOD                = 1 << 4
	OFPPC_NO_FWD                  = 1 << 5
	OFPPC_NO_PACKET_IN            = 1 << 6
)

// ofp_port_state
type PortState uint32

const (
	OFPPS_LINK_DOWN   PortState = 1 << 0
	OFPPS_STP_LISTEN            = 0 << 8 /* Not learning or relaying frames. */
	OFPPS_STP_LEARN             = 1 << 8 /* Learning but not relaying frames. */
	OFPPS_STP_FORWARD           = 2 << 8 /* Learning and relaying frames. */
	OFPPS_STP_BLOCK             = 3 << 8 /* Not part of spanning tree. */
	OFPPS_STP_MASK              = 3 << 8 /* Bit mask for OFPPS_STP_* values. */
)

type PortFeature uint32

// ofp_port_features
const (
	OFPPF_10MB_HD    PortFeature = 1 << 0  /* 10 Mb half-duplex rate support. */
	OFPPF_10MB_FD                = 1 << 1  /* 10 Mb full-duplex rate support. */
	OFPPF_100MB_HD               = 1 << 2  /* 100 Mb half-duplex rate support. */
	OFPPF_100MB_FD               = 1 << 3  /* 100 Mb full-duplex rate support. */
	OFPPF_1GB_HD                 = 1 << 4  /* 1 Gb half-duplex rate support. */
	OFPPF_1GB_FD                 = 1 << 5  /* 1 Gb full-duplex rate support. */
	OFPPF_10GB_FD                = 1 << 6  /* 10 Gb full-duplex rate support. */
	OFPPF_COPPER                 = 1 << 7  /* Copper medium. */
	OFPPF_FIBER                  = 1 << 8  /* Fiber medium. */
	OFPPF_AUTONEG                = 1 << 9  /* Auto-negotiation. */
	OFPPF_PAUSE                  = 1 << 10 /* Pause. */
	OFPPF_PAUSE_ASYM             = 1 << 11 /* Asymmetric pause. */
)

// ofp_capabilities
const (
	OFPC_FLOW_STATS   = 1 << 0 /* Flow statistics. */
	OFPC_TABLE_STATS  = 1 << 1 /* Table statistics. */
	OFPC_PORT_STATS   = 1 << 2 /* Port statistics. */
	OFPC_STP          = 1 << 3 /* 802.1d spanning tree. */
	OFPC_RESERVED     = 1 << 4 /* Reserved, must be zero. */
	OFPC_IP_REASM     = 1 << 5 /* Can reassemble IP fragments. */
	OFPC_QUEUE_STATS  = 1 << 6 /* Queue statistics. */
	OFPC_ARP_MATCH_IP = 1 << 7 /* Match IP addresses in ARP pkts. */
)

// ofp_action_type
type ActionType uint16

const (
	OFPAT_OUTPUT       ActionType = iota /* Output to switch port. */
	OFPAT_SET_VLAN_VID                   /* Set the 802.1q VLAN id. */
	OFPAT_SET_VLAN_PCP                   /* Set the 802.1q priority. */
	OFPAT_STRIP_VLAN                     /* Strip the 802.1q header. */
	OFPAT_SET_DL_SRC                     /* Ethernet source address. */
	OFPAT_SET_DL_DST                     /* Ethernet destination address. */
	OFPAT_SET_NW_SRC                     /* IP source address. */
	OFPAT_SET_NW_DST                     /* IP destination address. */
	OFPAT_SET_NW_TOS                     /* IP ToS (DSCP field, 6 bits). */
	OFPAT_SET_TP_SRC                     /* TCP/UDP source port. */
	OFPAT_SET_TP_DST                     /* TCP/UDP destination port. */
	OFPAT_ENQUEUE                        /* Output to queue. */
	OFPAT_VENDOR       = 0xffff
)

// ofp_flow_wildcards
const (
	OFPFW_IN_PORT     = 1 << 0  /* Switch input port. */
	OFPFW_DL_VLAN     = 1 << 1  /* VLAN id. */
	OFPFW_DL_SRC      = 1 << 2  /* Ethernet source address. */
	OFPFW_DL_DST      = 1 << 3  /* Ethernet destination address. */
	OFPFW_DL_TYPE     = 1 << 4  /* Ethernet frame type. */
	OFPFW_NW_PROTO    = 1 << 5  /* IP protocol. */
	OFPFW_TP_SRC      = 1 << 6  /* TCP/UDP source port. */
	OFPFW_TP_DST      = 1 << 7  /* TCP/UDP destination port. */
	OFPFW_DL_VLAN_PCP = 1 << 20 /* VLAN priority. */
	OFPFW_NW_TOS      = 1 << 21 /* IP ToS (DSCP field, 6 bits). */
)

type FlowModifyCmd uint16

const (
	OFPFC_ADD           FlowModifyCmd = iota /* New flow. */
	OFPFC_MODIFY                             /* Modify all matching flows. */
	OFPFC_MODIFY_STRICT                      /* Modify entry strictly matching wildcards */
	OFPFC_DELETE                             /* Delete all matching flows. */
	OFPFC_DELETE_STRICT                      /* Strictly match wildcards and priority. */
)

type PortChangedReason uint8

const (
	OFPPR_ADD    PortChangedReason = iota /* The port was added. */
	OFPPR_DELETE                          /* The port was removed. */
	OFPPR_MODIFY                          /* Some attribute of the port has changed. */
)

// ofp_flow_mod_flags
const (
	OFPFF_SEND_FLOW_REM = 1 << 0 /* Send flow removed message when flow expires or is deleted. */
	OFPFF_CHECK_OVERLAP = 1 << 1 /* Check for overlapping entries first. */
	OFPFF_EMERG         = 1 << 2 /* Remark this is for emergency. */
)

type PortNumber uint16

const (
	/* Maximum number of physical switch ports. */
	OFPP_MAX PortNumber = 0xff00
	/* Fake output "ports". */
	OFPP_IN_PORT    = 0xfff8 /* Send the packet out the input port. */
	OFPP_TABLE      = 0xfff9 /* Perform actions in flow table. */
	OFPP_NORMAL     = 0xfffa /* Process with normal L2/L3 switching. */
	OFPP_FLOOD      = 0xfffb /* All physical ports except input port and those disabled by STP. */
	OFPP_ALL        = 0xfffc /* All physical ports except input port. */
	OFPP_CONTROLLER = 0xfffd /* Send to controller. */
	OFPP_LOCAL      = 0xfffe /* Local openflow "port". */
	OFPP_NONE       = 0xffff /* Not associated with a physical port. */
)

type PacketInReason uint8

const (
	OFPR_NO_MATCH PacketInReason = iota
	OFPR_ACTION
)

type FlowRemovedReason uint8

const (
	OFPRR_IDLE_TIMEOUT FlowRemovedReason = iota
	OFPRR_HARD_TIMEOUT
	OFPRR_DELETE
)

type StatsType uint16

const (
	OFPST_DESC StatsType = iota
	OFPST_FLOW
	OFPST_AGGREGATE
	OFPST_TABLE
	OFPST_PORT
	OFPST_QUEUE
	OFPST_VENDOR = 0xffff
)

type ConfigFlag uint16

const (
	OFPC_FRAG_NORMAL ConfigFlag = iota
	OFPC_FRAG_DROP
	OFPC_FRAG_REASM
	OFPC_FRAG_MASK
)
