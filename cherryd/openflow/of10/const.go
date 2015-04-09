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

const (
	/* Description of this OpenFlow switch.
	 * The request body is empty.
	 * The reply body is struct ofp_desc_stats. */
	OFPST_DESC = iota
	/* Individual flow statistics.
	 * The request body is struct ofp_flow_stats_request.
	 * The reply body is an array of struct ofp_flow_stats. */
	OFPST_FLOW
	/* Aggregate flow statistics.
	 * The request body is struct ofp_aggregate_stats_request.
	 * The reply body is struct ofp_aggregate_stats_reply. */
	OFPST_AGGREGATE
	/* Flow table statistics.
	 * The request body is empty.
	 * The reply body is an array of struct ofp_table_stats. */
	OFPST_TABLE
	/* Physical port statistics.
	 * The request body is struct ofp_port_stats_request.
	 * The reply body is an array of struct ofp_port_stats. */
	OFPST_PORT
	/* Queue statistics for a port
	 * The request body defines the port
	 * The reply body is an array of struct ofp_queue_stats */
	OFPST_QUEUE
	/* Vendor extension.
	 * The request and reply bodies begin with a 32-bit vendor ID, which takes
	 * the same form as in "struct ofp_vendor_header". The request and reply
	 * bodies are otherwise vendor-defined. */
	OFPST_VENDOR = 0xffff
)

const (
	OFPPC_PORT_DOWN    = 1 << 0
	OFPPC_NO_STP       = 1 << 1
	OFPPC_NO_RECV      = 1 << 2
	OFPPC_NO_RECV_STP  = 1 << 3
	OFPPC_NO_FLOOD     = 1 << 4
	OFPPC_NO_FWD       = 1 << 5
	OFPPC_NO_PACKET_IN = 1 << 6
)

const (
	OFPPS_LINK_DOWN   = 1 << 0
	OFPPS_STP_LISTEN  = 0 << 8 /* Not learning or relaying frames. */
	OFPPS_STP_LEARN   = 1 << 8 /* Learning but not relaying frames. */
	OFPPS_STP_FORWARD = 2 << 8 /* Learning and relaying frames. */
	OFPPS_STP_BLOCK   = 3 << 8 /* Not part of spanning tree. */
	OFPPS_STP_MASK    = 3 << 8 /* Bit mask for OFPPS_STP_* values. */
)

const (
	OFPPF_10MB_HD    = 1 << 0  /* 10 Mb half-duplex rate support. */
	OFPPF_10MB_FD    = 1 << 1  /* 10 Mb full-duplex rate support. */
	OFPPF_100MB_HD   = 1 << 2  /* 100 Mb half-duplex rate support. */
	OFPPF_100MB_FD   = 1 << 3  /* 100 Mb full-duplex rate support. */
	OFPPF_1GB_HD     = 1 << 4  /* 1 Gb half-duplex rate support. */
	OFPPF_1GB_FD     = 1 << 5  /* 1 Gb full-duplex rate support. */
	OFPPF_10GB_FD    = 1 << 6  /* 10 Gb full-duplex rate support. */
	OFPPF_COPPER     = 1 << 7  /* Copper medium. */
	OFPPF_FIBER      = 1 << 8  /* Fiber medium. */
	OFPPF_AUTONEG    = 1 << 9  /* Auto-negotiation. */
	OFPPF_PAUSE      = 1 << 10 /* Pause. */
	OFPPF_PAUSE_ASYM = 1 << 11 /* Asymmetric pause. */
)
