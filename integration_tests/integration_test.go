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
	ctx := context.Background()
	Expect(startTestCluster(ctx)).To(Succeed())
	Expect(SetupKubernetesUsers(ctx)).To(Succeed())
})

var _ = AfterSuite(func() {
	Expect(stopTestCluster(context.Background())).To(Succeed())
	err := DeleteKubernetesUsersFromKubeconfig(context.Background())
	// Even if user deletion fails, we should still continue and delete the cluster, so just print it
	if err != nil {
		fmt.Println(err)
	}

})
