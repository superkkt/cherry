/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

const (
	/* Immutable messages. */
	OFPT_HELLO        uint8 = iota /* Symmetric message */
	OFPT_ERROR                     /* Symmetric message */
	OFPT_ECHO_REQUEST              /* Symmetric message */
	OFPT_ECHO_REPLY                /* Symmetric message */
	OFPT_EXPERIMENTER              /* Symmetric message */
	/* Switch configuration messages. */
	OFPT_FEATURES_REQUEST   /* Controller/switch message */
	OFPT_FEATURES_REPLY     /* Controller/switch message */
	OFPT_GET_CONFIG_REQUEST /* Controller/switch message */
	OFPT_GET_CONFIG_REPLY   /* Controller/switch message */
	OFPT_SET_CONFIG         /* Controller/switch message */
	/* Asynchronous messages. */
	OFPT_PACKET_IN    /* Async message */
	OFPT_FLOW_REMOVED /* Async message */
	OFPT_PORT_STATUS  /* Async message */
	/* Controller command messages. */
	OFPT_PACKET_OUT /* Controller/switch message */
	OFPT_FLOW_MOD   /* Controller/switch message */
	OFPT_GROUP_MOD  /* Controller/switch message */
	OFPT_PORT_MOD   /* Controller/switch message */
	OFPT_TABLE_MOD  /* Controller/switch message */
	/* Multipart messages. */
	OFPT_MULTIPART_REQUEST /* Controller/switch message */
	OFPT_MULTIPART_REPLY   /* Controller/switch message */
	/* Barrier messages. */
	OFPT_BARRIER_REQUEST /* Controller/switch message */
	OFPT_BARRIER_REPLY   /* Controller/switch message */
	/* Queue Configuration messages. */
	OFPT_QUEUE_GET_CONFIG_REQUEST /* Controller/switch message */
	OFPT_QUEUE_GET_CONFIG_REPLY   /* Controller/switch message */
	/* Controller role change request messages. */
	OFPT_ROLE_REQUEST /* Controller/switch message */
	OFPT_ROLE_REPLY   /* Controller/switch message */
	/* Asynchronous message configuration. */
	OFPT_GET_ASYNC_REQUEST /* Controller/switch message */
	OFPT_GET_ASYNC_REPLY   /* Controller/switch message */
	OFPT_SET_ASYNC         /* Controller/switch message */
	/* Meters and rate limiters configuration messages. */
	OFPT_METER_MOD /* Controller/switch message */
)

const (
	OFPC_FLOW_STATS   = 1 << 0 /* Flow statistics. */
	OFPC_TABLE_STATS  = 1 << 1 /* Table statistics. */
	OFPC_PORT_STATS   = 1 << 2 /* Port statistics. */
	OFPC_GROUP_STATS  = 1 << 3 /* Group statistics. */
	OFPC_IP_REASM     = 1 << 5 /* Can reassemble IP fragments. */
	OFPC_QUEUE_STATS  = 1 << 6 /* Queue statistics. */
	OFPC_PORT_BLOCKED = 1 << 8 /* Switch will block looping ports. */
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
	 * The reply body is struct ofp_desc. */
	OFPMP_DESC = 0
	/* Individual flow statistics.
	 * The request body is struct ofp_flow_stats_request.
	 * The reply body is an array of struct ofp_flow_stats. */
	OFPMP_FLOW = 1
	/* Aggregate flow statistics.
	 * The request body is struct ofp_aggregate_stats_request.
	 * The reply body is struct ofp_aggregate_stats_reply. */
	OFPMP_AGGREGATE = 2
	/* Flow table statistics.
	 * The request body is empty.
	 * The reply body is an array of struct ofp_table_stats. */
	OFPMP_TABLE = 3
	/* Port statistics.
	 * The request body is struct ofp_port_stats_request.
	 * The reply body is an array of struct ofp_port_stats. */
	OFPMP_PORT_STATS = 4
	/* Queue statistics for a port
	 * The request body is struct ofp_queue_stats_request.
	 * The reply body is an array of struct ofp_queue_stats */
	OFPMP_QUEUE = 5
	/* Group counter statistics.
	 * The request body is struct ofp_group_stats_request.
	 * The reply is an array of struct ofp_group_stats. */
	OFPMP_GROUP = 6
	/* Group description.
	 * The request body is empty.
	 * The reply body is an array of struct ofp_group_desc_stats. */
	OFPMP_GROUP_DESC = 7
	/* Group features.
	 * The request body is empty.
	 * The reply body is struct ofp_group_features. */
	OFPMP_GROUP_FEATURES = 8
	/* Meter statistics.
	 * The request body is struct ofp_meter_multipart_requests.
	 * The reply body is an array of struct ofp_meter_stats. */
	OFPMP_METER = 9
	/* Meter configuration.
	 * The request body is struct ofp_meter_multipart_requests.
	 * The reply body is an array of struct ofp_meter_config. */
	OFPMP_METER_CONFIG = 10
	/* Meter features.
	 * The request body is empty.
	 * The reply body is struct ofp_meter_features. */
	OFPMP_METER_FEATURES = 11
	/* Table features.
	 * The request body is either empty or contains an array of
	 * struct ofp_table_features containing the controller's
	 * desired view of the switch. If the switch is unable to
	 * set the specified view an error is returned.
	 * The reply body is an array of struct ofp_table_features. */
	OFPMP_TABLE_FEATURES = 12
	/* Port description.
	 * The request body is empty.
	 * The reply body is an array of struct ofp_port. */
	OFPMP_PORT_DESC = 13
	/* Experimenter extension.
	 * The request and reply bodies begin with
	 * struct ofp_experimenter_multipart_header.
	 * The request and reply bodies are otherwise experimenter-defined. */
	OFPMP_EXPERIMENTER = 0xffff
)

const (
	OFPPC_PORT_DOWN    = 1 << 0 /* Port is administratively down. */
	OFPPC_NO_RECV      = 1 << 2
	OFPPC_NO_FWD       = 1 << 5
	OFPPC_NO_PACKET_IN = 1 << 6
)

const (
	OFPPS_LINK_DOWN = 1 << 0 /* No physical link present. */
	OFPPS_BLOCKED   = 1 << 1
	OFPPS_LIVE      = 1 << 2
)

const (
	OFPPF_10MB_HD    = 1 << 0
	OFPPF_10MB_FD    = 1 << 1
	OFPPF_100MB_HD   = 1 << 2
	OFPPF_100MB_FD   = 1 << 3
	OFPPF_1GB_HD     = 1 << 4
	OFPPF_1GB_FD     = 1 << 5
	OFPPF_10GB_FD    = 1 << 6
	OFPPF_40GB_FD    = 1 << 7
	OFPPF_100GB_FD   = 1 << 8
	OFPPF_1TB_FD     = 1 << 9
	OFPPF_OTHER      = 1 << 10
	OFPPF_COPPER     = 1 << 11
	OFPPF_FIBER      = 1 << 12
	OFPPF_AUTONEG    = 1 << 13
	OFPPF_PAUSE      = 1 << 14
	OFPPF_PAUSE_ASYM = 1 << 15
)

const (
	OFPXMT_OFB_IN_PORT = iota
	OFPXMT_OFB_IN_PHY_PORT
	OFPXMT_OFB_METADATA
	OFPXMT_OFB_ETH_DST
	OFPXMT_OFB_ETH_SRC
	OFPXMT_OFB_ETH_TYPE
	OFPXMT_OFB_VLAN_VID
	OFPXMT_OFB_VLAN_PCP
	OFPXMT_OFB_IP_DSCP
	OFPXMT_OFB_IP_ECN
	OFPXMT_OFB_IP_PROTO
	OFPXMT_OFB_IPV4_SRC
	OFPXMT_OFB_IPV4_DST
	OFPXMT_OFB_TCP_SRC
	OFPXMT_OFB_TCP_DST
	OFPXMT_OFB_UDP_SRC
	OFPXMT_OFB_UDP_DST
	OFPXMT_OFB_SCTP_SRC
	OFPXMT_OFB_SCTP_DST
	OFPXMT_OFB_ICMPV4_TYPE
	OFPXMT_OFB_ICMPV4_CODE
	OFPXMT_OFB_ARP_OP
	OFPXMT_OFB_ARP_SPA
	OFPXMT_OFB_ARP_TPA
	OFPXMT_OFB_ARP_SHA
	OFPXMT_OFB_ARP_THA
	OFPXMT_OFB_IPV6_SRC
	OFPXMT_OFB_IPV6_DST
	OFPXMT_OFB_IPV6_FLABEL
	OFPXMT_OFB_ICMPV6_TYPE
	OFPXMT_OFB_ICMPV6_CODE
	OFPXMT_OFB_IPV6_ND_TARGET
	OFPXMT_OFB_IPV6_ND_SLL
	OFPXMT_OFB_IPV6_ND_TLL
	OFPXMT_OFB_MPLS_LABEL
	OFPXMT_OFB_MPLS_TC
	OFPXMT_OFP_MPLS_BOS
	OFPXMT_OFB_PBB_ISID
	OFPXMT_OFB_TUNNEL_ID
	OFPXMT_OFB_IPV6_EXTHDR
)

const (
	OFPMT_STANDARD = 0
	OFPMT_OXM      = 1
)

const (
	OFPAT_OUTPUT    = 0
	OFPAT_SET_FIELD = 25
)

const (
	OFPP_TABLE      = 0xfffffff9
	OFPP_ALL        = 0xfffffffc
	OFPP_CONTROLLER = 0xfffffffd
	OFPP_ANY        = 0xffffffff
)

const (
	OFPG_ANY = 0xffffffff
)

const (
	OFPFC_ADD           = 0 /* New flow. */
	OFPFC_MODIFY        = 1 /* Modify all matching flows. */
	OFPFC_MODIFY_STRICT = 2 /* Modify entry strictly matching wildcards and priority. */
	OFPFC_DELETE        = 3 /* Delete all matching flows. */
	OFPFC_DELETE_STRICT = 4 /* Delete entry strictly matching wildcards and priority */
)

const (
	OFPIT_GOTO_TABLE     = 1      /* Setup the next table in the lookup pipeline */
	OFPIT_WRITE_METADATA = 2      /* Setup the metadata field for use later in pipeline */
	OFPIT_WRITE_ACTIONS  = 3      /* Write the action(s) onto the datapath action set */
	OFPIT_APPLY_ACTIONS  = 4      /* Applies the action(s) immediately */
	OFPIT_CLEAR_ACTIONS  = 5      /* Clears all actions from the datapath action set */
	OFPIT_METER          = 6      /* Apply meter (rate limiter) */
	OFPIT_EXPERIMENTER   = 0xFFFF /* Experimenter instruction */
)

const (
	OFP_NO_BUFFER = 0xffffffff
)

const (
	OFPFF_SEND_FLOW_REM = 1 << 0 /* Send flow removed message when flow expires or is deleted. */
	OFPFF_CHECK_OVERLAP = 1 << 1 /* Check for overlapping entries first. */
	OFPFF_RESET_COUNTS  = 1 << 2 /* Reset flow packet and byte counts. */
	OFPFF_NO_PKT_COUNTS = 1 << 3 /* Don't keep track of packet count. */
	OFPFF_NO_BYT_COUNTS = 1 << 4 /* Don't keep track of byte count. */
)

const (
	/* Last usable table number. */
	OFPTT_MAX = 0xfe
	/* Fake tables. */
	OFPTT_ALL = 0xff /* Wildcard table used for table config, flow stats and flow deletes. */
)
