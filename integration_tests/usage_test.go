package integration_tests

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Usage", func() {
	Context("when executed with no arguments", func() {
		It("exits successfully", func() {
			cmd, err := cliCmd(context.Background())
			Expect(err).NotTo(HaveOccurred())

			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
