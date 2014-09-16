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
	"sync"
)

// Listen on a source port and forward to a destination address and port. The
// destination address can be reconfigured at any time.
type ProxyListener struct {
	destPort   string

	destAddr   string
	listener   net.Listener
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

func (this *ProxyListener) reconfigure(destAddr string) {
	this.destAddr = fmt.Sprintf("%s:%s", destAddr, this.destPort)
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

	cond := sync.NewCond(&sync.Mutex{})
	done := 0
	notify := func() {
		cond.L.Lock()
		defer cond.L.Unlock()
		done++
		cond.Signal()
	}

	// Create two goroutines to asynchronously copy data to/from the source and
	// destination addresses.
	go (func() {
		if _, err = io.Copy(dest, cn); err != nil {
			log.Printf("Failed to copy to destination %s: %s", this.destAddr, err.Error())
		}
		notify()
	})()
	go (func() {
		if _, err = io.Copy(cn, dest); err != nil {
			log.Printf("Failed to copy from destination %s: %s", this.destAddr, err.Error())
		}
		notify()
	})()

	// Wait until we get notifications from both goroutines that the connection
	// is finished.
	cond.L.Lock()
	for done != 2 {
		cond.Wait()
	}
	cond.L.Unlock()

	log.Printf("Connection %s finished", cn.RemoteAddr())
}
