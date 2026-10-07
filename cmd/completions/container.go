package completions

import (
	"fmt"
	"strings"

	"github.com/alphagov/govuk-cli/internal/kubernetes"
	"github.com/spf13/cobra"
)

func KubernetesDeploymentContainers(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	// The deployment argument must have been completed already
	if len(args) < 1 {
		return []cobra.Completion{}, cobra.ShellCompDirectiveNoFileComp
	}

	deploymentName := args[0]
	deploymentName = strings.TrimPrefix(deploymentName, "deployments/")
	deploymentName = strings.TrimPrefix(deploymentName, "deployment/")
	deploymentName = strings.TrimPrefix(deploymentName, "deploy/")

	// After removing prefixes if we are left with no deployment name, we cannot offer container suggestions
	if deploymentName == "" {
		return []cobra.Completion{}, cobra.ShellCompDirectiveNoFileComp
	}

	kubeconfigFlag := cmd.Flags().Lookup("kubeconfig")
	kubeConfig, err := kubernetes.CreateKubeConfig(kubeconfigFlag)
	if err != nil {
		cobra.CompError(fmt.Sprintf("error creating kubeconfig: %s", err.Error()))
		return []string{}, cobra.ShellCompDirectiveError
	}

	client, err := kubernetes.CreateAppsV1Client(cmd.Context(), kubeConfig)
	if err != nil {
		cobra.CompError(fmt.Sprintf("error creating appsv1 client: %s", err.Error()))
		return []string{}, cobra.ShellCompDirectiveError
	}

	containerNames, err := client.GetContainerNamesInDeployment(cmd.Flag("namespace").Value.String(), deploymentName)
	if err != nil {
		cobra.CompError(fmt.Sprintf("error getting list of containers in deployment: %s", err.Error()))
		return []string{}, cobra.ShellCompDirectiveError
	}

	filtereredContainerNames := []cobra.Completion{}

	for _, containerName := range containerNames {
		if strings.HasPrefix(containerName, toComplete) {
			filtereredContainerNames = append(filtereredContainerNames, containerName)
		}
	}

	return filtereredContainerNames, cobra.ShellCompDirectiveNoFileComp
}
