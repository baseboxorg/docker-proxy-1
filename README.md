docker-proxy
============

Zero-downtime TCP-proxy tool for Docker

## Overview

When deploying docker containers, we do not want to sever active connections, nor do we want to leave the service unreachable while one container is starting up and another is shutting down. There are a number of ways of working around this; docker-proxy takes a fairly simple and unsophisticated approach that suffices for many services.

Docker-proxy works by opening ports on the external network and proxying traffic to Docker containers. If a new instance of the container is started, the proxy immediately begins forwarding new connections to that container. Old containers are forcefully killed after a grace period.

Although the current behavior is simple, it affords a few advantages. Containers do not have to worry about exposing ports; they can be configured once in the proxy and then ignored thereafter. The proxy also ensures that only one "copy" of a container is running at a time. The proxy itself can also be Docker-ized.

## Installation

docker-proxy requires Go 1.3.1, which you can download from http://golang.org/dl/. Installation looks something like:

```
go get github.com/edmodo/docker-proxy
cd $GOPATH/src/github.com/edmodo/docker-proxy
go install
```

The binary will be `$GOPATH/docker-proxy`. (Note that the Makefile is currently for internal use so it doesn't do anything.)

## Parameters
 - `-address` - The address the proxy server will listen to. Default is 0.0.0.0.
 - `-docker` - URL for the Docker Remote API. Default is `unix:///var/run/docker.sock`.
 - `-ports` - Specification for how external ports map onto containers. This is a comma-delimited list of mappings, where each mapping is either a single port, or a port assignment like `80=3000`. For example: `80,443=3000` would indicate that port 80 maps to container:80, and port 443 maps to container:3000.
 - `-status_url` - A URL to query the container (via HTTP) to see whether it is ready to respond to requests. It can return anything, as long as a `200` response indicates successful startup. Once the proxy gets a `200` it accepts the container as ready. The URL should be specified as a port and URI, for example, `:80/server-status`. The proxy will pretend the container IP when querying. If no status URL is given, new containers will immediately begin receiving requests.
 - `-status_timeout` - The amount of time the proxy should spend attempting to query a new container's status URL. If the time is exceeded, the container will be ignored. The default is `10s` (ten seconds).
 - `-tag` - Tag for containers that the proxy will watch for. When the proxy starts, it will kill all but the most recent container with this tag.
 - `-grace_period` - The amount of time to wait before terminating old containers. The default is `10s` (ten seconds). For example, once the proxy has determined that a new container is ready, it will send a `docker rm -f` command to the old container.

## As a Container

When running docker-proxy inside a container, take care to expose its ports and map the Docker Remote API URL as a volume. For example, if you are exposing port 80 via the proxy, you might need something like: `docker run -p 80 -e -v /var/run/docker.sock:/var/run/docker.sock ...`.

A sample startup script is provided in `config/deploy/start.sh.erb` (it is an ERB template) for making generic docker-proxy containers. This script allows the container to be configured at runtime rather than have its settings baked into an image.
