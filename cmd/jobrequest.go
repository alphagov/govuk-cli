package cmd

import (
	"path/filepath"

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
	jobrequestCmd.PersistentFlags().BoolP("follow", "f", false, "Wait for Job to be created and tail logs")
}
