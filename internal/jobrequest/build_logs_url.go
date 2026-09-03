package jobrequest

import (
	"errors"
	"fmt"
	"time"

	"charm.land/log/v2"
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
)

type (
	CurrentAccountId string
)

const (
	baseKibanaUrl                       = "https://kibana.logit.io/s/"
	kibanaPath                          = "/app/data-explorer/discover#?"
	logitIntEnvUid                      = "42f4d2d5-e9ce-451f-8ffc-cdb25bd624f8"
	logitStagingEnvUid                  = "b8a10798-a30e-4611-9393-8843d2339dd2"
	logitProdEnvUid                     = "13d1a0b1-f54f-407b-a4e5-f53ba653fac3"
	awsIntegration     CurrentAccountId = "210287912431"
	awsStaging         CurrentAccountId = "696911096973"
	awsProduction      CurrentAccountId = "172025368201"
)

func BuildLogsUrl(jobRequest *jrv1.JobRequest, review *jrv1.JobRequestReview, currAccountId CurrentAccountId) string {
	urlPath, pathErr := buildParams(jobRequest, review)
	if pathErr != nil {
		log.Debugf("Unable to build OpenSearch logs _path_ part of the url: %s\n", pathErr.Error())
		return ""
	}

	switch currAccountId {
	case awsIntegration:
		return baseKibanaUrl + logitIntEnvUid + urlPath
	case awsStaging:
		return baseKibanaUrl + logitStagingEnvUid + urlPath
	case awsProduction:
		return baseKibanaUrl + logitProdEnvUid + urlPath
	}

	log.Debugf("Environment does not have a OpenSearch cluster, so cannot build proper OpenSearch logs url\n")

	return ""
}

func buildParams(jobRequest *jrv1.JobRequest, jobRequestReview *jrv1.JobRequestReview) (string, error) {
	urlTimeParam, timeParamErr := buildTimeParamStr(jobRequestReview)
	if timeParamErr != nil {
		return "", errors.New("the job has not started, so cannot calculate the time range for the logs url")
	}

	urlQueryParam := buildQueryParamStr(jobRequest.Status.JobName)
	urlDiscoverParam := buildDiscoverParamStr(jobRequest.CreationTimestamp.Format("2006.01.02"))

	urlPath := kibanaPath + *urlTimeParam + urlDiscoverParam + urlQueryParam

	return urlPath, nil
}

func buildTimeParamStr(review *jrv1.JobRequestReview) (*string, error) {
	if review.Status.State != jrv1.JobRequestReviewApproved {
		return nil, errors.New("the job has not started, so cannot calculate the time range for the logs url")
	}

	timeParamStr := fmt.Sprintf("_g=(time:(from:'%s',to:'%s'))", review.CreationTimestamp.UTC().Format(time.RFC3339), review.CreationTimestamp.Add(time.Hour*3).UTC().Format(time.RFC3339))

	return &timeParamStr, nil
}

func buildQueryParamStr(jobName string) string {
	return fmt.Sprintf("&_q=(query:(language:kuery,query:'kubernetes.job.name:%%20%s'))", jobName)
}

func buildDiscoverParamStr(jobDate string) string {
	return fmt.Sprintf("&_a=(discover:(metadata:(indexPattern:'filebeat-%s',view:discover)))", jobDate)
}
