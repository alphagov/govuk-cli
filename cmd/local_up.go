/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/alphagov/govuk-cli/internal/local"
	"github.com/alphagov/govuk-cli/internal/local/github"
	l "github.com/alphagov/govuk-cli/internal/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// localUpCmd represents the local up command
var localUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the GOV.UK Local stack",
	Long: `Start the GOV.UK Local stack.

Creates a K3d Kubernetes cluster (if one does not already exist), loads the
application definitions from local/apps, and deploys each application to the
cluster in dependency order.

Requires the GITHUB_TOKEN environment variable to be set, which is used to
look up the latest container image tags from GitHub.`,
	Run: func(cmd *cobra.Command, args []string) {
		l.Log().Info("starting GOV.UK Local stack")

		clusterName, err := cmd.Flags().GetString("cluster-name")
		cobra.CheckErr(err)
		registryPort, err := cmd.Flags().GetInt("registry-port")
		cobra.CheckErr(err)
		appPort, err := cmd.Flags().GetInt("app-port")
		cobra.CheckErr(err)
		apiPort, err := cmd.Flags().GetInt("k8s-api-port")
		cobra.CheckErr(err)

		config := local.ClusterConfig{
			Name:         clusterName,
			AppPort:      appPort,
			RegistryPort: registryPort,
			ApiPort:      apiPort,
		}

		cluster, err := local.K3dClusterEnsure(config)
		cobra.CheckErr(err)

		l.Log().Info("app def load")

		defs, err := local.LoadDefinitions("local/apps")
		cobra.CheckErr(err)

		l.Log().WithField("count", len(defs)).Info("loaded definitions")

		ordered, err := local.OrderApplications(defs)

		githubToken, ok := os.LookupEnv("GITHUB_TOKEN")
		if !ok {
			l.Log().Error("GITHUB_TOKEN not set")
			os.Exit(1)
		}

		for _, app := range ordered {
			imageOwner := app.ImageOwner()
			imageRepo := app.ImageRepo()
			imageTag := app.ImageTag()

			if imageTag == "" {
				imageTag, err = github.GetImageTag(githubToken, imageOwner, imageRepo)
				cobra.CheckErr(err)
			}

			l.Log().WithFields(logrus.Fields{
				"name":       app.Name,
				"imageOwner": imageOwner,
				"imageRepo":  imageRepo,
				"imageTag":   imageTag,
			}).Info("ensuring app")

			restConfig, err := local.RestConfigGet(cluster)
			cobra.CheckErr(err)

			app.Values["registryHost"] = "ghcr.io"
			app.Values["imageRepo"] = fmt.Sprintf("%s/%s", imageOwner, imageRepo)
			app.Values["imageTag"] = imageTag

			err = local.EnsureApplication(
				*app,
				restConfig,
			)
			cobra.CheckErr(err)
		}
	},
}

func init() {
	localCmd.AddCommand(localUpCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// localUpCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// localUpCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
