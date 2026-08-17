package integration_tests

import (
	"context"
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
	Expect(SetupKubernetesUsers(context.Background())).To(Succeed())
})

var _ = AfterSuite(func() {
	DeleteKubernetesUsersFromKubeconfig(context.Background())

	Expect(stopTestCluster()).To(Succeed())
})
