package integration_tests

import (
	// "strings"
	// "time"
	"fmt"
	"os"
	"path/filepath"

	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("jobrequest list", func() {
	const namespace = "apps"

	Context("when the job request does not exist", Ordered, func() {
		BeforeAll(func(ctx SpecContext) {
			err := SwitchToKubernetesUser(ctx, JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func(ctx SpecContext) {
			err := SwitchToKubernetesAdminUser(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Errors and tells the user the job request does not exist", func(ctx SpecContext) {
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

	Context("when the Pod has logs available", func() {
		const jobRequestName = "follow-existing-logs"
		const jobName = "follow-existing-logs-x7k2p"
		const podName = "follow-existing-logs-x7k2p-9fh3s"
		const nodeName = "kwok-node-existing-logs"
		const namespace = "apps"

		AfterEach(func(ctx SpecContext) {
			err := SwitchToKubernetesAdminUser(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("streams the pod's logs", func(ctx SpecContext) {
			logsFile := filepath.Join(GinkgoT().TempDir(), "pod.log")
			logLines := "2026-07-14T15:00:00.000000001Z stdout F migration started\n" +
				"2026-07-14T15:00:00.000000002Z stdout F migration finished\n"
			Expect(os.WriteFile(logsFile, []byte(logLines), 0o644)).To(Succeed())

			node := kwokNode(nodeName)
			Expect(createNode(ctx, node)).To(Succeed())

			err := SwitchToKubernetesUser(ctx, JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteNode(ctx, node)).To(Succeed())
			})

			Expect(createPodLogs(ctx, podName, namespace, "app", logsFile)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deletePodLogs(ctx, podName, namespace)).To(Succeed())
			})

			jr := pendingJobRequest(jobRequestName, namespace, JobRequesterUser.ARN)
			jr.Status.State = jrv1.JobRequestStarted
			jr.Status.JobName = jobName
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})

			job := suspendedJob(jobName, namespace)
			Expect(createJob(ctx, job)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJob(ctx, job)).To(Succeed())
			})

			pod := pendingPod(podName, namespace, jobName)
			pod.Spec.NodeName = nodeName
			pod.Spec.Tolerations = []corev1.Toleration{
				{
					Key:      "kwok.x-k8s.io/node",
					Operator: corev1.TolerationOpExists,
				},
			}
			Expect(createPod(ctx, pod)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deletePod(ctx, pod)).To(Succeed())
			})

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
			Eventually(session.Err, "10s").Should(gbytes.Say("migration started"))
			Eventually(session.Err, "10s").Should(gbytes.Say("migration finished"))
		})
	})

	Context("when logs are not being followed but job exists", func() {
		const jobRequestName = "has-job-name"
		const jobName = "has-job-name-x7k2p"
		const namespace = "apps"

		It("prints a kubectl command", func(ctx SpecContext) {
			err := SwitchToKubernetesUser(ctx, JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())

			jr := pendingJobRequest(jobRequestName, namespace, JobRequesterUser.ARN)
			jr.Status.State = jrv1.JobRequestStarted
			jr.Status.JobName = jobName
			Expect(createJobRequest(ctx, jr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})

			cmd, err := cliCmd(ctx,
				"jobrequest", "logs", jobRequestName,
				"--kubeconfig", kubeconfigPath,
				"--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			outputString := string(output)

			Expect(outputString).To(Equal(
				fmt.Sprintf(
					"Print logs:\n $ kubectl -n %s logs -f job/%s\n",
					namespace,
					jobName,
				),
			))
		})
	})

	Context("when logs are not being followed and job has completed", func() {
		const jobRequestName = "has-job-name"
		const jobName = "has-job-name-x7k2p"
		const namespace = "apps"

		It("prints a kubectl command and a link to Logit", func(ctx SpecContext) {
			err := SwitchToKubernetesUser(ctx, JobRequesterUser)
			Expect(err).NotTo(HaveOccurred())

			jr := pendingJobRequest(jobRequestName, namespace, JobRequesterUser.ARN)
			jr.Status.State = jrv1.JobRequestComplete
			jr.Status.ReviewName = jrv1.JobRequestReviewReviewedByAnnotation
			jr.Status.JobName = jobName
			Expect(createJobRequest(ctx, jr)).To(Succeed())
			jrr := approvedJobRequestReview(jobName, namespace, jobRequestName)
			Expect(createJobRequestReview(ctx, jrr)).To(Succeed())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})

			cmd, err := cliCmd(ctx,
				"jobrequest", "logs", jobRequestName,
				"--kubeconfig", kubeconfigPath,
				"--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			outputString := string(output)

			fmt.Println(outputString)

			Expect(outputString).To(ContainSubstring(
				fmt.Sprintf(
					"Print logs:\n $ kubectl -n %s logs -f job/%s\n",
					namespace,
					jobName,
				),
			))

			Expect(outputString).To(ContainSubstring("View logs in OpenSearch:"))
		})
	})
})

// Print logs:
//  $ kubectl -n apps logs -f job/jr-govuk-replatform-test-app-119631169
// View logs in OpenSearch:
// https://kibana.logit.io/s/42f4d2d5-e9ce-451f-8ffc-cdb25bd624f8/app/data-explorer/discover#?_g=(time:(from:'2026-10-01T16:14:45Z',to:'2026-10-01T19:14:45Z'))&_a=(discover:(metadata:(indexPattern:'filebeat-2026.10.01',view:discover)))&_q=(query:(language:kuery,query:'kubernetes.job.name:%20jr-govuk-replatform-test-app-119631169'))
