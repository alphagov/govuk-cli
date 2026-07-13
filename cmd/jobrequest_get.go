/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"al.essio.dev/pkg/shellescape"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"charm.land/log/v2"
	"github.com/alphagov/govuk-cli/internal/jobrequest"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("get called")
		kubeconfig := cmd.Flag("kubeconfig").Value.String()
		namespace := cmd.Flag("namespace").Value.String()
		follow, err := cmd.Flags().GetBool("follow")

		if err != nil {
			log.Error("error getting follow flag", "error", err)
			os.Exit(1)
		}

		client, err := jobrequest.CreateJobRequestClient(kubeconfig, namespace)
		if err != nil {
			log.Error("error creating job request client", "error", err)
			os.Exit(1)
		}

		if len(args) != 1 {
			log.Error("incorrect number of args", "len", len(args))
			cmd.Help()
			os.Exit(1)
		}

		jobRequestName := args[0]
		log.Debug("getting job request", "name", jobRequestName)
		jr, err := client.JobRequest(jobRequestName)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Error("job request not found", "name", jobRequestName)
				os.Exit(1)
			}
			log.Error("error getting job request", "error", err)
			os.Exit(1)
		}

		requestedBy, isOk := jr.ObjectMeta.Annotations["platform.publishing.service.gov.uk/requested-by"]
		if !isOk {
			requestedBy = "-"
		}

		arn, err := jobrequest.ParseAssumedRoleArn(requestedBy)
		if err == nil {
			log.Debug("parsed requester arn", "arn", arn)
			requestedBy = fmt.Sprintf("%s (%s)", arn.UserName, arn.RoleName)
		} else {
			log.Error("error parsing requester ARN", "arn", requestedBy, "error", err)
		}

		state := jr.Status.State
		if state == "" {
			state = "Unknown"
		}

		baseStyle := lipgloss.NewStyle().
			Padding(0, 1)
		keyStyle := baseStyle.
			Bold(true).
			Align(lipgloss.Left)

		t := table.New().
			Border(lipgloss.NormalBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#1d70b8"))).
			StyleFunc(func(row int, col int) lipgloss.Style {
				switch col {
				case 0:
					return keyStyle
				default:
					return baseStyle
				}
			})
		t.Row("Name", jr.Name)
		t.Row("Command", fmt.Sprintf("%s %s", jr.Spec.Command, shellescape.QuoteCommand(jr.Spec.Args)))
		t.Row("Status", state)
		t.Row("Requested By", requestedBy)

		if jr.Status.ReviewName != "" {
			reviewedBy := "Unknown"

			jrr, err := client.JobRequestReview(jr.Status.ReviewName)
			if err != nil {
				log.Debug("error getting JobRequestReview resource", "error", err, "jobRequestReviewName", jr.Status.ReviewName)
				reviewedBy = "[Error getting JobRequestReview]"
			} else {
				reviewedByAnnotation, isOk := jrr.ObjectMeta.Annotations["platform.publishing.service.gov.uk/reviewed-by"]
				if !isOk {
					reviewedBy = "[Review has no reviewed-by annotation]"
				} else {
					reviewedArn, err := jobrequest.ParseAssumedRoleArn(reviewedByAnnotation)
					if err != nil {
						log.Debug("error parsing reviewed-by ARN", "arn", reviewedByAnnotation, "error", err)
						reviewedBy = "[Error parsing reviewer's role ARN]"
					} else {
						reviewedBy = fmt.Sprintf("%s (%s)", reviewedArn.UserName, reviewedArn.RoleName)
					}
				}
				t.Row()
				t.Row("Review Decision", jrr.Spec.Decision)
			}

			t.Row("Reviewed By", reviewedBy)
		}

		lipgloss.Println(t)

		commandStyle := lipgloss.NewStyle().PaddingLeft(1)
		boldStyle := lipgloss.NewStyle().Bold(true)
		if jr.Status.JobName != "" {
			kubectlCommand := fmt.Sprintf("$ kubectl -n %s logs -f job/%s", shellescape.Quote(namespace), shellescape.Quote(jr.Status.JobName))
			lipgloss.Println(boldStyle.Render("Print logs:"))
			lipgloss.Println(commandStyle.Render(kubectlCommand))
		}

		if jr.Status.ReviewName != "" {
			kubectlCommand := fmt.Sprintf("$ kubectl -n %s get jobrequestreview %s -o yaml", shellescape.Quote(namespace), shellescape.Quote(jr.Status.ReviewName))
			lipgloss.Println(boldStyle.Render("Get review:"))
			lipgloss.Println(commandStyle.Render(kubectlCommand))
		}

		if follow {
			err = client.FollowJobRequest(jobRequestName)
			if err != nil {
				log.Error("error following job request", "error", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	jobrequestCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	getCmd.Flags().BoolP("follow", "f", false, "Wait for Job to be created and tail logs")
}
