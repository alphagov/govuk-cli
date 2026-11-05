package cmd

import (
	"fmt"
	"os"
	"strings"

	"al.essio.dev/pkg/shellescape"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/alphagov/govuk-cli/cmd/completions"
	"github.com/alphagov/govuk-cli/internal/jobrequest"
	"github.com/alphagov/govuk-cli/internal/kubernetes"

	"github.com/alphagov/govuk-cli/internal/style"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Get logs for a job request",
	Example: `govuk-cli jobrequest logs <jobrequest>
govuk-cli jobrequest logs <jobrequest> --follow`,
	Long: "Get the logs for a jobrequest, optionally following them as they are updated.",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		// Only offer autocomplete suggestions for the first argument to the command
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return completions.JobRequest(cmd, args, toComplete)
	},
	Run: func(cmd *cobra.Command, args []string) {
		namespace := cmd.Flag("namespace").Value.String()

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

		outputLogsUrl, err := initLogUrlGenerator(ctx, kubeConfig, client, os.Stderr)
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

		if follow {
			err = client.FollowJobRequest(jobRequestName)
		} else {
			if !jr.HasBeenReviewed() {
				_, err := lipgloss.Println("JobRequest has not yet been reviewed, cannot print the logs, specify --follow if you want to watch the status of the JobRequest")
				cobra.CheckErr(err)
				os.Exit(1)
			}
			if jr.HasBeenRejected() {
				_, err := lipgloss.Println("JobRequest was Rejected, there are no logs to print")
				cobra.CheckErr(err)
				os.Exit(1)
			}
			err = client.PrintLogs(jr)
		}

		if err != nil {
			log.Error("Error printing job request logs", "error", err)
			_, err = lipgloss.Printf("\n")
			cobra.CheckErr(err)
			outputLogsUrl(jr)
			os.Exit(1)
		} else {
			_, err = lipgloss.Fprintf(os.Stderr, "--- LOGS END ---\n\n")
			cobra.CheckErr(err)
		}

		if jr.Status.JobName != "" {
			kubectlCommand := fmt.Sprintf("$ kubectl -n %s logs -f %s", shellescape.Quote(namespace), shellescape.Quote(fmt.Sprintf("job/%s", jr.Status.JobName)))
			_, err = lipgloss.Fprintln(os.Stderr, style.BoldStyle.Render("Print logs:"))
			cobra.CheckErr(err)
			_, err = lipgloss.Fprintln(os.Stderr, style.CommandStyle.Render(kubectlCommand))
			cobra.CheckErr(err)
			_, err = lipgloss.Fprintln(os.Stderr, "")
			cobra.CheckErr(err)

			outputLogsUrl(jr)
		}
	},
}

func init() {
	logsCmd.Flags().Bool("follow", false, "Follow logs")

	jobrequestCmd.AddCommand(logsCmd)
}
