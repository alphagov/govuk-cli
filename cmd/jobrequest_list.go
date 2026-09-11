package cmd

import (
	"os"

	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/alphagov/govuk-cli/internal/jobrequest"
	"github.com/alphagov/govuk-cli/internal/kubernetes"

	"github.com/alphagov/govuk-cli/internal/whoami"
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List job requests",
	Example: `govuk-cli jobrequest list
govuk-cli jobrequest list --mine`,
	Long: "Get a list of your jobrequests (with the --mine flag), or all jobrequests from the cluster.",
	Run: func(cmd *cobra.Command, args []string) {
		namespace, err := cmd.Flags().GetString("namespace")
		if err != nil {
			log.Error("Error getting flag 'mine'", "error", err)
			cobra.CheckErr(cmd.Help())
			os.Exit(1)
		}

		mine, err := cmd.Flags().GetBool("mine")
		if err != nil {
			log.Error("Error getting flag 'mine'", "error", err)
			cobra.CheckErr(cmd.Help())
			os.Exit(1)
		}

		paginationLimit, err := cmd.Flags().GetInt64("pagination-limit")
		if err != nil {
			log.Error("Error getting flag 'pagination-limit'", "error", err)
			cobra.CheckErr(cmd.Help())
			os.Exit(1)
		}

		kubeconfigFlag := cmd.Flags().Lookup("kubeconfig")
		kubeConfig, err := kubernetes.CreateKubeConfig(kubeconfigFlag)
		if err != nil {
			log.Error("error creating kubeconfig", "error", err)
			os.Exit(1)
		}

		client, err := jobrequest.CreateJobRequestClient(kubeConfig, namespace)
		if err != nil {
			log.Error("Error creating job request client", "error", err)
			os.Exit(1)
		}

		if len(args) != 0 {
			log.Error("No arguments should be provided", "argumentCount", len(args))
			cobra.CheckErr(cmd.Help())
			os.Exit(1)
		}

		log.Debug("Getting list of job requests", "mine", mine)

		var userIdentity *jrv1.UserIdentity = nil
		if mine {
			whoamiClient, err := whoami.CreateWhoAmIClient(kubeConfig)
			if err != nil {
				log.Errorf("Couldn't create kubernetes WhoAmI client, error: %s", err.Error())
				os.Exit(1)
			}

			userIdentity, err = whoamiClient.WhoAmI()
			if err != nil {
				log.Errorf("Couldn't execute WhoAmI query against Kuberentes, error: %s", err.Error())
				os.Exit(1)
			}
		}

		jobRequests, err := client.ListJobRequests(userIdentity, paginationLimit)
		if err != nil {
			log.Error("Couldn't list JobRequests", "error", err)
			os.Exit(1)
		}

		if len(jobRequests) == 0 {
			if mine {
				log.Printf("No JobRequests found created by you in namespace %s", namespace)
			} else {
				log.Printf("No JobRequests found in namespace %s", namespace)
			}

			os.Exit(0)
		}

		for _, jobRequest := range jobRequests {
			log.Debug("Job request retrieved", *jobRequest)
		}

		t, err := client.JobRequestDetailsListTable(jobRequests)
		if err != nil {
			log.Error("Couldn't generate Job Request Table", "error", err)
			os.Exit(1)
		}

		_, err = lipgloss.Println(t)
		cobra.CheckErr(err)
	},
}

func init() {
	listCmd.Flags().Bool("mine", false, "List only JobRequests you created")
	listCmd.Flags().Int64("pagination-limit", 100, "Set the maxium items retrieved in a single call to the kubernetes API")

	jobrequestCmd.AddCommand(listCmd)
}
