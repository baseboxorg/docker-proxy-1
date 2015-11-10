package main_test

import (
	. "github.com/edmodo/docker-proxy"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProxyServer", func() {
	It("can create a NewProxyServer", func() {
		ps, err := NewProxyServer("0.0.0.0", "8000=80")
		Expect(err).ToNot(HaveOccurred())

		Expect(ps.Listeners()).To(HaveLen(1))
		ps.Stop()
	})

	It("can create listeners for comma separated ports", func() {
		ps, err := NewProxyServer("0.0.0.0", "8000=80,9000=90")
		Expect(err).ToNot(HaveOccurred())

		Expect(ps.Listeners()).To(HaveLen(2))
		ps.Stop()
	})

	It("can create listeners with container port defaulting to host port", func() {
		ps, err := NewProxyServer("0.0.0.0", "8000,9000")
		Expect(err).ToNot(HaveOccurred())

		Expect(ps.Listeners()).To(HaveLen(2))
		ps.Stop()
	})

	It("can create listeners for a range of ports", func() {
		ps, err := NewProxyServer("0.0.0.0", "8000-8009=80-89")
		Expect(err).ToNot(HaveOccurred())

		Expect(ps.Listeners()).To(HaveLen(10))
		ps.Stop()
	})
})
