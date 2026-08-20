package integration_tests

import (
	"context"
	"fmt"
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
	err := DeleteKubernetesUsersFromKubeconfig(context.Background())
	// Even if user deletion fails, we should still continue and delete the cluster, so just print it
	if err != nil {
		fmt.Println(err)
	}

	Expect(stopTestCluster()).To(Succeed())
})
