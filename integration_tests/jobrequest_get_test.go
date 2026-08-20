package integration_tests

import (
	"fmt"

	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// pendingJobRequest returns a minimal valid JobRequest fixture in Pending
// state that individual specs can adjust before creating.
func pendingJobRequest(name string, namespace string) *jrv1.JobRequest {
	return &jrv1.JobRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				jrv1.JobRequestRequestedByAnnotation: JobRequesterUser.ARN,
			},
		},
		Spec: jrv1.JobRequestSpec{
			ContainerFrom: jrv1.JobRequestContainerFrom{
				PodSpecFrom: jrv1.JobRequestPodSpecFrom{
					Group: "apps/v1",
					Kind:  "Deployment",
					Name:  "publishing-api",
				},
				ContainerName: "app",
			},
			Command: "echo",
			Args:    []string{"hello"},
		},
		Status: jrv1.JobRequestStatus{
			State: jrv1.JobRequestPending,
		},
	}
}

// approvedJobRequestReview returns a minimal valid JobRequestReview fixture
// with an Approved decision that individual specs can adjust before creating.
func approvedJobRequestReview(name string, namespace string, jobRequestName string) *jrv1.JobRequestReview {
	return &jrv1.JobRequestReview{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				jrv1.JobRequestReviewReviewedByAnnotation: JobReviewerUser.ARN,
			},
		},
		Spec: jrv1.JobRequestReviewSpec{
			JobRequestName: jobRequestName,
			Decision:       string(jrv1.JobRequestReviewApproved),
		},
	}
}

var _ = Describe("jobrequest get", func() {
	Context("when the JobRequest is in Pending state", func() {
		const jobRequestName = "run-db-migration"
		const namespace = "default"

		BeforeEach(func(ctx SpecContext) {
			jr := pendingJobRequest(jobRequestName, namespace)
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})
		})

		It("prints the job request details", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring(jobRequestName))
			Expect(string(output)).To(ContainSubstring("echo hello"))
			Expect(string(output)).To(ContainSubstring(string(jrv1.JobRequestPending)))
			Expect(string(output)).To(ContainSubstring(fmt.Sprintf("%s (%s)", JobRequesterUser.Name, JobRequesterUser.Role)))
			Expect(string(output)).To(ContainSubstring("deployment/publishing-api"))
			Expect(string(output)).ToNot(ContainSubstring("Print logs:"))
			Expect(string(output)).ToNot(ContainSubstring("Get review:"))
			Expect(string(output)).ToNot(ContainSubstring("Job Name"))
		})

		It("errors when a job request can't be found", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", "thisjobdoesntexist", "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).To(MatchError("exit status 1"))

			Expect(string(output)).To(ContainSubstring("Job request not found"))
		})
	})

	Context("when the requester ARN is invalid", func() {
		const jobRequestName = "invalid-requester-arn"
		const namespace = "default"

		BeforeEach(func(ctx SpecContext) {
			jr := pendingJobRequest(jobRequestName, namespace)
			jr.Annotations[jrv1.JobRequestRequestedByAnnotation] = "not-a-valid-arn"
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})
		})

		It("prints the raw annotation value and logs a parse error", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring("Error parsing requester ARN"))
			Expect(string(output)).To(ContainSubstring("not-a-valid-arn"))
		})
	})

	Context("when an invalid number of arguments are passed", func() {
		It("errors and prints usage", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", "one-job", "another-job", "--kubeconfig", kubeconfigPath, "--namespace", "default")
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).To(MatchError("exit status 1"))

			Expect(string(output)).To(ContainSubstring("Only one argument should be provided"))
			Expect(string(output)).To(ContainSubstring("Usage:"))
		})
	})

	Context("when the requested-by annotation isn't present", func() {
		const jobRequestName = "no-requested-by-annotation"
		const namespace = "default"

		BeforeEach(func(ctx SpecContext) {
			jr := pendingJobRequest(jobRequestName, namespace)
			jr.Annotations = nil
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})
		})

		It("prints a placeholder for the requester", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(MatchRegexp(`Requested By\s*│\s*-\s*│`))
		})
	})

	Context("when the status state is an empty string", func() {
		const jobRequestName = "empty-status-state"
		const namespace = "default"

		BeforeEach(func(ctx SpecContext) {
			jr := pendingJobRequest(jobRequestName, namespace)
			jr.Status.State = ""
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})
		})

		It("prints the status as Unknown", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(MatchRegexp(`Status\s*│\s*Unknown\s*│`))
		})
	})

	Context("when the JobRequest has a job name in its status", func() {
		const jobRequestName = "has-job-name"
		const jobName = "has-job-name-x7k2p"
		const namespace = "default"

		BeforeEach(func(ctx SpecContext) {
			jr := pendingJobRequest(jobRequestName, namespace)
			jr.Status.State = jrv1.JobRequestStarted
			jr.Status.JobName = jobName
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})
		})

		It("prints the kubectl logs command", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring("Print logs:"))
			Expect(string(output)).To(ContainSubstring("$ kubectl -n default logs -f job/" + jobName))
		})

		It("includes the job name in the output", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(MatchRegexp(`Job Name[^\r\n]+%s`, jobName))
		})
	})

	Context("when the JobRequest references a review that doesn't exist", func() {
		const jobRequestName = "review-not-found"
		const reviewName = "review-not-found-review"
		const namespace = "default"

		BeforeEach(func(ctx SpecContext) {
			jr := pendingJobRequest(jobRequestName, namespace)
			jr.Status.ReviewName = reviewName
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})
		})

		It("prints an error placeholder for the reviewer", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring("[Error getting JobRequestReview]"))
		})

		It("prints the kubectl command to get the review", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring("Get review:"))
			Expect(string(output)).To(ContainSubstring("$ kubectl -n default get jobrequestreview " + reviewName + " -o yaml"))
		})
	})

	Context("when the review has no reviewed-by annotation", func() {
		const jobRequestName = "review-no-annotation"
		const reviewName = "review-no-annotation-review"
		const namespace = "default"

		BeforeEach(func(ctx SpecContext) {
			jr := pendingJobRequest(jobRequestName, namespace)
			jr.Status.ReviewName = reviewName
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			jrr := approvedJobRequestReview(reviewName, namespace, jobRequestName)
			jrr.Annotations = nil
			Expect(createJobRequestReview(ctx, jrr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
				Expect(deleteJobRequestReview(ctx, jrr)).To(Succeed())
			})
		})

		It("prints a placeholder for the reviewer", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring("[Review has no reviewed-by annotation]"))
		})
	})

	Context("when the review has a valid reviewed-by ARN", func() {
		const jobRequestName = "review-valid-arn"
		const reviewName = "review-valid-arn-review"
		const namespace = "default"

		BeforeEach(func(ctx SpecContext) {
			jr := pendingJobRequest(jobRequestName, namespace)
			jr.Status.ReviewName = reviewName
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			jrr := approvedJobRequestReview(reviewName, namespace, jobRequestName)
			Expect(createJobRequestReview(ctx, jrr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
				Expect(deleteJobRequestReview(ctx, jrr)).To(Succeed())
			})
		})

		It("prints the reviewer and the review decision", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring(fmt.Sprintf("%s (%s)", JobReviewerUser.Name, JobReviewerUser.Role)))
			Expect(string(output)).To(MatchRegexp(`Review Decision\s*│\s*%s\s*│`, jrv1.JobRequestReviewApproved))
		})
	})

	Context("when the review has an invalid reviewed-by ARN", func() {
		const jobRequestName = "review-invalid-arn"
		const reviewName = "review-invalid-arn-review"
		const namespace = "default"

		BeforeEach(func(ctx SpecContext) {
			jr := pendingJobRequest(jobRequestName, namespace)
			jr.Status.ReviewName = reviewName
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			jrr := approvedJobRequestReview(reviewName, namespace, jobRequestName)
			jrr.Annotations[jrv1.JobRequestReviewReviewedByAnnotation] = "not-a-valid-arn"
			Expect(createJobRequestReview(ctx, jrr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
				Expect(deleteJobRequestReview(ctx, jrr)).To(Succeed())
			})
		})

		It("prints an ARN parse error placeholder for the reviewer", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", jobRequestName, "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring("[Error parsing reviewer's role ARN]"))
		})
	})
})
