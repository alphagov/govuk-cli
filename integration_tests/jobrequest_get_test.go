package integration_tests

import (
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("jobrequest get", func() {
	Context("when the JobRequest is in Pending state", func() {
		const jobRequestName = "run-db-migration"
		const namespace = "default"

		BeforeEach(func(ctx SpecContext) {
			jr := &jrv1.JobRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobRequestName,
					Namespace: namespace,
					Annotations: map[string]string{
						"platform.publishing.service.gov.uk/requested-by": "arn:aws:sts::1234:assumed-role/some.user-platformengineer/test-platformengineer",
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
					State: "Pending",
				},
			}
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
			Expect(string(output)).To(ContainSubstring("Pending"))
			Expect(string(output)).To(ContainSubstring("some.user (platformengineer)"))
		})

		It("errors when a job request can't be found", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "get", "thisjobdoesntexist", "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).To(MatchError("exit status 1"))

			Expect(string(output)).To(ContainSubstring("job request not found"))
		})
	})
})
