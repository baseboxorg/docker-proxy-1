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
	"strings"
)

// Maintain a set of listeners on different ports.
type ProxyServer struct {
	sourceAddr string
	listeners  map[string]*ProxyListener
}

func NewProxyServer(sourceAddr, destPorts string) (*ProxyServer, error) {
	listeners := map[string]*ProxyListener{}
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

		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", sourceAddr, hostPort))
		if err != nil {
			return nil, err
		}

		portName := fmt.Sprintf("%s/tcp", containerPort)
		listeners[portName] = &ProxyListener{
			destPort: containerPort,
			destAddr: "unknown:unknown",
			listener: listener,
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
