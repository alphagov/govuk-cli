package jobrequest

import (
	"fmt"

	"al.essio.dev/pkg/shellescape"
	"charm.land/lipgloss/v2/table"
	"charm.land/log/v2"
	"github.com/alphagov/govuk-cli/internal/style"
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
)

// get a lipgloss table filled with details of a JobRequest resource
func (c *JobRequestClient) JobRequestDetailsKVTable(jr *jrv1.JobRequest) (*table.Table, error) {
	requestedBy, isOk := jr.Annotations[RequestedByAnnotation]
	if !isOk {
		requestedBy = "-"
	}

	arn, err := ParseAssumedRoleArn(requestedBy)
	if err == nil {
		log.Debug("Parsed requester arn", "arn", arn)
		requestedBy = fmt.Sprintf("%s (%s)", arn.UserName, arn.RoleName)
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
	t.Row("Status", state)
	t.Row("Requested By", requestedBy)

	return t, nil
}
