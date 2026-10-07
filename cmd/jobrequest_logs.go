package cmd

import (
	"fmt"
	"os"
	"strings"

	"al.essio.dev/pkg/shellescape"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/alphagov/govuk-cli/internal/jobrequest"
	"github.com/alphagov/govuk-cli/internal/kubernetes"

	"github.com/alphagov/govuk-cli/internal/style"
	// "github.com/alphagov/govuk-cli/internal/whoami"
	// jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Get logs for a job request",
	Example: `govuk-cli jobrequest logs <jobrequest>
govuk-cli jobrequest logs <jobrequest> --follow`,
	Long: "Get the logs for a jobrequest, optionally following them as they are updated.",
	Run: func(cmd *cobra.Command, args []string) {
		namespace, err := cmd.Flags().GetString("namespace")

		ctx := cmd.Context()

		follow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			log.Error("Error getting follow flag", "error", err)
			os.Exit(1)
		}

		kubeconfigFlag := cmd.Flags().Lookup("kubeconfig")
		kubeConfig, err := kubernetes.CreateKubeConfig(kubeconfigFlag)
		if err != nil {
			log.Error("error creating kubeconfig", "error", err)
			os.Exit(1)
		}

		client, err := jobrequest.CreateJobRequestClient(ctx, kubeConfig, namespace)
		if err != nil {
			log.Error("Error creating job request client", "error", err)
			os.Exit(1)
		}

		outputLogsUrl, err := initLogUrlGenerator(ctx, kubeConfig, client)
		if err != nil {
			log.Error("Error creating kube client for whoami", "error", err)
			os.Exit(1)
		}

		if len(args) != 1 {
			log.Error("Only one argument should be provided", "argumentCount", len(args))
			cobra.CheckErr(cmd.Help())
			os.Exit(1)
		}

		jobRequestName := args[0]
		log.Debug("Getting job request", "name", jobRequestName)
		jr, err := client.JobRequest(jobRequestName)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Error("Job request not found", "name", jobRequestName)
				os.Exit(1)
			}
			log.Error("Error getting job request", "error", err)
			os.Exit(1)
		}

		if jr.Status.JobName != "" && !follow {
			kubectlCommand := fmt.Sprintf("$ kubectl -n %s logs -f %s", shellescape.Quote(namespace), shellescape.Quote(fmt.Sprintf("job/%s", jr.Status.JobName)))
			_, err = lipgloss.Println(style.BoldStyle.Render("Print logs:"))
			cobra.CheckErr(err)
			_, err = lipgloss.Println(style.CommandStyle.Render(kubectlCommand))
			cobra.CheckErr(err)

			outputLogsUrl(jr)
		}

		if follow {
			err = client.FollowJobRequest(jobRequestName)
			if err != nil {
				log.Error("Error following job request", "error", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	logsCmd.Flags().Bool("follow", false, "Follow logs")

	jobrequestCmd.AddCommand(logsCmd)
}
