# vim: set ts=4 sw=4 tw=99 noet:
include $(GOPATH)/src/github.com/edmodo/minion/rules.mk

export SERVICE_NAME=proxyserver
export CONFIG_FILES=

build_docker_with_binary: deployment_setup run_docker_build

build_docker: install build_docker_with_binary
