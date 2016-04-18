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
	"net/http"
	"os"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

type DockerClient struct {
	dc            *docker.Client
	proxy         *ProxyServer
	tag           string
	events        chan *docker.APIEvents
	statusURL     string
	statusTimeout time.Duration
	gracePeriod   time.Duration
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

func (this *DockerClient) SetStatusInfo(statusURL string, statusTimeout time.Duration) {
	this.statusURL = statusURL
	this.statusTimeout = statusTimeout
}

func (this *DockerClient) SetGracePeriod(gracePeriod time.Duration) {
	this.gracePeriod = gracePeriod
}

// Strip the sub-tag off a container name, and compare it to the base tag we're
// looking for.
func (this *DockerClient) matchTag(tag string) bool {
	components := strings.Split(tag, ":")
	return components[0] == this.tag
}

// Detect any existing containers matching our tag. Kill all but the latest,
// and start proxying to the latest.
func (this *DockerClient) DetectExistingContainers() error {
	options := docker.ListContainersOptions{}
	list, err := this.dc.ListContainers(options)
	if err != nil {
		return err
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
		return nil
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
	this.onContainerStarted(latest.ID)
	return nil
}

// Called when a new container is started.
func (this *DockerClient) onContainerStarted(id string) bool {
	container, err := this.dc.InspectContainer(id)
	if err != nil {
		log.Printf("Could not inspect docker container %s: %s\n", id, err.Error())
		return false
	}

	address := container.NetworkSettings.IPAddress

	firstContainer := true
        for _, listener := range this.proxy.listeners {
                oldID := listener.reconfigure(id, address)
                if oldID != "" {
			firstContainer = false
                }
        }

	if firstContainer {
		log.Printf("First container came online, routing traffic to it.")
	} else if this.statusURL != "" {
		url := fmt.Sprintf("http://%s%s", address, this.statusURL)
		end := time.Now().Add(this.statusTimeout)
		for {
			if time.Now().After(end) {
				log.Printf("New container came online, but did not respond to status queries.")
				return false
			}
			client := &http.Client{
				Timeout: time.Second,
			}
			response, err := client.Get(url)
			if err == nil {
				response.Body.Close()
				break
			}
			log.Printf("querying status failed: %s", err.Error())
			time.Sleep(time.Second)
		}
	}

	log.Printf("Switching to container %s (ip: %s)\n", id, address)

	oldContainers := map[string]bool{}

	for _, listener := range this.proxy.listeners {
		oldID := listener.reconfigure(id, address)
		if oldID != "" {
			oldContainers[oldID] = true
		}
	}

	for oldID, _ := range oldContainers {
		go func(id string) {
			if this.gracePeriod != 0 {
				log.Printf("Waiting %s to kill container %s...", this.gracePeriod, id)
				time.Sleep(this.gracePeriod)
			}

			log.Printf("Killing old container: %s", id)
			err := this.dc.KillContainer(docker.KillContainerOptions{
				ID: id,
			})
			if err != nil {
				log.Printf("Failed to signal container: %s", err.Error())
			}
		}(oldID)
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
	address := flag.String("address", "0.0.0.0", "IP address to listen on")
	ports := flag.String("ports", "", "Comma-delimited list of host=container ports to listen on")
	docker := flag.String("docker", "unix:///var/run/docker.sock", "URL of the Docker host")
	tag := flag.String("tag", "", "Tag of docker images to watch")
	statusURL := flag.String("status_url", "", "Optional HTTP status URL of docker container, e.g. :80/status")
	statusTimeout := flag.Duration("status_timeout", 10*time.Second, "Time to wait for a new container to respond to a status query")
	gracePeriod := flag.Duration("grace_period", 10*time.Second, "Time to wait before killing an old container")
	flag.Parse()

	if *ports == "" {
		fmt.Fprintf(os.Stderr, "Must specify one or more port mappings.\n")
		flag.Usage()
		os.Exit(1)
	}
	if *tag == "" {
		fmt.Fprintf(os.Stderr, "Must specify a docker tag.\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create the proxy server and begin listening in the background.
	server, err := NewProxyServer(*address, *ports)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not listen on %s: %s\n", *address, err.Error())
		os.Exit(1)
	}
	server.Start()
	defer server.Stop()

	dc, err := NewDockerClient(*docker, *tag, server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to docker on %s: %s\n", *docker, err.Error())
		os.Exit(1)
	}

	if *statusURL != "" {
		dc.SetStatusInfo(*statusURL, *statusTimeout)
	}
	dc.SetGracePeriod(*gracePeriod)

	// Try and proxy to anything currently running.
	if err := dc.DetectExistingContainers(); err != nil {
		fmt.Fprintf(os.Stderr, "Could not query containers: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Listening...\n")
	dc.Listen()
}
