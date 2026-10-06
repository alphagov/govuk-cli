package cmd

import (
	"path/filepath"

	"charm.land/log/v2"
	"github.com/alphagov/govuk-cli/cmd/completions"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
)

// jobrequestCmd represents the jobrequest command
var jobrequestCmd = &cobra.Command{
	Use:     "jobrequest",
	Aliases: []string{"jr"},
	Short:   "Interact with job requests in a GOV.UK Kubernetes cluster",
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(cmd.Help())
	},
}

func init() {
	rootCmd.AddCommand(jobrequestCmd)

	jobrequestCmd.PersistentFlags().String("kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "Path to the kubeconfig file to use for CLI requests.")
	jobrequestCmd.PersistentFlags().StringP("namespace", "n", "apps", "The namespace scope for this CLI request")

	// Disable file completions for all commands and flags unless overridden
	jobrequestCmd.CompletionOptions.SetDefaultShellCompDirective(cobra.ShellCompDirectiveNoFileComp)

	err := jobrequestCmd.RegisterFlagCompletionFunc("namespace", completions.KubernetesNamespaces)
	if err != nil {
		log.Debug("Couldn't register Namepsace auto completion function")
	}

	// Enable file completions just for the kubeconfig flag (note that SellCompDirectiveDefault includes file completions
	err = jobrequestCmd.RegisterFlagCompletionFunc("kubeconfig", cobra.FixedCompletions(nil, cobra.ShellCompDirectiveDefault))
	if err != nil {
		log.Debug("Couldn't register kubeconfig file completions")
	}
}
