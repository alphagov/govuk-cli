package jobrequest

import (
	"time"

	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func newTestJobRequest() *jrv1.JobRequest {
	return &jrv1.JobRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-job-request",
			CreationTimestamp: metav1.NewTime(time.Date(2026, 12, 25, 10, 0, 0, 0, time.UTC)),
		},
		Status: jrv1.JobRequestStatus{
			JobName: "test-job-name",
			State:   jrv1.JobRequestStarted,
		},
	}
}

func newTestReview() *jrv1.JobRequestReview {
	return &jrv1.JobRequestReview{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-review",
			CreationTimestamp: metav1.NewTime(time.Date(2026, 12, 25, 11, 0, 0, 0, time.UTC)),
		},
		Status: jrv1.JobRequestReviewStatus{
			State: jrv1.JobRequestReviewApproved,
		},
	}
}

func newBstAdjustedJobRequestReview() *jrv1.JobRequestReview {
	bstZone := time.FixedZone("BST", 1*60*60)
	jobReview := newTestReview()
	jobReview.CreationTimestamp = metav1.NewTime(time.Date(2026, 12, 25, 11, 0, 0, 0, bstZone))
	return jobReview
}

func newRejectedReview() *jrv1.JobRequestReview {
	review := newTestReview()
	review.Status.State = jrv1.JobRequestReviewRejected
	return review
}

var _ = Describe("BuildLogsUrl", func() {
	DescribeTable(
		"builds the kibana logs url",
		func(jobRequest *jrv1.JobRequest, review *jrv1.JobRequestReview, currAccountId CurrentAccountId, want string) {
			Expect(BuildLogsUrl(jobRequest, review, currAccountId)).To(Equal(want))
		},
		Entry(
			"integration env",
			newTestJobRequest(),
			newTestReview(),
			awsIntegration,
			"https://kibana.logit.io/s/42f4d2d5-e9ce-451f-8ffc-cdb25bd624f8/app/data-explorer/discover#?_g=(time:(from:'2026-12-25T11:00:00Z',to:'2026-12-25T14:00:00Z'))&_a=(discover:(metadata:(indexPattern:'filebeat-2026.12.25',view:discover)))&_q=(query:(language:kuery,query:'kubernetes.job.name:%20test-job-name'))",
		),
		Entry(
			"staging env",
			newTestJobRequest(),
			newTestReview(),
			awsStaging,
			"https://kibana.logit.io/s/b8a10798-a30e-4611-9393-8843d2339dd2/app/data-explorer/discover#?_g=(time:(from:'2026-12-25T11:00:00Z',to:'2026-12-25T14:00:00Z'))&_a=(discover:(metadata:(indexPattern:'filebeat-2026.12.25',view:discover)))&_q=(query:(language:kuery,query:'kubernetes.job.name:%20test-job-name'))",
		),
		Entry(
			"production env",
			newTestJobRequest(),
			newTestReview(),
			awsProduction,
			"https://kibana.logit.io/s/13d1a0b1-f54f-407b-a4e5-f53ba653fac3/app/data-explorer/discover#?_g=(time:(from:'2026-12-25T11:00:00Z',to:'2026-12-25T14:00:00Z'))&_a=(discover:(metadata:(indexPattern:'filebeat-2026.12.25',view:discover)))&_q=(query:(language:kuery,query:'kubernetes.job.name:%20test-job-name'))",
		),
		Entry(
			"bst adjusted creation time",
			newTestJobRequest(),
			newBstAdjustedJobRequestReview(),
			awsIntegration,
			"https://kibana.logit.io/s/42f4d2d5-e9ce-451f-8ffc-cdb25bd624f8/app/data-explorer/discover#?_g=(time:(from:'2026-12-25T10:00:00Z',to:'2026-12-25T13:00:00Z'))&_a=(discover:(metadata:(indexPattern:'filebeat-2026.12.25',view:discover)))&_q=(query:(language:kuery,query:'kubernetes.job.name:%20test-job-name'))",
		),
		Entry(
			"review not approved returns empty string",
			newTestJobRequest(),
			newRejectedReview(),
			awsIntegration,
			"",
		),
		Entry(
			"unknown env returns empty string",
			newTestJobRequest(),
			newTestReview(),
			CurrentAccountId("123456"),
			"",
		),
	)
})

var _ = Describe("buildParams", func() {
	It("builds the url path for an approved review", func() {
		got, err := buildParams(newTestJobRequest(), newTestReview())
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal("/app/data-explorer/discover#?_g=(time:(from:'2026-12-25T11:00:00Z',to:'2026-12-25T14:00:00Z'))&_a=(discover:(metadata:(indexPattern:'filebeat-2026.12.25',view:discover)))&_q=(query:(language:kuery,query:'kubernetes.job.name:%20test-job-name'))"))
	})

	It("returns an error for an unapproved review", func() {
		got, err := buildParams(newTestJobRequest(), newRejectedReview())
		Expect(err).To(HaveOccurred())
		Expect(got).To(BeEmpty())
	})
})

var _ = Describe("buildTimeParamStr", func() {
	It("builds the time range for an approved review", func() {
		got, err := buildTimeParamStr(newTestReview())
		Expect(err).NotTo(HaveOccurred())
		Expect(got).NotTo(BeNil())
		Expect(*got).To(Equal("_g=(time:(from:'2026-12-25T11:00:00Z',to:'2026-12-25T14:00:00Z'))"))
	})

	It("returns an error for an unapproved review", func() {
		got, err := buildTimeParamStr(newRejectedReview())
		Expect(err).To(HaveOccurred())
		Expect(got).To(BeNil())
	})
})

var _ = Describe("buildQueryParamStr", func() {
	It("builds the query param for the job name", func() {
		Expect(buildQueryParamStr("my-job")).To(Equal("&_q=(query:(language:kuery,query:'kubernetes.job.name:%20my-job'))"))
	})
})

var _ = Describe("buildDiscoverParamStr", func() {
	It("builds the discover param for the job date", func() {
		Expect(buildDiscoverParamStr("2026.12.25")).To(Equal("&_a=(discover:(metadata:(indexPattern:'filebeat-2026.12.25',view:discover)))"))
	})
})
