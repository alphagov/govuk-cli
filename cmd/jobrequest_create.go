package cmd

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"

	"al.essio.dev/pkg/shellescape"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/alphagov/govuk-cli/internal/jobrequest"
	"github.com/alphagov/govuk-cli/internal/kubernetes"
	"github.com/alphagov/govuk-cli/internal/style"
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use: "create source-workload command [arg] ...",
	Example: `govuk-cli jobrequest create deploy/whitehall-admin rake 'my:task[some,args]'
govuk-cli jobrequest new signon rake hello:world`,
	Aliases: []string{"new"},
	Short:   "Create a new job request",
	Long: `Create a new job request.

Job environment (env vars, service accounts, etc) is pulled from source-workload.
Command returns a `,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			log.Error("Not enough arguments provided")
			cobra.CheckErr(cmd.Usage())
			os.Exit(1)
		}

		namespace, err := cmd.Flags().GetString("namespace")
		if err != nil {
			log.Error("Error getting namespace flag value", "error", err)
			os.Exit(1)
		}

		container, err := cmd.Flags().GetString("container")
		if err != nil {
			log.Error("Error getting container flag value", "error", err)
			os.Exit(1)
		}

		follow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			log.Error("error getting follow flag", "error", err)
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

		sourceWorkload := args[0]
		sourceWorkloadParts := strings.Split(sourceWorkload, "/")

		var resourceKind string
		var resourceName string
		if len(sourceWorkloadParts) == 1 {
			resourceKind = "deployment"
			resourceName = sourceWorkloadParts[0]
		} else {
			resourceKind = sourceWorkloadParts[0]
			resourceName = sourceWorkloadParts[1]
		}

		resolved, err := client.ResolveWorkload(resourceKind)
		if err != nil {
			log.Error("Error resolving workload kind", "error", err)
			os.Exit(1)
		}
		log.Debug(resourceName)
		log.Debug("Resolved workload", "kind", resolved)
		resolvedGroupVersion := fmt.Sprintf("%s/%s", resolved.Group, resolved.Version)

		jrName := generateJrName(resourceName)
		jobCommand, jobArgs := commandAndArgs(args[1:])

		jr := jrv1.JobRequest{
			TypeMeta: metav1.TypeMeta{
				APIVersion: jrv1.SchemeGroupVersion.String(),
				Kind:       "JobRequest",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      jrName,
				Namespace: namespace,
			},
			Spec: jrv1.JobRequestSpec{
				ContainerFrom: jrv1.JobRequestContainerFrom{
					ContainerName: container,
					PodSpecFrom: jrv1.JobRequestPodSpecFrom{
						Group: resolvedGroupVersion,
						Kind:  resolved.Kind,
						Name:  resourceName,
					},
				},
				Command: jobCommand,
				Args:    jobArgs,
			},
		}

		err = client.CreateJobRequest(jr)
		if err != nil {
			log.Error("Error creating job request", "error", err)
			os.Exit(1)
		}
		log.Info("Job request created")

		sourceWorkloadOutput, err := client.PodSpecFromToResourceName(jr.Spec.ContainerFrom.PodSpecFrom)
		if err != nil {
			log.Warn("Error resolving source workload", "error", err)
			sourceWorkloadOutput = "unknown"
		}

		t := style.KVTable()
		t.Row("Name", jr.Name)
		t.Row("Command", fmt.Sprintf("%s %s", jr.Spec.Command, shellescape.QuoteCommand(jr.Spec.Args)))
		t.Row("Source Workload", fmt.Sprintf("%s/%s", sourceWorkloadOutput, resourceName))
		_, err = lipgloss.Println(t)
		cobra.CheckErr(err)

		approveCommand := fmt.Sprintf("$ govuk-cli jobrequest review -n %s %s", shellescape.Quote(jr.Namespace), shellescape.Quote(jr.Name))

		_, err = lipgloss.Println(style.BoldStyle.Render("Review command:"))
		cobra.CheckErr(err)
		_, err = lipgloss.Println(style.CommandStyle.Render(approveCommand))
		cobra.CheckErr(err)

		if follow {
			err = client.FollowJobRequest(jr.Name)
			if err != nil {
				log.Error("Error following job request", "error", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	jobrequestCmd.AddCommand(createCmd)

	createCmd.Flags().StringP("container", "c", "app", "Name of the container to pull configuration from")
}

func commandAndArgs(parts []string) (string, []string) {
	command := parts[0]
	args := parts[1:]
	return command, args
}

func generateJrName(sourceWorkload string) string {
	num := strconv.Itoa(int(rand.Int32()))
	return fmt.Sprintf("jr-%s-%s", sourceWorkload, num)
}
