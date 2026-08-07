package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/alphagov/govuk-cli/internal/jobrequest"
	"github.com/alphagov/govuk-cli/internal/style"
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var reviewCommand = &cobra.Command{
	Use:   "review job-request-id",
	Short: "Review a pending job request",
	Example: `govuk-cli jobrequest review jr-12345678
govuk-cli jobrequest review jr-12345678 -f`,
	Long: `Review a pending job request.

If you want to tail logs for the resulting job,
use the --follow flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		kubeconfig := cmd.Flag("kubeconfig").Value.String()
		namespace := cmd.Flag("namespace").Value.String()
		follow, err := cmd.Flags().GetBool("follow")

		if err != nil {
			log.Error("Error getting follow flag", "error", err)
			os.Exit(1)
		}

		client, err := jobrequest.CreateJobRequestClient(kubeconfig, namespace)
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

		if jr.Status.State != "Pending" {
			log.Error("Only pending job requests are reviewable", "currentState", jr.Status.State)
			os.Exit(1)
		}

		if jr.Status.ReviewName != "" {
			log.Error("Job request has already been reviewed", "reviewName", jr.Status.ReviewName)
			os.Exit(1)
		}

		t, err := client.JobRequestDetailsKVTable(jr)
		if err != nil {
			log.Error("Error getting job request details", "error", err)
			os.Exit(1)
		}

		_, err = lipgloss.Println(t)
		cobra.CheckErr(err)

		stdinReader := bufio.NewReader(os.Stdin)

		decision := parseDecisionInput(stdinReader)

		_, err = lipgloss.Print(style.BoldStyle.Render("Comment: "))
		cobra.CheckErr(err)

		input, err := stdinReader.ReadString('\n')
		if err != nil {
			log.Error("Error reading comment input", "error", err)
			os.Exit(1)
		}

		comment := strings.TrimSpace(input)
		if comment == "" {
			comment = "[None provided]"
		}

		log.Debug("Decision made", "decision", decision, "comment", comment)

		confirmT := style.KVTable()
		confirmT.Row("Job Request Name", jr.Name)
		confirmT.Row("Decision", string(decision))
		confirmT.Row("Comment", comment)

		_, err = lipgloss.Println(confirmT)
		cobra.CheckErr(err)

		// returns nothing if approved, or exits the program if not
		parseSubmitInput(stdinReader)

		jrr := jrv1.JobRequestReview{
			TypeMeta: metav1.TypeMeta{
				APIVersion: jrv1.SchemeGroupVersion.String(),
				Kind:       "JobRequestReview",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("jrr-%s", jr.Name),
				Namespace: namespace,
			},
			Spec: jrv1.JobRequestReviewSpec{
				JobRequestName: jr.Name,
				Decision:       string(decision),
				Description:    comment,
			},
		}

		err = client.CreateJobRequestReview(jrr)
		if err != nil {
			log.Error("Error creating job request review", "error", err)
			os.Exit(1)
		}

		log.Info("Reviewed job request", "jobRequestName", jr.Name, "jobRequestReviewName", jrr.Name)

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
	jobrequestCmd.AddCommand(reviewCommand)
}

// Prompt the user to approve or reject the request.
// Returns either JobRequestReviewApproved or JobRequestReviewRejected
func parseDecisionInput(reader *bufio.Reader) jrv1.JobRequestReviewState {
	for {
		_, err := lipgloss.Printf("Review options: %spprove %seject\n", style.BoldStyle.Render("[A]"), style.BoldStyle.Render("[R]"))
		cobra.CheckErr(err)
		_, err = lipgloss.Print(style.BoldStyle.Render("Your decision: "))
		cobra.CheckErr(err)

		input, err := reader.ReadString('\n')
		if err != nil {
			log.Error("Error reading decision input", "error", err)
			os.Exit(1)
		}
		strippedInput := strings.ToLower(strings.TrimSpace(input))

		switch strippedInput {
		case "a", "approve":
			return jrv1.JobRequestReviewApproved
		case "r", "reject":
			return jrv1.JobRequestReviewRejected
		default:
			log.Error("Only 'approve' and 'reject' are valid decisions", "providedDecision", strippedInput)
		}
	}
}

// Prompt the user to approve submitting their review.
// Returns nothing if approved, exits if not.
func parseSubmitInput(reader *bufio.Reader) {
	for {
		_, err := lipgloss.Print(style.BoldStyle.Render("Submit review? [Y/n]: "))
		cobra.CheckErr(err)

		submitInput, err := reader.ReadString('\n')
		if err != nil {
			log.Error("Error reading comment input", "error", err)
			os.Exit(1)
		}

		submitDecision := strings.ToLower(strings.TrimSpace(submitInput))

		switch submitDecision {
		case "y", "yes":
			log.Debug("Submit approved")
			return
		case "n", "no":
			log.Debug("Submit not approved")
			os.Exit(0)
		default:
			log.Error("Only 'yes' and 'no' are valid submission decision options", "providedDecision", submitDecision)
		}
	}
}
