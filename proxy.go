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

func parsePortRange(portRange string) (uint64, uint64, error) {
	parts := strings.Split(portRange, "-")
	if len(parts) > 2 {
		return 0, 0, fmt.Errorf("port range must in the form of lo-hi, got %s", portRange)
	}

	lowerBound, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return 0, 0, fmt.Errorf("port range lower bound %s is not a valid port number", lowerBound)
	}

	upperBound, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return 0, 0, fmt.Errorf("port range upper bound %s is not a valid port number", upperBound)
	}

	if upperBound < lowerBound {
		return 0, 0, fmt.Errorf("container port range %s is invalid, %d <= %d", portRange, upperBound, lowerBound)
	}

	return lowerBound, upperBound, nil
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

		hostPortRange := parts[0]
		containerPortRange := hostPortRange
		if len(parts) == 2 {
			containerPortRange = parts[1]
		}

		hostPortLowerBound, hostPortUpperBound, err := parsePortRange(hostPortRange)
		if err != nil {
			return nil, err
		}

		containerPortLowerBound, containerPortUpperBound, err := parsePortRange(containerPortRange)
		if err != nil {
			return nil, err
		}

		if hostPortUpperBound - hostPortLowerBound != containerPortUpperBound - containerPortLowerBound {
			return nil, fmt.Errorf("port ranges %s and %s must be the same size", hostPortRange, containerPortRange)
		}

		for offset := uint64(0); offset <= hostPortUpperBound - hostPortLowerBound; offset++ {
			listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", sourceAddr, hostPortLowerBound + offset))
			if err != nil {
				return nil, err
			}

			listeners = append(listeners, &ProxyListener{
				destPort: fmt.Sprintf("%d", containerPortLowerBound + offset),
				destAddr: "unknown:unknown",
				listener: listener,
			})
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
