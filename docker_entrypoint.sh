#!/bin/bash

shutdown()
{
	# Stop services and clean up here
	pkill cherryd
	echo "Stopping rsyslog.." 
	/etc/init.d/rsyslog stop
}

update_config() 
{
	v=${!1}
	if [ "$v" ]; then
		sed -i "s/$1/$v/g" /usr/local/etc/cherryd.conf
	else
		echo >&2 "Error: Empty $1 environment variable! Add -e $1=..."
		exit 1
	fi
}

update_config "DB_HOST"
update_config "DB_PORT"
update_config "DB_USER"
update_config "DB_PASSWORD"
update_config "DB_NAME"

# Use the trap if you need to also do manual cleanup after the service is stopped,
# or need to start multiple services in the one container
trap 'shutdown; exit' SIGINT SIGQUIT SIGTERM

# Start services in background here
/etc/init.d/rsyslog start
/go/bin/cherryd &
wait
