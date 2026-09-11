package integration_tests

import (
	"strings"
	"time"

	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("jobrequest list", func() {
	const namespace = "apps"

	Context("when there are no JobRequests", Ordered, func() {
		BeforeAll(func() {
			err := SwitchToKubernetesUser(JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			err := SwitchToKubernetesAdminUser()
			Expect(err).NotTo(HaveOccurred())
		})

		It("Tells the user there are no job requests", func(ctx SpecContext) {
			cmd, err := cliCmd(
				ctx,
				"jobrequest", "list",
				"--kubeconfig", kubeconfigPath, "--namespace", namespace,
			)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			outputString := string(output)

			Expect(outputString).To(Equal("No JobRequests found in namespace apps\n"))
		})
	})

	Context("when there are only JobRequests which were not created by the user", Ordered, func() {
		BeforeAll(func(ctx SpecContext) {
			jrReviewer1 := pendingJobRequest("jr-reviewer-1", namespace, JobReviewerUser.ARN)
			Expect(createJobRequest(ctx, jrReviewer1)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jrReviewer1)).To(Succeed())
			})

			err := SwitchToKubernetesUser(JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			err := SwitchToKubernetesAdminUser()
			Expect(err).NotTo(HaveOccurred())
		})

		It("Tells the user there are none of their job requests", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx,
				"jobrequest", "list", "--mine",
				"--kubeconfig", kubeconfigPath, "--namespace", namespace,
			)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			outputString := string(output)

			Expect(outputString).To(Equal("No JobRequests found created by you in namespace apps\n"))
		})
	})

	Context("when there are JobRequests", Ordered, func() {
		BeforeAll(func(ctx SpecContext) {
			jrRequester1 := pendingJobRequest("jr-requester-1", namespace, JobRequesterUser.ARN)
			Expect(createJobRequest(ctx, jrRequester1)).To(Succeed())

			// Need to sleep so the creation times differ
			time.Sleep(1100 * time.Millisecond)
			jrReviewer1 := pendingJobRequest("jr-reviewer-1", namespace, JobReviewerUser.ARN)
			jrReviewer1.Status.State = jrv1.JobRequestApproved
			Expect(createJobRequest(ctx, jrReviewer1)).To(Succeed())

			// Need to sleep so the creation times differ
			time.Sleep(1100 * time.Millisecond)
			jrRequester2 := pendingJobRequest("jr-requester-2", namespace, JobRequesterUser.ARN)
			jrRequester2.Status.State = jrv1.JobRequestMalformed
			Expect(createJobRequest(ctx, jrRequester2)).To(Succeed())

			// Need to sleep so the creation times differ
			time.Sleep(1100 * time.Millisecond)
			jrReviewer2 := pendingJobRequest("jr-reviewer-2", namespace, JobReviewerUser.ARN)
			jrReviewer2.Status.State = jrv1.JobRequestComplete
			Expect(createJobRequest(ctx, jrReviewer2)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jrRequester1)).To(Succeed())
				Expect(deleteJobRequest(ctx, jrRequester2)).To(Succeed())
				Expect(deleteJobRequest(ctx, jrReviewer1)).To(Succeed())
				Expect(deleteJobRequest(ctx, jrReviewer2)).To(Succeed())
			})

			err := SwitchToKubernetesUser(JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			err := SwitchToKubernetesAdminUser()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("list is run without filtering", func() {
			var outputString string
			var outputLines []string

			BeforeAll(func(ctx SpecContext) {
				cmd, err := cliCmd(ctx,
					"jobrequest", "list",
					"--kubeconfig", kubeconfigPath, "--namespace", namespace,
				)
				Expect(err).NotTo(HaveOccurred())

				output, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred(), string(output))

				outputString = string(output)
				outputLines = strings.Split(outputString, "\n")
			})

			It("includes the Header row", func() {
				Expect(outputLines[1]).To(MatchRegexp(".*Name.*State.*Created By.*Created Time.*"))
			})

			It("has the columns in the correct order", func() {
				Expect(outputLines[3]).To(MatchRegexp(".*jr-requester-1.*Pending.*job.req.*%d", time.Now().Year()))
			})

			It("Includes all 4 job requests", func() {
				// 4 job requests, 1 table header and 3 table borders, and a newline at the end
				Expect(len(outputLines)).To(Equal(9))
				Expect(outputString).To(ContainSubstring("jr-requester-1"))
				Expect(outputString).To(ContainSubstring("jr-requester-2"))
				Expect(outputString).To(ContainSubstring("jr-reviewer-1"))
				Expect(outputString).To(ContainSubstring("jr-reviewer-2"))
			})

			It("Has correctly ordered the 4 job requests", func() {
				Expect(outputLines[3]).To(ContainSubstring("jr-requester-1"))
				Expect(outputLines[4]).To(ContainSubstring("jr-reviewer-1"))
				Expect(outputLines[5]).To(ContainSubstring("jr-requester-2"))
				Expect(outputLines[6]).To(ContainSubstring("jr-reviewer-2"))
			})

			It("Includes the State", func() {
				Expect(outputLines[3]).To(ContainSubstring("Pending"))
				Expect(outputLines[4]).To(ContainSubstring("Approved"))
				Expect(outputLines[5]).To(ContainSubstring("Malformed"))
				Expect(outputLines[6]).To(ContainSubstring("Complete"))
			})

			It("Includes the creator", func() {
				Expect(outputLines[3]).To(ContainSubstring("job.req"))
				Expect(outputLines[4]).To(ContainSubstring("job.rev"))
				Expect(outputLines[5]).To(ContainSubstring("job.req"))
				Expect(outputLines[6]).To(ContainSubstring("job.rev"))
			})
		})

		Context("list is filtered with --mine", func() {
			var outputString string
			var outputLines []string

			BeforeAll(func(ctx SpecContext) {
				cmd, err := cliCmd(
					ctx,
					"jobrequest", "list", "--mine",
					"--kubeconfig", kubeconfigPath, "--namespace", namespace,
				)
				Expect(err).NotTo(HaveOccurred())

				output, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred(), string(output))

				outputString = string(output)
				outputLines = strings.Split(outputString, "\n")
			})

			It("includes the Header row", func() {
				Expect(outputLines[1]).To(MatchRegexp(".*Name.*State.*Created By.*Created Time.*"))
			})

			It("Includes all 2 job requests", func() {
				// 2 job requests, 1 table header and 3 table borders, and a newline at the end
				Expect(len(outputLines)).To(Equal(7))
				Expect(outputString).To(ContainSubstring("jr-requester-1"))
				Expect(outputString).To(ContainSubstring("jr-requester-2"))
			})

			It("Has correctly ordered the 2 job requests", func() {
				Expect(outputLines[3]).To(ContainSubstring("jr-requester-1"))
				Expect(outputLines[4]).To(ContainSubstring("jr-requester-2"))
			})
		})

		Context("list is run with --pagination-limit 1 and requires multiple paged requests to the k8s API", func() {
			var outputString string
			var outputLines []string

			BeforeAll(func(ctx SpecContext) {
				cmd, err := cliCmd(
					ctx,
					"jobrequest", "list", "--pagination-limit", "1",
					"--kubeconfig", kubeconfigPath, "--namespace", namespace,
				)
				Expect(err).NotTo(HaveOccurred())

				output, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred(), string(output))

				outputString = string(output)
				outputLines = strings.Split(outputString, "\n")
			})

			It("Includes all 4 job requests", func() {
				// 4 job requests, 1 table header and 3 table borders, and a newline at the end
				Expect(len(outputLines)).To(Equal(9))
				Expect(outputString).To(ContainSubstring("jr-requester-1"))
				Expect(outputString).To(ContainSubstring("jr-requester-2"))
				Expect(outputString).To(ContainSubstring("jr-reviewer-1"))
				Expect(outputString).To(ContainSubstring("jr-reviewer-2"))
			})

			It("Has correctly ordered the 4 job requests", func() {
				Expect(outputLines[3]).To(ContainSubstring("jr-requester-1"))
				Expect(outputLines[4]).To(ContainSubstring("jr-reviewer-1"))
				Expect(outputLines[5]).To(ContainSubstring("jr-requester-2"))
				Expect(outputLines[6]).To(ContainSubstring("jr-reviewer-2"))
			})
		})
	})
})
