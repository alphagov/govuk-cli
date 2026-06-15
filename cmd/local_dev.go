/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/alphagov/govuk-cli/internal/local"
	"github.com/alphagov/govuk-cli/internal/local/build"
	l "github.com/alphagov/govuk-cli/internal/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// localDevCmd represents the local dev command
var localDevCmd = &cobra.Command{
	Use:   "dev appName",
	Short: "Start a GOV.UK app in dev mode",
	Long:  `Start an app in dev mode. Use 'govuk-cli local up' first.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			l.Log().Error("no app name provided")
			os.Exit(1)
		}

		appName := args[0]

		l.Log().WithField("name", appName).Info("starting app in dev mode")

		reposPath, err := cmd.Flags().GetString("repos-path")
		cobra.CheckErr(err)

		appDir, err := filepath.Abs(path.Join(reposPath, appName))
		cobra.CheckErr(err)

		imageTag := fmt.Sprintf("dev-%d", time.Now().Unix())

		cobra.CheckErr(build.BuildContainerImage(
			appDir,
			appName,
			"localhost:2013",
			imageTag,
		))

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

		app, ok := defs[appName]
		if !ok {
			l.Log().WithField("name", appName).Error("app not found")
			os.Exit(1)
		}

		imageOwner := app.ImageOwner()
		imageRepo := app.ImageRepo()

		l.Log().WithFields(logrus.Fields{
			"name":       app.Name,
			"imageOwner": imageOwner,
			"imageRepo":  imageRepo,
		}).Info("ensuring app")

		restConfig, err := local.RestConfigGet(cluster)
		cobra.CheckErr(err)

		app.Values["registryHost"] = config.RegistryHost()
		app.Values["imageRepo"] = fmt.Sprintf("%s/%s", imageOwner, imageRepo)
		app.Values["imageTag"] = imageTag

		err = local.EnsureApplication(
			*app,
			restConfig,
		)
		cobra.CheckErr(err)
	},
}

func init() {
	localCmd.AddCommand(localDevCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// localUpCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// localUpCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
