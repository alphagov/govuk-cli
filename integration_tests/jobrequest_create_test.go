package integration_tests

import (
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("jobrequest create", func() {
	const namespace = "default"

	Context("when given a resource in the form kind/name", func() {
		It("creates a JobRequest resolved to the given resource kind", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "create", "deploy/create-deploy-app", "echo", "hello", "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring("Job request created"))
			Expect(string(output)).To(ContainSubstring("echo hello"))

			jrName := regexp.MustCompile(`jr-create-deploy-app-\d+`).FindString(string(output))
			Expect(jrName).NotTo(BeEmpty(), string(output))

			jr, err := getJobRequest(ctx, jrName, namespace)
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})

			Expect(jr.Spec.ContainerFrom.PodSpecFrom.Kind).To(Equal("Deployment"))
			Expect(jr.Spec.ContainerFrom.PodSpecFrom.Group).To(Equal("apps/v1"))
			Expect(jr.Spec.ContainerFrom.PodSpecFrom.Name).To(Equal("create-deploy-app"))
			Expect(jr.Spec.Command).To(Equal("echo"))
			Expect(jr.Spec.Args).To(Equal([]string{"hello"}))

			Expect(string(output)).To(ContainSubstring("deployment/create-deploy-app"))
		})
	})

	Context("when an invalid resource kind is provided", func() {
		It("errors and does not create a JobRequest", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "create", "deployyy/create-invalid-app", "echo", "hello", "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).To(MatchError("exit status 1"))

			Expect(string(output)).To(ContainSubstring("Error resolving workload kind"))
			Expect(string(output)).To(ContainSubstring("cluster doesn't have a resource with name"))
			Expect(string(output)).To(ContainSubstring("deployyy"))
		})
	})

	Context("when no resource kind is specified", func() {
		It("defaults to a Deployment", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "jobrequest", "create", "create-default-app", "echo", "hello", "--kubeconfig", kubeconfigPath, "--namespace", namespace)
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(string(output)).To(ContainSubstring("Job request created"))

			jrName := regexp.MustCompile(`jr-create-default-app-\d+`).FindString(string(output))
			Expect(jrName).NotTo(BeEmpty(), string(output))

			jr, err := getJobRequest(ctx, jrName, namespace)
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func(ctx SpecContext) {
				Expect(deleteJobRequest(ctx, jr)).To(Succeed())
			})

			Expect(jr.Spec.ContainerFrom.PodSpecFrom.Kind).To(Equal("Deployment"))
			Expect(jr.Spec.ContainerFrom.PodSpecFrom.Group).To(Equal("apps/v1"))
			Expect(jr.Spec.ContainerFrom.PodSpecFrom.Name).To(Equal("create-default-app"))

			Expect(string(output)).To(ContainSubstring("deployment/create-default-app"))
		})
	})
})
