package integration_tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("jobrequest namespace autocompletion", Ordered, func() {
	It("provides auto complete suggestions for filenames", func(ctx SpecContext) {
		cliCmd, err := completionCliCmd(ctx, "jobrequest", "--kubeconfig", "")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveDefault"))
	})
})
