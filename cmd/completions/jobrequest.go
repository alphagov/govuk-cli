package completions

import (
	"fmt"
	"strings"

	"github.com/alphagov/govuk-cli/internal/jobrequest"
	"github.com/alphagov/govuk-cli/internal/kubernetes"
	"github.com/alphagov/govuk-cli/internal/whoami"
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
)

func JobRequest(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	prefix := jobRequestCompletionPrefix(toComplete)

	trimmedToComplete := strings.TrimPrefix(toComplete, prefix)

	kubeconfigFlag := cmd.Flags().Lookup("kubeconfig")
	kubeConfig, err := kubernetes.CreateKubeConfig(kubeconfigFlag)
	if err != nil {
		cobra.CompError(fmt.Sprintf("Failed to create kubeconfig with error: %s", err.Error()))
		return []cobra.Completion{}, cobra.ShellCompDirectiveError
	}

	jobRequests, err := getJobRequests(cmd, kubeConfig)
	if err != nil {
		cobra.CompError(fmt.Sprintf("Failed to get JobRequests with error: %s", err.Error()))
		return []cobra.Completion{}, cobra.ShellCompDirectiveError
	}

	filtereredJobRequestNames := []cobra.Completion{}

	for _, jobRequest := range jobRequests {
		if strings.HasPrefix(jobRequest.Name, trimmedToComplete) {
			filtereredJobRequestNames = append(filtereredJobRequestNames, prefix+jobRequest.Name)
		}
	}

	return filtereredJobRequestNames, cobra.ShellCompDirectiveNoFileComp
}

func ReviewableJobRequest(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	prefix := jobRequestCompletionPrefix(toComplete)

	trimmedToComplete := strings.TrimPrefix(toComplete, prefix)

	kubeconfigFlag := cmd.Flags().Lookup("kubeconfig")
	kubeConfig, err := kubernetes.CreateKubeConfig(kubeconfigFlag)
	if err != nil {
		cobra.CompError(fmt.Sprintf("Failed to create kubeconfig with error: %s", err.Error()))
		return []cobra.Completion{}, cobra.ShellCompDirectiveError
	}

	jobRequests, err := getJobRequests(cmd, kubeConfig)
	if err != nil {
		cobra.CompError(fmt.Sprintf("Failed to get JobRequests with error: %s", err.Error()))
		return []cobra.Completion{}, cobra.ShellCompDirectiveError
	}

	whoamiClient, err := whoami.CreateWhoAmIClient(cmd.Context(), kubeConfig)
	if err != nil {
		cobra.CompError(fmt.Sprintf("Failed to create whoami client with error: %s", err.Error()))
		return []cobra.Completion{}, cobra.ShellCompDirectiveError
	}

	userIdentity, err := whoamiClient.WhoAmI()
	if err != nil {
		cobra.CompError(fmt.Sprintf("Failed to query current user identity with error: %s", err.Error()))
		return []cobra.Completion{}, cobra.ShellCompDirectiveError
	}

	filtereredJobRequestNames := []cobra.Completion{}

	for _, jobRequest := range jobRequests {
		createdByCurrentUser, err := jobRequestWasCreatedByUser(jobRequest, userIdentity)
		if err != nil {
			cobra.CompError(
				fmt.Sprintf(
					"Unable to tell if jobrequest %s was created by current user, error: %s",
					jobRequest.Name, err.Error(),
				),
			)
		}

		if strings.HasPrefix(jobRequest.Name, trimmedToComplete) && !createdByCurrentUser && !jobRequest.HasBeenReviewed() {
			filtereredJobRequestNames = append(filtereredJobRequestNames, prefix+jobRequest.Name)
		}
	}

	return filtereredJobRequestNames, cobra.ShellCompDirectiveNoFileComp
}

func jobRequestWasCreatedByUser(jobRequest *jrv1.JobRequest, userIdentity *jrv1.UserIdentity) (bool, error) {
	requesterArn, err := jobRequest.GetRequestedBy()
	if err != nil {
		return false, err
	}

	requesterUserIdentity, err := jrv1.ParseUserIdentityFromARN(requesterArn)
	if err != nil {
		return false, err
	}

	return userIdentity.UserName == requesterUserIdentity.UserName, nil
}

func jobRequestCompletionPrefix(toComplete string) string {
	var prefix string

	if strings.HasPrefix(toComplete, "jobrequests/") {
		prefix = "jobrequests/"
	} else if strings.HasPrefix(toComplete, "jobrequest/") {
		prefix = "jobrequest/"
	} else if strings.HasPrefix(toComplete, "jr/") {
		prefix = "jr/"
	}

	return prefix
}

func getJobRequests(cmd *cobra.Command, kubeConfig *rest.Config) ([]*jrv1.JobRequest, error) {
	paginationLimit, err := cmd.Flags().GetInt64("pagination-limit")
	if err != nil {
		return []*jrv1.JobRequest{}, err
	}

	client, err := jobrequest.CreateJobRequestClient(cmd.Context(), kubeConfig, cmd.Flag("namespace").Value.String())
	if err != nil {
		return []*jrv1.JobRequest{}, err
	}

	jobRequests, err := client.ListJobRequests(nil, paginationLimit)
	if err != nil {
		return []*jrv1.JobRequest{}, err
	}

	return jobRequests, nil
}
