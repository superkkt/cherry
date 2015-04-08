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
