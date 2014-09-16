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
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

type DockerClient struct {
	dc     *docker.Client
	proxy  *ProxyServer
	tag    string
	events chan *docker.APIEvents
}

func NewDockerClient(address, tag string, proxy *ProxyServer) (*DockerClient, error) {
	dc, err := docker.NewClient(address)
	if err != nil {
		return nil, err
	}

	client := &DockerClient{
		dc:     dc,
		proxy:  proxy,
		tag:    tag,
		events: make(chan *docker.APIEvents),
	}

	if err := dc.AddEventListener(client.events); err != nil {
		return nil, err
	}

	return client, nil
}

// Strip the sub-tag off a container name, and compare it to the base tag we're
// looking for.
func (this *DockerClient) matchTag(tag string) bool {
	components := strings.Split(tag, ":")
	return components[0] == this.tag
}

// Detect any existing containers matching our tag. Kill all but the latest,
// and start proxying to the latest.
func (this *DockerClient) DetectExistingContainers() {
	options := docker.ListContainersOptions{}
	list, err := this.dc.ListContainers(options)
	if err != nil {
		log.Printf("Could not query existing containers: %s\n", err.Error())
		return
	}

	var latest docker.APIContainers

	matching := []docker.APIContainers{}
	for _, container := range list {
		if this.matchTag(container.Image) {
			matching = append(matching, container)
			if container.Created > latest.Created {
				latest = container
			}
		}
	}

	if len(matching) == 0 {
		return
	}

	// Go through and kill off old containers.
	for _, container := range matching {
		if container.ID == latest.ID {
			continue
		}

		log.Printf("Killing off old container: %s\n", container.ID)
		this.dc.KillContainer(docker.KillContainerOptions{
			ID: container.ID,
		})
	}

	// Use the latest container.
	this.containerStarted(latest.ID)
}

// Called when a new container is started.
func (this *DockerClient) onContainerStarted(id string) bool {
	container, err := this.dc.InspectContainer(id)
	if err != nil {
		log.Printf("Could not inspect docker container %s: %s\n", id, err.Error())
		return false
	}

	address := container.NetworkSettings.IPAddress

	log.Printf("Switching to container %s (ip: %s)\n", id, address)

	for _, listener := range this.proxy.listeners {
		listener.reconfigure(address)
	}
	return true
}

func (this *DockerClient) Listen() {
	for {
		event := <-this.events
		if !this.matchTag(event.From) {
			continue
		}

		switch event.Status {
		case "start":
			this.onContainerStarted(event.ID)
		}
	}
}

func main() {
	addressp := flag.String("address", "0.0.0.0", "IP address to listen on")
	portsp := flag.String("ports", "", "Comma-delimited list of host=container ports to listen on")
	dockerp := flag.String("docker", "unix:///var/run/docker.sock", "URL of the Docker host")
	tagp := flag.String("tag", "", "Tag of docker images to watch")
	flag.Parse()

	if *portsp == "" {
		fmt.Fprintf(os.Stderr, "Must specify one or more port mappings.\n")
		os.Exit(1)
	}
	if *tagp == "" {
		fmt.Fprintf(os.Stderr, "Must specify a docker tag.\n")
		os.Exit(1)
	}

	// Create the proxy server and begin listening in the background.
	server, err := NewProxyServer(*addressp, *portsp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not listen on %s: %s\n", *addressp, err.Error())
		os.Exit(1)
	}
	server.Start()
	defer server.Stop()

	dc, err := NewDockerClient(*dockerp, *tagp, server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to docker on %s: %s\n", *dockerp, err.Error())
		os.Exit(1)
	}

	// Try and proxy to anything currently running.
	dc.DetectExistingContainers()

	fmt.Fprintf(os.Stdout, "Listening...\n")
	dc.Listen()
}
