/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

const (
	OFPT_HELLO = iota
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

const (
	OFPAT_OUTPUT       = iota /* Output to switch port. */
	OFPAT_SET_VLAN_VID        /* Set the 802.1q VLAN id. */
	OFPAT_SET_VLAN_PCP        /* Set the 802.1q priority. */
	OFPAT_STRIP_VLAN          /* Strip the 802.1q header. */
	OFPAT_SET_DL_SRC          /* Ethernet source address. */
	OFPAT_SET_DL_DST          /* Ethernet destination address. */
	OFPAT_SET_NW_SRC          /* IP source address. */
	OFPAT_SET_NW_DST          /* IP destination address. */
	OFPAT_SET_NW_TOS          /* IP ToS (DSCP field, 6 bits). */
	OFPAT_SET_TP_SRC          /* TCP/UDP source port. */
	OFPAT_SET_TP_DST          /* TCP/UDP destination port. */
	OFPAT_ENQUEUE             /* Output to queue. */
	OFPAT_VENDOR       = 0xffff
)

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

const (
	OFPC_FRAG_NORMAL = iota /* No special handling for fragments. */
	OFPC_FRAG_DROP          /* Drop fragments. */
	OFPC_FRAG_REASM         /* Reassemble (only if OFPC_IP_REASM set). */
	OFPC_FRAG_MASK
)
