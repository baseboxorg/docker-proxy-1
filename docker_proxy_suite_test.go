package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDockerProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DockerProxy Suite")
}
