/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/alphagov/govuk-cli/internal/local"
	"github.com/spf13/cobra"
)

// localUpCmd represents the local up command
var localDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the GOV.UK Local stack",
	Long: `Stop the GOV.UK Local stack by destroying the K3d Kubernetes cluster
and everything running in it.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("local down called")
		clusterName, err := cmd.Flags().GetString("cluster-name")
		cobra.CheckErr(err)

		err = local.K3dClusterDestroy(local.ClusterConfig{Name: clusterName})
		cobra.CheckErr(err)
	},
}

func init() {
	localCmd.AddCommand(localDownCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// localUpCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// localUpCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
