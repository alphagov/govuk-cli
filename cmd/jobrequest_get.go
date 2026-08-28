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
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get job-request-id",
	Short: "Get a job request",
	Example: `govuk-cli jobrequest get jr-12345678
govuk-cli jobrequest get jr-12345678 -f`,
	Long: `Get details of a job request from the cluster.

If you want to wait for the job request to be reviewed and
tail logs for the resulting job, use the --follow flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		namespace := cmd.Flag("namespace").Value.String()
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

		client, err := jobrequest.CreateJobRequestClient(kubeConfig, namespace)
		if err != nil {
			log.Error("Error creating job request client", "error", err)
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

		t, err := client.JobRequestDetailsKVTable(jr)
		if err != nil {
			log.Error("Error getting job request details", "error", err)
			os.Exit(1)
		}

		if jr.HasBeenReviewed() {
			var reviewedBy string

			jrr, err := client.JobRequestReview(jr.Status.ReviewName)
			if err != nil {
				log.Warn("Error getting JobRequestReview resource", "error", err, "jobRequestReviewName", jr.Status.ReviewName)
				reviewedBy = "[Error getting JobRequestReview]"
			} else {
				reviewedByAnnotation, err := jrr.GetReviewedBy()
				if err != nil {
					reviewedBy = "[Review has no reviewed-by annotation]"
				} else {
					userIdentity, err := jrv1.ParseUserIdentityFromARN(reviewedByAnnotation)
					if err != nil {
						log.Warn("Error parsing reviewed-by ARN", "arn", reviewedByAnnotation, "error", err)
						reviewedBy = "[Error parsing reviewer's role ARN]"
					} else {
						reviewedBy = fmt.Sprintf("%s (%s)", userIdentity.UserName, userIdentity.RoleName)
					}
				}
				t.Row()
				t.Row("Review Decision", jrr.Spec.Decision)
			}

			t.Row("Reviewed By", reviewedBy)
		}

		_, err = lipgloss.Println(t)
		cobra.CheckErr(err)

		if jr.Status.JobName != "" && !follow {
			kubectlCommand := fmt.Sprintf("$ kubectl -n %s logs -f %s", shellescape.Quote(namespace), shellescape.Quote(fmt.Sprintf("job/%s", jr.Status.JobName)))
			_, err = lipgloss.Println(style.BoldStyle.Render("Print logs:"))
			cobra.CheckErr(err)
			_, err = lipgloss.Println(style.CommandStyle.Render(kubectlCommand))
			cobra.CheckErr(err)
		}

		if !jr.HasBeenReviewed() {
			kubectlCommand := fmt.Sprintf("$ kubectl -n %s get jobrequestreview %s -o yaml", shellescape.Quote(namespace), shellescape.Quote(jr.Status.ReviewName))
			_, err = lipgloss.Println(style.BoldStyle.Render("Get review:"))
			cobra.CheckErr(err)
			_, err = lipgloss.Println(style.CommandStyle.Render(kubectlCommand))
			cobra.CheckErr(err)
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
	jobrequestCmd.AddCommand(getCmd)
}
