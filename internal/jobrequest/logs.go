package jobrequest

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sort"

	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	FollowLogs   bool = true
	NoFollowLogs bool = false
)

func (c *JobRequestClient) PrintLogs(jobRequest *jrv1.JobRequest) error {
	jobName := jobRequest.Status.JobName
	if jobName == "" {
		return errors.New("JobRequest has not yet started a Job, cannot print logs")
	}

	// This is really to give more helpful errors to the user than just not found
	_, err := c.clientSet.BatchV1().Jobs(c.namespace).Get(c.ctx, jobName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf(
				"the JobRequest spawned Job %s but that Job was not found, cannot print logs",
				jobName,
			)
		} else if apierrors.IsGone(err) {
			return fmt.Errorf(
				"the JobRequest spawned Job %s, but that Job has already been garbage collected, cannot print logs",
				jobRequest.Status.JobName,
			)
		} else {
			return err
		}
	}

	pods, err := c.clientSet.CoreV1().Pods(c.namespace).List(c.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil {
		return err
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf(
			"no pods found for Job %s which was spawned by this JobRequest, it has either not yet started, or has been garbage collected",
			jobName,
		)
	}
	sort.Slice(pods.Items, func(i, j int) bool {
		return pods.Items[i].CreationTimestamp.After(pods.Items[j].CreationTimestamp.Time)
	})
	podName := pods.Items[0].Name

	return streamLogs(c, podName, NoFollowLogs)
}

func streamLogs(c *JobRequestClient, podName string, follow bool) error {
	logs := c.clientSet.CoreV1().Pods(c.namespace).GetLogs(
		podName,
		&corev1.PodLogOptions{Follow: follow},
	)
	stream, err := logs.Stream(c.ctx)
	if err != nil {
		return err
	}
	defer stream.Close() //nolint:errcheck
	reader := bufio.NewReader(stream)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		fmt.Print(line)
	}
}
