package completions

import (
	"fmt"
	"strings"

	"github.com/alphagov/govuk-cli/internal/kubernetes"
	"github.com/spf13/cobra"
)

func KubernetesDeployments(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	var prefix string

	if strings.HasPrefix(toComplete, "deployments/") {
		prefix = "deployments/"
	} else if strings.HasPrefix(toComplete, "deployment/") {
		prefix = "deployment/"
	} else if strings.HasPrefix(toComplete, "deploy/") {
		prefix = "deploy/"
	}

	trimmedToComplete := strings.TrimPrefix(toComplete, prefix)

	kubeconfigFlag := cmd.Flags().Lookup("kubeconfig")
	kubeConfig, err := kubernetes.CreateKubeConfig(kubeconfigFlag)

	if err != nil {
		cobra.CompError(fmt.Sprintf("error creating kubeconfig: %s", err.Error()))
		return []string{}, cobra.ShellCompDirectiveError
	}

	paginationLimit, err := cmd.Flags().GetInt64("pagination-limit")
	if err != nil {
		cobra.CompError("error getting pagination-limit as an Int64")
		return []string{}, cobra.ShellCompDirectiveError
	}

	client, err := kubernetes.CreateAppsV1Client(cmd.Context(), kubeConfig)
	if err != nil {
		cobra.CompError(fmt.Sprintf("error creating appsv1 client: %s", err.Error()))
		return []string{}, cobra.ShellCompDirectiveError
	}

	deploymentNames, err := client.GetDeploymentNames(cmd.Flag("namespace").Value.String(), paginationLimit)
	if err != nil {
		cobra.CompError(fmt.Sprintf("error getting list of namespaces: %s", err.Error()))
		return []string{}, cobra.ShellCompDirectiveError
	}

	filtereredDeploymentNames := []cobra.Completion{}

	for _, deploymentName := range deploymentNames {
		if strings.HasPrefix(deploymentName, trimmedToComplete) {
			filtereredDeploymentNames = append(filtereredDeploymentNames, prefix+deploymentName)
		}
	}

	return filtereredDeploymentNames, cobra.ShellCompDirectiveNoFileComp
}
