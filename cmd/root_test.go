package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// This is a placeholder test which can be replaced when we have real functionality
var _ = Describe("rootCmd", func() {
	Context("when called with no args", func() {
		It("Exits without an error", func() {
			err := rootCmd.Execute()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
