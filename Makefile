build:
	docker build -f docker/Dockerfile.compile -t localhost/docker-proxy-build .
	docker run -v $(shell pwd)/bin:/volume/bin/ localhost/docker-proxy-build cp /go/bin/app /volume/bin/docker-proxy
	docker build -f docker/Dockerfile.build -t registry.edmodo.io/docker-proxy .

run:
	docker run -t registry.edmodo.io/docker-proxy

push:
	docker push registry.edmodo.io/docker-proxy
