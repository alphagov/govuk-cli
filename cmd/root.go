package cmd

import (
	"os"

	"charm.land/log/v2"
	"github.com/spf13/cobra"
)

var version string = "dev"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "govuk-cli",
	Short:   "GOV.UK Platform CLI",
	Long:    `A CLI for interacting with the GOV.UK Platform.`,
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		logLevel, err := cmd.Flags().GetString("log-level")
		if err != nil {
			return err
		}
		switch logLevel {
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		default:
			log.Warn("unknown log level provided. Defaulting to info", "logLevel", logLevel)
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "Log level (debug, info, warn or error)")
}
