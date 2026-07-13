/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
)

// jobrequestCmd represents the jobrequest command
var jobrequestCmd = &cobra.Command{
	Use:     "jobrequest",
	Aliases: []string{"jr"},
	Short:   "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("jobrequest called")
	},
}

func init() {
	rootCmd.AddCommand(jobrequestCmd)

	// log.SetLevel(log.DebugLevel)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// jobrequestCmd.PersistentFlags().String("foo", "", "A help for foo")
	jobrequestCmd.PersistentFlags().String("kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "Path to the kubeconfig file to use for CLI requests.")
	jobrequestCmd.PersistentFlags().StringP("namespace", "n", "apps", "The namespace scope for this CLI request")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// jobrequestCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
