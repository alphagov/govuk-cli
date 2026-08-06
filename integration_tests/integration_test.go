package integration_tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEverything(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration test suite")
}

var _ = BeforeSuite(func() {
	Expect(startTestCluster()).To(Succeed())
})

var _ = AfterSuite(func() {
	Expect(stopTestCluster()).To(Succeed())
})
