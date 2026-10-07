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

var _ = BeforeSuite(func(ctx SpecContext) {
	Expect(startTestCluster(ctx)).To(Succeed())
	Expect(SetupKubernetesUsers(ctx)).To(Succeed())
	_, err := kubectl(ctx, "config", "set-context", "--current", "--namespace=apps")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func(ctx SpecContext) {
	Expect(stopTestCluster(ctx)).To(Succeed())
})
