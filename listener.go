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
	"io"
	"log"
	"net"
)

// Listen on a source port and forward to a destination address and port. The
// destination address can be reconfigured at any time.
type ProxyListener struct {
	destPort string

	containerID string
	destAddr    string
	listener    net.Listener
}

func (this *ProxyListener) start() {
	for {
		cn, err := this.listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %s", err.Error())
			continue
		}
		go this.handleConnection(cn)
	}
}

func (this *ProxyListener) stop() {
	this.listener.Close()
}

// Change the address we proxy to, and return the old container ID.
func (this *ProxyListener) reconfigure(containerID, destAddr string) string {
	oldID := this.containerID
	this.containerID = containerID
	this.destAddr = fmt.Sprintf("%s:%s", destAddr, this.destPort)
	return oldID
}

type Message struct {
	err error
	id  string
}

func (this *ProxyListener) handleConnection(cn net.Conn) {
	defer cn.Close()

	dest, err := net.Dial("tcp", this.destAddr)
	if err != nil {
		log.Printf("Failed to dial destination address %s: %s", this.destAddr, err.Error())
		return
	}
	defer dest.Close()

	log.Printf("Accepted %s to forward to %s", cn.RemoteAddr(), this.destAddr)

	notify := make(chan Message, 2)

	// Create two goroutines to asynchronously copy data to/from the source and
	// destination addresses.
	go (func() {
		_, err := io.Copy(dest, cn)
		notify <- Message{err, "out"}
	})()
	go (func() {
		_, err := io.Copy(cn, dest)
		notify <- Message{err, "in"}
	})()

	// Wait until at least one copy has finished.
	msg := <-notify
	if msg.err != nil {
		log.Printf("Failed to marshal %s traffic: %s", msg.id, msg.err.Error())
	}

	// Hang up both connections, then wait for the other goroutine to finish.
	// It will error since we've closed the connection, so we just discard the
	// error.
	cn.Close()
	dest.Close()
	<-notify
}
