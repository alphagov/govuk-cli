package integration_tests

import (
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type jobRequestCompletionTest struct {
	Name      string
	State     jrv1.JobRequestState
	Requester *ClusterUser
	Namespace string
}

var _ = Describe("jobrequest create deployment autocompletion", Ordered, func() {
	appsNamespace := "apps"
	otherNamespace := "other-namespace"
	jobRequestTests := []*jobRequestCompletionTest{
		{Name: "baz", State: jrv1.JobRequestComplete, Requester: JobRequesterUser, Namespace: appsNamespace},
		{Name: "foo", State: jrv1.JobRequestApproved, Requester: JobRequesterUser, Namespace: appsNamespace},
		{Name: "quux", State: jrv1.JobRequestPending, Requester: JobReviewerUser, Namespace: appsNamespace},
		{Name: "bar", State: jrv1.JobRequestPending, Requester: JobRequesterUser, Namespace: appsNamespace},
		{Name: "quz", State: jrv1.JobRequestPending, Requester: JobRequesterUser, Namespace: appsNamespace},
		{Name: "boo", State: jrv1.JobRequestPending, Requester: JobRequesterUser, Namespace: appsNamespace},
		{Name: "bat", State: jrv1.JobRequestPending, Requester: JobRequesterUser, Namespace: otherNamespace},
	}
	jobRequestNamesInAppsNamespace := []string{
		"baz",
		"foo",
		"quux",
		"bar",
		"quz",
		"boo",
	}

	var jobRequests = []*jrv1.JobRequest{}

	BeforeAll(func(ctx SpecContext) {
		By("Creating test JobRequests")

		for _, jobRequestTest := range jobRequestTests {
			jobRequest := pendingJobRequest(jobRequestTest.Name, jobRequestTest.Namespace, jobRequestTest.Requester.ARN)
			Expect(createJobRequest(ctx, jobRequest)).To(Succeed())
			jobRequests = append(jobRequests, jobRequest)
		}

		Expect(SwitchToKubernetesUser(ctx, JobReviewerUser)).To(Succeed())
	})

	AfterAll(func(ctx SpecContext) {
		Expect(SwitchToKubernetesAdminUser(ctx)).To(Succeed())

		By("Deleting test JobRequests")
		for _, jobRequest := range jobRequests {
			Expect(deleteJobRequest(ctx, jobRequest)).To(Succeed())
		}
	})

	Describe("jobrequest get command", func() {
		It("completes all jobRequests when none are specified", func(ctx SpecContext) {
			cliCmd, err := completionCliCmd(ctx, "jobrequest", "get", "")
			Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

			completionResult, err := getCompletionResult(cliCmd)
			Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

			Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
			Expect(completionResult.Suggestions).To(ContainElements(jobRequestNamesInAppsNamespace))
		})

		It("completes all jobrequests when responses are paginated", func(ctx SpecContext) {
			cliCmd, err := completionCliCmd(ctx, "jobrequest", "get", "--pagination-limit", "2", "")
			Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

			completionResult, err := getCompletionResult(cliCmd)
			Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

			Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
			Expect(completionResult.Suggestions).To(ContainElements(jobRequestNamesInAppsNamespace))
		})

		It("generates completions based on the partial value already typed", func(ctx SpecContext) {
			cliCmd, err := completionCliCmd(ctx, "jobrequest", "get", "qu")
			Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

			completionResult, err := getCompletionResult(cliCmd)
			Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

			Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
			Expect(completionResult.Suggestions).To(ContainElements("quz", "quux"))
			Expect(completionResult.Suggestions).NotTo(ContainElements("bar"))
		})

		It("respects the namespace flag and lists jobrequests in the namespace specified if already in the command", func(ctx SpecContext) {
			cliCmd, err := completionCliCmd(ctx, "jobrequest", "get", "-n", otherNamespace, "")
			Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

			completionResult, err := getCompletionResult(cliCmd)
			Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

			Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
			Expect(completionResult.Suggestions).To(ContainElements("bat"))
			Expect(completionResult.Suggestions).NotTo(ContainElements(jobRequestNamesInAppsNamespace))
		})

		DescribeTable(
			"completes deployment names including the prefix when a valid kubernetes JobRequest prefix is used",
			func(ctx SpecContext, prefix string) {
				jobRequestNamesWithPrefix := make([]string, len(jobRequestNamesInAppsNamespace))
				for i, jobRequestName := range jobRequestNamesInAppsNamespace {
					jobRequestNamesWithPrefix[i] = prefix + jobRequestName
				}

				cliCmd, err := completionCliCmd(ctx, "jobrequest", "get", prefix)
				Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

				completionResult, err := getCompletionResult(cliCmd)
				Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

				Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
				Expect(completionResult.Suggestions).To(ContainElements(jobRequestNamesWithPrefix))
			},
			Entry("with jobrequest/ as a prefix", "jobrequest/"),
			Entry("with jobrequests/ as a prefix", "jobrequests/"),
			Entry("with jr/ as a prefix", "jr/"),
		)

		It("does not generate any suggestions when the user is not authenticated", func(ctx SpecContext) {
			cliCmd, err := cliCmd(ctx, "__complete", "jobrequest", "get", "")
			Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

			completionResult, err := getCompletionResult(cliCmd)
			Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

			Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveError"))
			Expect(completionResult.Suggestions).To(HaveLen(0))

		})
	})

	Describe("jobrequest review command", func() {
		reviewableJobRequests := []string{
			"bar",
			"quz",
			"boo",
		}
		unreviewableJobRequests := []string{
			"quux",
			"baz",
			"foo",
			"bat",
		}

		It("completes only reviewable jobRequests when none are specified", func(ctx SpecContext) {
			cliCmd, err := completionCliCmd(ctx, "jobrequest", "review", "")
			Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

			completionResult, err := getCompletionResult(cliCmd)
			Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

			Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
			Expect(completionResult.Suggestions).To(ContainElements(reviewableJobRequests))
			Expect(completionResult.Suggestions).NotTo(ContainElements(unreviewableJobRequests))
		})

		It("completes all jobrequests when responses are paginated", func(ctx SpecContext) {
			cliCmd, err := completionCliCmd(ctx, "jobrequest", "review", "--pagination-limit", "2", "")
			Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

			completionResult, err := getCompletionResult(cliCmd)
			Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

			Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
			Expect(completionResult.Suggestions).To(ContainElements(reviewableJobRequests))
			Expect(completionResult.Suggestions).NotTo(ContainElements(unreviewableJobRequests))
		})

		It("generates completions based on the partial value already typed", func(ctx SpecContext) {
			cliCmd, err := completionCliCmd(ctx, "jobrequest", "review", "b")
			Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

			completionResult, err := getCompletionResult(cliCmd)
			Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

			Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
			Expect(completionResult.Suggestions).To(ContainElements("bar", "boo"))
			Expect(completionResult.Suggestions).NotTo(ContainElements("quz"))
			Expect(completionResult.Suggestions).NotTo(ContainElements(unreviewableJobRequests))
		})

		It("respects the namespace flag and lists jobrequests in the namespace specified if already in the command", func(ctx SpecContext) {
			cliCmd, err := completionCliCmd(ctx, "jobrequest", "review", "-n", otherNamespace, "")
			Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

			completionResult, err := getCompletionResult(cliCmd)
			Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

			Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
			Expect(completionResult.Suggestions).To(ContainElements("bat"))
			Expect(completionResult.Suggestions).NotTo(ContainElements(reviewableJobRequests))
		})

		DescribeTable(
			"completes deployment names including the prefix when a valid kubernetes JobRequest prefix is used",
			func(ctx SpecContext, prefix string) {
				jobRequestNamesWithPrefix := make([]string, len(reviewableJobRequests))
				for i, jobRequestName := range reviewableJobRequests {
					jobRequestNamesWithPrefix[i] = prefix + jobRequestName
				}

				cliCmd, err := completionCliCmd(ctx, "jobrequest", "review", prefix)
				Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

				completionResult, err := getCompletionResult(cliCmd)
				Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

				Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
				Expect(completionResult.Suggestions).To(ContainElements(jobRequestNamesWithPrefix))
			},
			Entry("with jobrequest/ as a prefix", "jobrequest/"),
			Entry("with jobrequests/ as a prefix", "jobrequests/"),
			Entry("with jr/ as a prefix", "jr/"),
		)

		It("does not generate any suggestions when the user is not authenticated", func(ctx SpecContext) {
			cliCmd, err := cliCmd(ctx, "__complete", "jobrequest", "review", "")
			Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

			completionResult, err := getCompletionResult(cliCmd)
			Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

			Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveError"))
			Expect(completionResult.Suggestions).To(HaveLen(0))

		})
	})
})
