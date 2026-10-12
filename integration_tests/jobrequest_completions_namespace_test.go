package integration_tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("jobrequest namespace autocompletion", Ordered, func() {
	// These are purposefully out of order
	namespaces := []string{
		"baz",
		"foo",
		"quux",
		"bar",
		"quz",
	}

	BeforeAll(func(ctx SpecContext) {
		By("Creating test namespaces")
		for _, namespace := range namespaces {
			_, err := kubectl(ctx, "create", "namespace", namespace)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	AfterAll(func(ctx SpecContext) {
		By("Deleting test namespaces")
		for _, namespace := range namespaces {
			_, err := kubectl(ctx, "delete", "namespace", namespace)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("completes all namespaces when none are specified", func(ctx SpecContext) {
		cliCmd, err := completionCliCmd(ctx, "jobrequest", "-n", "")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
		Expect(completionResult.Suggestions).To(ContainElements(namespaces))
	})

	It("completes all namespaces when responses are paginated", func(ctx SpecContext) {
		cliCmd, err := completionCliCmd(ctx, "jobrequest", "--pagination-limit", "2", "-n", "")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
		Expect(completionResult.Suggestions).To(ContainElements(namespaces))
	})

	It("generates completions based on the partial value already typed", func(ctx SpecContext) {
		cliCmd, err := completionCliCmd(ctx, "jobrequest", "-n", "qu")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
		Expect(completionResult.Suggestions).To(ContainElements([]string{"quux", "quz"}))
	})

	It("does not generate any suggestions when the user is not authenticated", func(ctx SpecContext) {
		cliCmd, err := cliCmd(ctx, "__complete", "jobrequest", "-n", "")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveError"))
		Expect(completionResult.Suggestions).To(HaveLen(0))

	})
})
