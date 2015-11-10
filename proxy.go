// vim: set ts=4 sw=4 tw=99 noet:
//
// Copyright 2014, Edmodo, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this work except in compliance with the License.
// You may obtain a copy of the License in the LICENSE file, or at:
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language
// governing permissions and limitations under the License.

package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Maintain a set of net.Listeners on a number of ports. All the ports are
// bound to a single source IP address. When a new container comes online,
// the listeners are reconfigured to forward traffic to a new IP.
type ProxyServer struct {
	sourceAddr string
	listeners  []*ProxyListener
}

// Create a new proxy server on the given source IP address, with a list of
// port mappings. The port mapping list should be a comma-delimited list of
// port numbers and/or host=container port mappings (such as 80=3000).
//
// So containers do not have to expose any ports, we assume their internal
// ports are fixed.
func NewProxyServer(sourceAddr, destPorts string) (*ProxyServer, error) {
	listeners := []*ProxyListener{}
	for _, mapping := range strings.Split(destPorts, ",") {
		parts := strings.Split(mapping, "=")
		if len(parts) > 2 {
			return nil, fmt.Errorf("port must in the form of host=container")
		}

		hostPort := parts[0]
		containerPort := hostPort
		if len(parts) == 2 {
			containerPort = parts[1]
		}

		hostRange := strings.Split(hostPort, "-")

		hostStartString := hostRange[0]
		hostEndString := hostRange[0]
		if len(hostRange) == 2 {
			hostEndString = hostRange[1]
		}

		containerRange := strings.Split(containerPort, "-")

		containerStartString := containerRange[0]

		hostStart, err := strconv.ParseUint(hostStartString, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("port %s is not a valid port number", hostStartString)
		}

		hostEnd, err := strconv.ParseUint(hostEndString, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("port %s is not a valid port number", hostEndString)
		}

		containerStart, err := strconv.ParseUint(containerStartString, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("port %s is not a valid port number", containerStartString)
		}

		fmt.Println(hostStart)
		fmt.Println(hostEnd)
		fmt.Println(containerStart)
		for hostStart <= hostEnd {
			fmt.Println("Listen on")
			fmt.Println(hostStart)
			listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", sourceAddr, hostStart))
			if err != nil {
				return nil, err
			}

			fmt.Println("creating listener to")
			fmt.Println(strconv.FormatUint(containerStart, 10))
			listeners = append(listeners, &ProxyListener{
				destPort: strconv.FormatUint(containerStart, 10),
				destAddr: "unknown:unknown",
				listener: listener,
			})

			hostStart += 1
			containerStart += 1
		}
	}

	return &ProxyServer{
		sourceAddr: sourceAddr,
		listeners:  listeners,
	}, nil
}

func (this *ProxyServer) Start() {
	for _, listener := range this.listeners {
		go listener.start()
	}
}

func (this *ProxyServer) Stop() {
	for _, listener := range this.listeners {
		listener.stop()
	}
}
