package integration_tests

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type deploymentTemplateData struct {
	Name string
}

var _ = Describe("jobrequest namespace autocompletion", Ordered, func() {
	// These are purposefully out of order
	deploymentNames := []string{
		"baz",
		"foo",
		"quux",
		"bar",
		"quz",
	}

	var tmpDir string

	BeforeAll(func(ctx SpecContext) {
		var err error

		By("Creating test deployments")
		tmpDir, err = os.MkdirTemp("", "govuk-cli-integration-tests-deployment-templates-*")
		Expect(err).NotTo(HaveOccurred())

		for _, deploymentName := range deploymentNames {
			manifestPath := path.Join(tmpDir, deploymentName+".yaml")
			err := renderTemplate(ctx,
				"govukReplatformTestApp.template.yaml",
				manifestPath,
				&deploymentTemplateData{Name: deploymentName},
			)
			Expect(err).NotTo(HaveOccurred())

			_, err = kubectl(ctx, "apply", "-n", "apps", "-f", manifestPath)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	AfterAll(func(ctx SpecContext) {
		By("Deleting test deployments")
		for _, deploymentName := range deploymentNames {
			_, err := kubectl(ctx, "delete", "deployment", "-n", "apps", deploymentName)
			Expect(err).NotTo(HaveOccurred())
		}

		if tmpDir != "" {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		}
	})

	It("completes all deployments when none are specified", func(ctx SpecContext) {
		cliCmd, err := completionCliCmd(ctx, "jobrequest", "create", "")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
		Expect(completionResult.Suggestions).To(ContainElements(deploymentNames))
	})

	DescribeTable(
		"completes deployment names including the prefix when a valid kubernetes deployment prefix is used",
		func(ctx SpecContext, prefix string) {
			deploymentNamesWithPrefix := make([]string, len(deploymentNames))
			for i, deploymentName := range deploymentNames {
				deploymentNamesWithPrefix[i] = prefix + deploymentName
			}

			cliCmd, err := completionCliCmd(ctx, "jobrequest", "create", prefix)
			Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

			completionResult, err := getCompletionResult(cliCmd)
			Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

			Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
			Expect(completionResult.Suggestions).To(ContainElements(deploymentNamesWithPrefix))
		},
		Entry("with deployment/ as a prefix", "deployment/"),
		Entry("with deployments/ as a prefix", "deployments/"),
		Entry("with deploy/ as a prefix", "deploy/"),
	)

	It("generates completions based on the partial value already typed", func(ctx SpecContext) {
		cliCmd, err := completionCliCmd(ctx, "jobrequest", "create", "qu")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
		Expect(completionResult.Suggestions).To(ContainElements([]string{"quux", "quz"}))
	})

	It("respects the namespace flag and lists deployments in the namespace specified if already in the command", func(ctx SpecContext) {
		By("Creating a deployment in a namespace other than 'apps'")
		otherDeploymentName := "other-namespace-deployment"

		manifestPath := path.Join(tmpDir, otherDeploymentName+".yaml")
		err := renderTemplate(ctx,
			"govukReplatformTestApp.template.yaml",
			manifestPath,
			&deploymentTemplateData{Name: otherDeploymentName},
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = kubectl(ctx, "apply", "-n", "default", "-f", manifestPath)
		Expect(err).NotTo(HaveOccurred())

		defer func() {
			_, err = kubectl(ctx, "delete", "deployment", "-n", "default", otherDeploymentName)
			Expect(err).NotTo(HaveOccurred())
		}()

		By("Testing the completion command")
		cliCmd, err := completionCliCmd(ctx, "jobrequest", "-n", "default", "create", "")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveNoFileComp"))
		Expect(completionResult.Suggestions).To(ContainElements(otherDeploymentName))
		Expect(completionResult.Suggestions).NotTo(ContainElements(deploymentNames))
	})

	It("does not generate any suggestions when the user is not authenticated", func(ctx SpecContext) {
		cliCmd, err := cliCmd(ctx, "__complete", "jobrequest", "create", "")
		Expect(err).NotTo(HaveOccurred(), "Couldn't create completion cli command")

		completionResult, err := getCompletionResult(cliCmd)
		Expect(err).NotTo(HaveOccurred(), "Couldn't parse completion results")

		Expect(completionResult.CobraCompletionDirectiveName).To(Equal("ShellCompDirectiveError"))
		Expect(completionResult.Suggestions).To(HaveLen(0))

	})
})
