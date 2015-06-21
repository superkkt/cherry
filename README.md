# Cherry

Cherry is an OpenFlow controller written in Go that supports OpenFlow 1.0 and 1.3 protocols.

## Features

* Support OpenFlow 1.0 and 1.3 protocols
* Support network topology that has loops in it
* Provide several northbound applications: ProxyARP, L2Switch, Router (in progress), Firewall (in progress), Loadbalancer (in progress), etc.
* Plugin system for northbound applications

## Quick Start

* Install Go language if you don't have it on your system by following instruction: http://golang.org/doc/install
* Clone and compile Cherry: 

 ```$ go get github.com/superkkt/cherry/cherryd```

* Copy the compiled binary and template configuration: 
 
 ```$ sudo cp $GOPATH/bin/cherryd /usr/local/bin```
 
 ```$ sudo cp $GOPATH/src/github.com/superkkt/cherry/cherryd/cherryd.conf /usr/local/etc```

* Run:

 ```$ /usr/local/bin/cherryd &```

* That's it! Cherry will be started as L2 switch mode.

## Documentation

in progress..

## Roadmap

* Support loadbalancer, router, and firewall applications until June 2015
* Support multiple controllers for load balancing and automatic failover until July 2015
