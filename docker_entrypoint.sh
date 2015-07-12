#!/bin/bash

shutdown()
{
	# Stop services and clean up here
	pkill cherryd
	echo "Stopping rsyslog.." 
	/etc/init.d/rsyslog stop
}

# Use the trap if you need to also do manual cleanup after the service is stopped,
# or need to start multiple services in the one container
trap 'shutdown; exit' SIGINT SIGQUIT SIGTERM

# Start services in background here
/etc/init.d/rsyslog start
/go/bin/cherryd &
wait
