# Cherry

Cherry is an OpenFlow controller written in Go that supports OpenFlow 1.0 and 1.3 protocols.

## Features

* Supports OpenFlow 1.0 and 1.3 protocols
* Focuses on compatibility with commercial OpenFlow-enabled switches
* Supports network topology that has loops in it
* Provides several northbound applications: ProxyARP, L2Switch, Router (in progress), Firewall (in progress), Loadbalancer (in progress), etc.
* Provides simple plugin system for northbound applications

## Tested OpenFlow-enabled Switches

* Dell Force10 S4810
* HP 2920G

## Quick Start

* Install Go language if you don't have it on your system by following instruction: http://golang.org/doc/install
* Clone and compile Cherry: 

 ```$ go get github.com/superkkt/cherry/cherryd```

* Copy the compiled binary and template configuration: 
 
 ```$ sudo cp $GOPATH/bin/cherryd /usr/local/bin```
 
 ```$ sudo cp $GOPATH/src/github.com/superkkt/cherry/cherryd/cherryd.conf /usr/local/etc```

* Run:

 ```$ /usr/local/bin/cherryd &```

* That's it! Cherry will be started in L2 switch mode.

## Documentation

in progress..

## Roadmap

* Support loadbalancer, router, and firewall applications until June 30, 2015
* Support multiple controllers for load balancing and automatic failover until July 31, 2015

## Copyright and License

```
Copyright (C) 2015 Samjung Data Service, Inc. All rights reserved.
Kitae Kim <superkkt@sds.co.kr>

This program is free software; you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation; either version 2 of the License, or
any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.
```
