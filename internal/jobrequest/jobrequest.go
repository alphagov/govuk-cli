package jobrequest

import (
	"context"

	"charm.land/log/v2"
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

const (
	JobRequestResourceName       = "jobrequests"
	JobRequestReviewResourceName = "jobrequestreviews"
)

type JobRequestClient struct {
	namespace     string
	dynamicClient *dynamic.DynamicClient
	clientSet     *kubernetes.Clientset
	ctx           context.Context
}

func (c *JobRequestClient) InterfaceFor(resourceName string) dynamic.ResourceInterface {
	return c.dynamicClient.Resource(jrv1.SchemeGroupVersion.WithResource(resourceName)).Namespace(c.namespace)
}

func (c *JobRequestClient) JobRequest(jobRequestName string) (*jrv1.JobRequest, error) {
	i := c.InterfaceFor(JobRequestResourceName)
	unstructured, err := i.Get(c.ctx, jobRequestName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	jobRequest := jrv1.JobRequest{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.Object, &jobRequest)
	if err != nil {
		return nil, err
	}

	return &jobRequest, nil
}

func (c *JobRequestClient) CreateJobRequest(jobRequest jrv1.JobRequest) error {
	unstructuredJr, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&jobRequest)
	if err != nil {
		return err
	}
	i := c.InterfaceFor(JobRequestResourceName)
	res, err := i.Create(c.ctx, &unstructured.Unstructured{Object: unstructuredJr}, metav1.CreateOptions{})
	log.Debug("job request create result", "res", res)
	return err
}

func (c *JobRequestClient) JobRequestReview(jobRequestReviewName string) (*jrv1.JobRequestReview, error) {
	i := c.InterfaceFor(JobRequestReviewResourceName)
	unstructured, err := i.Get(c.ctx, jobRequestReviewName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	jobRequestReview := jrv1.JobRequestReview{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.Object, &jobRequestReview)
	if err != nil {
		return nil, err
	}

	return &jobRequestReview, nil
}

func (c *JobRequestClient) CreateJobRequestReview(jobRequestReview jrv1.JobRequestReview) error {
	unstructuredJrr, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&jobRequestReview)
	if err != nil {
		return err
	}
	i := c.InterfaceFor(JobRequestReviewResourceName)
	res, err := i.Create(c.ctx, &unstructured.Unstructured{Object: unstructuredJrr}, metav1.CreateOptions{})
	log.Debug("job request review create result", "res", res)
	return err
}

func CreateJobRequestClient(kubeRestClientConfig *restclient.Config, namespace string) (*JobRequestClient, error) {
	log.Debug("creating job request client")
	dynamic, err := dynamic.NewForConfig(kubeRestClientConfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(kubeRestClientConfig)
	if err != nil {
		return nil, err
	}

	return &JobRequestClient{
		clientSet:     clientset,
		dynamicClient: dynamic,
		ctx:           context.Background(),
		namespace:     namespace,
	}, nil
}
