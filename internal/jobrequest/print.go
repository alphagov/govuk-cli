package jobrequest

import (
	"fmt"

	"al.essio.dev/pkg/shellescape"
	"charm.land/lipgloss/v2/table"
	"charm.land/log/v2"
	"github.com/alphagov/govuk-cli/internal/style"
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
)

// get a lipgloss table filled with a list of JobRequests
func (c *JobRequestClient) JobRequestDetailsListTable(jrs []*jrv1.JobRequest) (*table.Table, error) {
	headers := []string{
		// For now leaving out reviewed name since it means calling the API for every JR to get the JRR
		"Name", "State", "Created By" /*"Reviewed By",*/, "Created Time",
	}

	t := style.ListTable(headers)

	for _, jobRequest := range jrs {
		row, err := c.addJobRequestListRow(jobRequest)
		if err != nil {
			return nil, err
		}
		t.Row(row...)
	}

	return t, nil
}

func (c *JobRequestClient) addJobRequestListRow(jr *jrv1.JobRequest) ([]string, error) {
	jobName := jr.Name
	state := string(jr.Status.State)
	createdByArn, err := jr.GetRequestedBy()
	if err != nil {
		return []string{}, err
	}
	createdByUserIdentity, err := jrv1.ParseUserIdentityFromARN(createdByArn)
	if err != nil {
		return []string{}, err
	}

	createdAt := jr.GetCreationTimestamp().Format("2006/01/02 15:04:05")

	return []string{jobName, state, createdByUserIdentity.UserName, createdAt}, nil
}

// get a lipgloss table filled with details of a JobRequest resource
func (c *JobRequestClient) JobRequestDetailsKVTable(jr *jrv1.JobRequest) (*table.Table, error) {
	requestedBy, err := jr.GetRequestedBy()
	if err != nil {
		requestedBy = "-"
	}

	userIdentity, err := jrv1.ParseUserIdentityFromARN(requestedBy)
	if err == nil {
		log.Debug("Parsed requester arn", "arn", userIdentity)
		requestedBy = fmt.Sprintf("%s (%s)", userIdentity.UserName, userIdentity.RoleName)
	} else {
		log.Error("Error parsing requester ARN", "arn", requestedBy, "error", err)
	}

	state := jr.Status.State
	if state == "" {
		state = "Unknown"
	}

	sourceWorkloadKind, err := c.PodSpecFromToResourceName(jr.Spec.ContainerFrom.PodSpecFrom)
	if err != nil {
		return nil, err
	}

	sourceWorkload := fmt.Sprintf("%s/%s", sourceWorkloadKind, jr.Spec.ContainerFrom.PodSpecFrom.Name)

	t := style.KVTable()
	t.Row("Job Request Name", jr.Name)
	t.Row("Command", fmt.Sprintf("%s %s", jr.Spec.Command, shellescape.QuoteCommand(jr.Spec.Args)))
	t.Row("Source Workload", sourceWorkload)
	t.Row("Status", string(state))
	t.Row("Requested By", requestedBy)

	if jr.Status.JobName != "" {
		t.Row("Job Name", jr.Status.JobName)
	}

	return t, nil
}
