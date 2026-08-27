package integration_tests

import (
	"strings"

	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("jobrequest review", func() {
	const namespace = "apps"

	AfterEach(func() {
		err := SwitchToKubernetesAdminUser()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when the job request can't be found", func() {
		It("errors", func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobReviewerUser)
			Expect(err).NotTo(HaveOccurred())

			cmd, err := cliCmd(ctx, "jobrequest", "review", "thisjobdoesntexist", "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).To(MatchError("exit status 1"))

			Expect(string(output)).To(ContainSubstring("Job request not found"))
		})
	})

	Context("when an invalid number of arguments are provided", func() {
		It("errors and prints usage", func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobReviewerUser)
			Expect(err).NotTo(HaveOccurred())

			cmd, err := cliCmd(ctx, "jobrequest", "review", "one-job", "another-job", "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).To(MatchError("exit status 1"))

			Expect(string(output)).To(ContainSubstring("Only one argument should be provided"))
			Expect(string(output)).To(ContainSubstring("Usage:"))
		})
	})

	Context("when the job request is not in Pending state", func() {
		const jobRequestName = "review-not-pending"

		BeforeEach(func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())

			jr := pendingJobRequest(jobRequestName, namespace)
			jr.Status.State = jrv1.JobRequestApproved
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})
		})

		It("errors", func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobReviewerUser)
			Expect(err).NotTo(HaveOccurred())

			cmd, err := cliCmd(ctx, "jobrequest", "review", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).To(MatchError("exit status 1"))

			Expect(string(output)).To(ContainSubstring("Only pending job requests are reviewable"))
		})
	})

	Context("when the job request has already been reviewed", func() {
		const jobRequestName = "review-already-reviewed"

		BeforeEach(func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())

			jr := pendingJobRequest(jobRequestName, namespace)
			jr.Status.ReviewName = jobRequestName + "-review"
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})
		})

		It("errors", func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobReviewerUser)
			Expect(err).NotTo(HaveOccurred())

			cmd, err := cliCmd(ctx, "jobrequest", "review", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).To(MatchError("exit status 1"))

			Expect(string(output)).To(ContainSubstring("Job request has already been reviewed"))
		})
	})

	Context("when the job request was also created by the reviewer", func() {
		const jobRequestName = "reviewer-and-requester-the-same"

		It("errors", func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())

			jr := pendingJobRequest(jobRequestName, namespace)
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})

			cmd, err := cliCmd(ctx, "jobrequest", "review", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).To(MatchError("exit status 1"))

			Expect(string(output)).To(ContainSubstring("You cannot review your own JobRequest"))
		})
	})

	// The review command reads the decision, an optional comment and a submit
	// confirmation from stdin, one line each. A trailing "y" confirms the
	// submission so the JobRequestReview is actually created.
	DescribeTable("accepting a decision and its alias",
		func(ctx SpecContext, jobRequestName string, stdin string, expectedDecision jrv1.JobRequestReviewState) {
			err := SwitchToKubernetesUser(JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())

			jr := pendingJobRequest(jobRequestName, namespace)
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})

			reviewName := "jrr-" + jobRequestName
			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequestReview(ctx, &jrv1.JobRequestReview{
					ObjectMeta: metav1.ObjectMeta{Name: reviewName, Namespace: namespace},
				})).To(Succeed())
			})
			err = SwitchToKubernetesUser(JobReviewerUser)
			Expect(err).NotTo(HaveOccurred())

			cmd, err := cliCmd(ctx, "jobrequest", "review", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())
			cmd.Stdin = strings.NewReader(stdin)

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring("Reviewed job request"))

			jrr, err := getJobRequestReview(ctx, reviewName, namespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(jrr.Spec.JobRequestName).To(Equal(jobRequestName))
			Expect(jrr.Spec.Decision).To(Equal(string(expectedDecision)))
		},
		Entry("accepts 'a' to approve", "review-approve-short", "a\nlooks good\ny\n", jrv1.JobRequestReviewApproved),
		Entry("accepts 'approve' to approve", "review-approve-long", "approve\nlooks good\ny\n", jrv1.JobRequestReviewApproved),
		Entry("accepts 'r' to reject", "review-reject-short", "r\nnot now\ny\n", jrv1.JobRequestReviewRejected),
		Entry("accepts 'reject' to reject", "review-reject-long", "reject\nnot now\ny\n", jrv1.JobRequestReviewRejected),
	)

	Context("when an invalid decision is entered", func() {
		const jobRequestName = "review-invalid-decision"

		BeforeEach(func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())

			jr := pendingJobRequest(jobRequestName, namespace)
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequestReview(ctx, &jrv1.JobRequestReview{
					ObjectMeta: metav1.ObjectMeta{Name: "jrr-" + jobRequestName, Namespace: namespace},
				})).To(Succeed())
			})
		})

		It("prompts again and accepts a subsequent valid decision", func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobReviewerUser)
			Expect(err).NotTo(HaveOccurred())

			cmd, err := cliCmd(ctx, "jobrequest", "review", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())
			cmd.Stdin = strings.NewReader("maybe\napprove\nlooks good\ny\n")

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring("Only 'approve' and 'reject' are valid decisions"))
			Expect(string(output)).To(ContainSubstring("Reviewed job request"))

			jrr, err := getJobRequestReview(ctx, "jrr-"+jobRequestName, namespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(jrr.Spec.Decision).To(Equal(string(jrv1.JobRequestReviewApproved)))
		})
	})

	Context("when the submit confirmation is declined", func() {
		const jobRequestName = "review-submit-declined"

		BeforeEach(func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())

			jr := pendingJobRequest(jobRequestName, namespace)
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})
		})

		It("exits without creating a review", func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobReviewerUser)
			Expect(err).NotTo(HaveOccurred())

			cmd, err := cliCmd(ctx, "jobrequest", "review", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())
			cmd.Stdin = strings.NewReader("a\nlooks good\nn\n")

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).NotTo(ContainSubstring("Reviewed job request"))

			_, err = getJobRequestReview(ctx, "jrr-"+jobRequestName, namespace)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when an invalid submit confirmation is entered", func() {
		const jobRequestName = "review-submit-invalid"

		BeforeEach(func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())

			jr := pendingJobRequest(jobRequestName, namespace)
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequestReview(ctx, &jrv1.JobRequestReview{
					ObjectMeta: metav1.ObjectMeta{Name: "jrr-" + jobRequestName, Namespace: namespace},
				})).To(Succeed())
			})
		})

		It("prompts again and accepts a subsequent valid confirmation", func(ctx SpecContext) {
			err := SwitchToKubernetesUser(JobReviewerUser)
			Expect(err).NotTo(HaveOccurred())

			cmd, err := cliCmd(ctx, "jobrequest", "review", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())
			cmd.Stdin = strings.NewReader("a\nlooks good\nmaybe\ny\n")

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring("Only 'yes' and 'no' are valid submission decision options"))
			Expect(string(output)).To(ContainSubstring("Reviewed job request"))

			jrr, err := getJobRequestReview(ctx, "jrr-"+jobRequestName, namespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(jrr.Spec.Decision).To(Equal(string(jrv1.JobRequestReviewApproved)))
		})
	})
})
