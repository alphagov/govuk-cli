package integration_tests

import (
	"fmt"
	"os"
	"path/filepath"

	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("jobrequest logs", Ordered, func() {
	const namespace = "apps"
	const jobRequestName = "jobrequest-logs-test"
	const reviewName = "jrr-jobrequest-logs-test"
	const jobName = "jobrequest-logs-test-x7k2p"
	const podName = "jobrequest-logs-test-x7k2p-9fh3s"
	const nodeName = "kwok-node-jobrequest-logs-test"

	var job *batchv1.Job
	var jobRequest *jrv1.JobRequest
	var jobRequestReview *jrv1.JobRequestReview
	var pod *corev1.Pod
	var node *corev1.Node

	BeforeEach(func(ctx SpecContext) {
		job = nil
		jobRequest = nil
		jobRequestReview = nil
		node = nil
		pod = nil
	})

	AfterEach(func(ctx SpecContext) {
		Expect(SwitchToKubernetesAdminUser(ctx)).To(Succeed())

		if pod != nil {
			Expect(deletePod(ctx, pod)).To(Succeed())
		}

		if node != nil {
			Expect(deleteNode(ctx, node)).To(Succeed())
		}

		if job != nil {
			Expect(deleteJob(ctx, job)).To(Succeed())
		}

		if jobRequest != nil {
			Expect(deleteJobRequest(ctx, jobRequest)).To(Succeed())
		}

		if jobRequestReview != nil {
			Expect(deleteJobRequestReview(ctx, jobRequestReview)).To(Succeed())
		}

	})

	Context("when the job request does not exist", Ordered, func() {
		It("Errors and tells the user the job request does not exist", func(ctx SpecContext) {
			Expect(SwitchToKubernetesUser(ctx, JobRequesterUser)).To(Succeed())

			cmd, err := cliCmd(
				ctx,
				"jobrequest", "logs",
				"--kubeconfig", kubeconfigPath,
				"--namespace", namespace,
				"no-such-job",
			)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).To(HaveOccurred(), string(output))

			outputString := string(output)

			Expect(outputString).To(Equal("ERROR Job request not found name=no-such-job\n"))
		})
	})

	Context("when the JobRequest exists, but it has not yet been reviewed", Ordered, func() {
		It("gives an error saying the JobRequest has not yet been Reviewed", func(ctx SpecContext) {
			Expect(SwitchToKubernetesUser(ctx, JobRequesterUser)).To(Succeed())

			jobRequest = pendingJobRequest(jobRequestName, namespace, JobRequesterUser.ARN)
			Expect(createJobRequest(ctx, jobRequest)).To(Succeed())

			cmd, err := cliCmd(ctx,
				"jobrequest", "logs", jobRequestName,
				"--kubeconfig", kubeconfigPath,
				"--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func() {
				session.Kill().Wait()
			})
			Expect(session.Wait("2s").Out).Should(gbytes.Say("JobRequest has not yet been reviewed"))
			Expect(session.ExitCode()).To(Equal(1))
		})
	})

	Context("when the JobRequest has been Rejected", Ordered, func() {
		It("gives an error saying the JobRequest was rejected", func(ctx SpecContext) {
			Expect(SwitchToKubernetesUser(ctx, JobRequesterUser)).To(Succeed())

			jobRequest = pendingJobRequest(jobRequestName, namespace, JobRequesterUser.ARN)
			jobRequest.Status.State = jrv1.JobRequestRejected
			Expect(createJobRequest(ctx, jobRequest)).To(Succeed())

			cmd, err := cliCmd(ctx,
				"jobrequest", "logs", jobRequestName,
				"--kubeconfig", kubeconfigPath,
				"--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func() {
				session.Kill().Wait()
			})
			Expect(session.Wait("2s").Out).Should(gbytes.Say("JobRequest was Rejected"))
			Expect(session.ExitCode()).To(Equal(1))
		})
	})

	Context("when the Job has been created and the Pod has logs available", func() {
		BeforeEach(func(ctx SpecContext) {
			logsFile := filepath.Join(GinkgoT().TempDir(), "pod.log")
			logLines := "2026-07-14T15:00:00.000000001Z stdout F migration started\n" +
				"2026-07-14T15:00:00.000000002Z stdout F migration finished\n"
			Expect(os.WriteFile(logsFile, []byte(logLines), 0o644)).To(Succeed())

			node = kwokNode(nodeName)
			Expect(createNode(ctx, node)).To(Succeed())

			// We need to use the IntegrationEnvUser so an OpenSearch URL can be generated
			err := SwitchToKubernetesUser(ctx, IntegrationEnvUser)
			Expect(err).NotTo(HaveOccurred())

			Expect(createPodLogs(ctx, podName, namespace, "app", logsFile)).To(Succeed())

			jobRequest = pendingJobRequest(jobRequestName, namespace, IntegrationEnvUser.ARN)
			jobRequest.Status.State = jrv1.JobRequestStarted
			jobRequest.Status.JobName = jobName
			jobRequest.Status.ReviewName = reviewName
			Expect(createJobRequest(ctx, jobRequest)).To(Succeed())

			jobRequestReview = approvedJobRequestReview(reviewName, namespace, jobRequestName)
			Expect(createJobRequestReview(ctx, jobRequestReview)).To(Succeed())

			job = suspendedJob(jobName, namespace)
			Expect(createJob(ctx, job)).To(Succeed())

			pod = pendingPod(podName, namespace, jobName)
			pod.Spec.NodeName = nodeName
			pod.Spec.Tolerations = []corev1.Toleration{
				{
					Key:      "kwok.x-k8s.io/node",
					Operator: corev1.TolerationOpExists,
				},
			}
			Expect(createPod(ctx, pod)).To(Succeed())
		})

		AfterEach(func(ctx SpecContext) {
			Expect(deletePodLogs(ctx, podName, namespace)).To(Succeed())
		})

		It("when following the logs it streams the pod's logs", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx,
				"jobrequest", "logs", jobRequestName,
				"--follow",
				"--log-level", "debug",
				"--kubeconfig", kubeconfigPath,
				"--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				session.Kill().Wait()
			})
			Expect(setJobStartTime(ctx, job)).To(Succeed())

			Eventually(session.Err, "10s").Should(gbytes.Say("Tailing logs from pod"))
			Eventually(session.Out, "10s").Should(gbytes.Say("migration started"))
			Eventually(session.Out, "10s").Should(gbytes.Say("migration finished"))
		})

		It("when not following the logs prints the pod's logs and also a Logit link", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx,
				"jobrequest", "logs", jobRequestName,
				"--kubeconfig", kubeconfigPath,
				"--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())

			outputString := string(output)

			Expect(outputString).To(
				ContainSubstring(
					fmt.Sprintf(
						"migration started\n"+
							"migration finished\n"+
							"--- LOGS END ---\n"+
							"\n"+
							"Print logs:\n"+
							" $ kubectl -n %s logs -f job/%s\n"+
							"\n"+
							"View logs in OpenSearch:\n",
						namespace,
						jobName,
					),
				),
			)
		})

	})

	Context("when the Job exists but the pod has been garbage collected", func() {
		BeforeEach(func(ctx SpecContext) {
			// We need to use the IntegrationEnvUser so an OpenSearch URL can be generated
			err := SwitchToKubernetesUser(ctx, IntegrationEnvUser)
			Expect(err).NotTo(HaveOccurred())

			jobRequest = pendingJobRequest(jobRequestName, namespace, IntegrationEnvUser.ARN)
			jobRequest.Status.State = jrv1.JobRequestComplete
			jobRequest.Status.JobName = jobName
			jobRequest.Status.ReviewName = reviewName
			Expect(createJobRequest(ctx, jobRequest)).To(Succeed())

			jobRequestReview = approvedJobRequestReview(reviewName, namespace, jobRequestName)
			Expect(createJobRequestReview(ctx, jobRequestReview)).To(Succeed())

			job = completedJob(jobName, namespace)
			Expect(createJob(ctx, job)).To(Succeed())
		})

		It("prints a helpful error and a Logit link", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx,
				"jobrequest", "logs", jobRequestName,
				"--kubeconfig", kubeconfigPath,
				"--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, _ := cmd.CombinedOutput()
			Expect(cmd.ProcessState.ExitCode()).To(Equal(1))

			outputString := string(output)

			Expect(outputString).To(ContainSubstring("either not yet started, or has been garbage collected"))

			Expect(outputString).To(ContainSubstring("View logs in OpenSearch:"))
		})
	})
})
