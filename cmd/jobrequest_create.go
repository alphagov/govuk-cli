/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"

	"charm.land/log/v2"
	"github.com/alphagov/govuk-cli/internal/jobrequest"
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
	Short:   "Create a new job requestß",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			log.Error("not enough arguments provided")
			cobra.CheckErr(cmd.Usage())
			os.Exit(1)
		}

		kubeconfig, err := cmd.Flags().GetString("kubeconfig")
		if err != nil {
			log.Error("error getting kubeconfig flag value", "error", err)
			os.Exit(1)
		}

		namespace, err := cmd.Flags().GetString("namespace")
		if err != nil {
			log.Error("error getting namespace flag value", "error", err)
			os.Exit(1)
		}

		container, err := cmd.Flags().GetString("container")
		if err != nil {
			log.Error("error getting container flag value", "error", err)
			os.Exit(1)
		}

		client, err := jobrequest.CreateJobRequestClient(kubeconfig, namespace)
		if err != nil {
			log.Error("error creating job request client", "error", err)
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
			log.Error("error resolving workload kind", "error", err)
			os.Exit(1)
		}
		log.Debug(resourceName)
		log.Debug("resolved workload", "kind", resolved)
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
		out, err := json.Marshal(jr)
		log.Debug("built JobRequest")
		log.Debug(string(out))

		err = client.CreateJobRequest(jr)
		if err != nil {
			log.Error("error creating job request", "error", err)
			os.Exit(1)
		}
		log.Info("job request created")
	},
}

func init() {
	jobrequestCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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
