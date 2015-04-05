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
