package integration_tests

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("jobrequest create container autocompletion", Ordered, func() {
	targetDeploymentName := "container-test-deployment"
	nonTargetDeploymentName := "container-test-other-deployment"
	// These are purposefully out of order
	targetContainerNames := []string{
		"baz",
		"foo",
		"quux",
		"bar",
		"quz",
	}
	nonTargetContainerNames := []string{
		"flibble",
		"wibble",
	}

	var tmpDir string

	BeforeAll(func(ctx SpecContext) {
		var err error

		By("Creating a test deployment with multiple containers")
		tmpDir, err = os.MkdirTemp("", "govuk-cli-integration-tests-container-completion-*")
		Expect(err).NotTo(HaveOccurred())

		manifestPath := path.Join(tmpDir, targetDeploymentName+".yaml")
		err = renderTemplate(ctx,
			"deployment.template.yaml",
			manifestPath,
			&deploymentTemplateData{
				Name:           targetDeploymentName,
				ContainerNames: targetContainerNames,
			},
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = kubectl(ctx, "apply", "-n", "apps", "-f", manifestPath)
		Expect(err).NotTo(HaveOccurred())

		By("Creating another deployment with differently named containers")
		manifestPath = path.Join(tmpDir, nonTargetDeploymentName+".yaml")

		err = renderTemplate(ctx,
			"deployment.template.yaml",
			manifestPath,
			&deploymentTemplateData{
				Name:           nonTargetDeploymentName,
				ContainerNames: nonTargetContainerNames,
			},
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = kubectl(ctx, "apply", "-n", "apps", "-f", manifestPath)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func(ctx SpecContext) {
		if tmpDir != "" {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		}

		By("Deleting the target test deployment")
		_, err := kubectl(ctx, "delete", "deployment", "-n", "apps", targetDeploymentName)
		Expect(err).NotTo(HaveOccurred())

		By("Deleting the non-target test deployment")
		_, err = kubectl(ctx, "delete", "deployment", "-n", "apps", nonTargetDeploymentName)
		Expect(err).NotTo(HaveOccurred())
	})

	It("completes all containers when none are specified", func(ctx SpecContext) {
		cliCmd, err := completionCliCmd(ctx, "jobrequest", "create", targetDeploymentName, "-c", "")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
		Expect(completionResult.Suggestions).To(ContainElements(targetContainerNames))
		Expect(completionResult.Suggestions).NotTo(ContainElements(nonTargetContainerNames))
	})

	DescribeTable(
		"completes container names where the deployment has a valid kubernetes deployment prefix",
		func(ctx SpecContext, prefix string) {
			cliCmd, err := completionCliCmd(ctx, "jobrequest", "create", prefix+targetDeploymentName, "-c", "")
			Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

			completionResult, err := getCompletionResult(cliCmd)
			Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

			Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
			Expect(completionResult.Suggestions).To(ContainElements(targetContainerNames))
			Expect(completionResult.Suggestions).NotTo(ContainElements(nonTargetContainerNames))
		},
		Entry("with deployment/ as a prefix", "deployment/"),
		Entry("with deployments/ as a prefix", "deployments/"),
		Entry("with deploy/ as a prefix", "deploy/"),
	)

	It("generates completions based on the partial value already typed", func(ctx SpecContext) {
		cliCmd, err := completionCliCmd(ctx, "jobrequest", "create", targetDeploymentName, "-c", "qu")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
		Expect(completionResult.Suggestions).To(ContainElements([]string{"quux", "quz"}))
		Expect(completionResult.Suggestions).NotTo(ContainElements("baz"))
		Expect(completionResult.Suggestions).NotTo(ContainElements(nonTargetContainerNames))
	})

	It("respects the namespace flag and lists deployments in the namespace specified if already in the command", func(ctx SpecContext) {
		By("Creating a deployment in a namespace other than 'apps'")
		otherDeploymentName := "other-namespace-deployment"
		containerName := "other-container-name"

		manifestPath := path.Join(tmpDir, otherDeploymentName+".yaml")
		err := renderTemplate(ctx,
			"deployment.template.yaml",
			manifestPath,
			&deploymentTemplateData{
				Name:           otherDeploymentName,
				ContainerNames: []string{containerName},
			},
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = kubectl(ctx, "apply", "-n", "default", "-f", manifestPath)
		Expect(err).NotTo(HaveOccurred())

		defer func() {
			_, err = kubectl(ctx, "delete", "deployment", "-n", "default", otherDeploymentName)
			Expect(err).NotTo(HaveOccurred())
		}()

		By("Testing the completion command")
		cliCmd, err := completionCliCmd(ctx, "jobrequest", "-n", "default", "create", otherDeploymentName, "-c", "")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
		Expect(completionResult.Suggestions).To(ContainElements(containerName))
		Expect(completionResult.Suggestions).NotTo(ContainElements(targetContainerNames))
	})

	It("does not generate any suggestions when the user is not authenticated", func(ctx SpecContext) {
		cliCmd, err := cliCmd(ctx, "__complete", "jobrequest", "create", targetDeploymentName, "-c", "")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveError"))
		Expect(completionResult.Suggestions).To(HaveLen(0))

	})
})
