package completions

import (
	"fmt"
	"strings"

	"github.com/alphagov/govuk-cli/internal/kubernetes"
	"github.com/spf13/cobra"
)

func KubernetesNamespaces(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	kubeconfigFlag := cmd.Flags().Lookup("kubeconfig")
	kubeConfig, err := kubernetes.CreateKubeConfig(kubeconfigFlag)

	if err != nil {
		cobra.CompError(fmt.Sprintf("error creating kubeconfig: %s", err.Error()))
		return []string{}, cobra.ShellCompDirectiveError
	}

	client, err := kubernetes.CreateCoreV1Client(cmd.Context(), kubeConfig)
	if err != nil {
		cobra.CompError(fmt.Sprintf("error creating corev1 client: %s", err.Error()))
		return []string{}, cobra.ShellCompDirectiveError
	}

	namespaceNames, err := client.GetNamespaceNames()
	if err != nil {
		cobra.CompError(fmt.Sprintf("error getting list of namespaces: %s", err.Error()))
		return []string{}, cobra.ShellCompDirectiveError
	}

	filtereredNamespaceNames := []cobra.Completion{}

	for _, namespaceName := range namespaceNames {
		if strings.HasPrefix(namespaceName, toComplete) {
			filtereredNamespaceNames = append(filtereredNamespaceNames, namespaceName)
		}
	}

	return filtereredNamespaceNames, cobra.ShellCompDirectiveNoFileComp
}
