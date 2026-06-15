/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// localCmd represents the local command
var localCmd = &cobra.Command{
	Use:   "local",
	Short: "Manage a local GOV.UK development environment",
	Long: `Manage GOV.UK Local, a local development environment that runs
GOV.UK applications in a K3d Kubernetes cluster on your machine.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("local called")
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig(cmd)
	},
}

func initConfig(cmd *cobra.Command) error {
	viper.SetEnvPrefix("GOVUK_LOCAL")
	viper.AutomaticEnv()
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		homeConfigDir, err := os.UserConfigDir()
		if err != nil {
			return err
		}

		viper.AddConfigPath(path.Join(homeConfigDir, "govuk-cli"))
		viper.SetConfigName("govuk-local")
		viper.SetConfigType("yaml")
	}

	err := viper.ReadInConfig()
	if err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return err
		}
	}

	err = viper.BindPFlags(cmd.Flags())
	return err
}

func init() {
	rootCmd.AddCommand(localCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// localCmd.PersistentFlags().String("foo", "", "A help for foo")
	persistentFlags := localCmd.PersistentFlags()
	persistentFlags.StringVar(&cfgFile, "config", "", "Path to config file for GOV.UK Local")

	persistentFlags.StringP("cluster-name", "n", "govuk-local", "Name of the GOV.UK Local K3d cluster")

	persistentFlags.IntP("app-port", "p", 2012, "Port to use for accessing running GOV.UK apps")
	persistentFlags.IntP("registry-port", "r", 2013, "Port to use for the local container registry")
	persistentFlags.IntP("k8s-api-port", "k", 2014, "Port to use for the Kubernetes API")

	persistentFlags.StringP("repos-path", "g", "/Users/samsimpson/govuk", "Path to your GOV.UK source code repositories")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// localCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
