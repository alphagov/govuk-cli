package jobrequest

import (
	"context"
	"sort"

	"charm.land/log/v2"
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func (c *JobRequestClient) ListJobRequests(forUser *jrv1.UserIdentity, apiPaginationLimit int64) ([]*jrv1.JobRequest, error) {
	jobRequests := []*jrv1.JobRequest{}

	listOptions := metav1.ListOptions{
		Limit:    apiPaginationLimit,
		Continue: "",
	}
	groupVersionResource := schema.GroupVersionResource{
		Group:    jrv1.GroupVersion.Group,
		Version:  jrv1.GroupVersion.Version,
		Resource: JobRequestResourceName,
	}
	namespacedInterface := c.dynamicClient.Resource(groupVersionResource).Namespace(c.namespace)

	for {
		unstructuredList, err := namespacedInterface.List(c.ctx, listOptions)
		if err != nil {
			return jobRequests, err
		}

		log.Debugf("listing JobRequests, received %d unstructured items in paginated request", len(unstructuredList.Items))
		structuredJobRequestList, err := c.unstructuredToStructuredJobRequestList(unstructuredList)
		if err != nil {
			return jobRequests, err
		}

		if forUser != nil {
			log.Debug("filtering JobRequests for user", "username", forUser.UserName)
			filteredList, err := c.filterJobRequestListForUser(structuredJobRequestList, *forUser)
			if err != nil {
				return jobRequests, err
			}

			log.Debugf("filtered list from %d down to %d", len(jobRequests), len(filteredList))
			structuredJobRequestList = filteredList
		}

		jobRequests = append(jobRequests, structuredJobRequestList...)

		listOptions.Continue = unstructuredList.GetContinue()
		if listOptions.Continue == "" {
			log.Debugf(
				"listing JobRequests, no pagination continuation token recevied, list complete with %d JobReqeusts",
				len(jobRequests),
			)
			break
		}
	}

	sort.Slice(jobRequests, func(i, j int) bool {
		return jobRequests[i].CreationTimestamp.Before(&jobRequests[j].CreationTimestamp)
	})

	return jobRequests, nil
}
func (c *JobRequestClient) unstructuredToStructuredJobRequestList(unstructuredList *unstructured.UnstructuredList) ([]*jrv1.JobRequest, error) {
	jobRequests := make([]*jrv1.JobRequest, len(unstructuredList.Items))

	for i, unstructuredJobRequest := range unstructuredList.Items {
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredJobRequest.Object, &jobRequests[i])
		if err != nil {
			return jobRequests, err
		}
	}

	return jobRequests, nil
}

func (c *JobRequestClient) filterJobRequestListForUser(jobRequestList []*jrv1.JobRequest, forUser jrv1.UserIdentity) ([]*jrv1.JobRequest, error) {
	jobRequests := []*jrv1.JobRequest{}

	for _, structuredJobRequest := range jobRequestList {
		requestedBy, err := structuredJobRequest.GetRequestedBy()
		if err != nil {
			return jobRequests, err
		}

		userIdentity, err := jrv1.ParseUserIdentityFromARN(requestedBy)
		if err != nil {
			return jobRequests, err
		}

		if forUser.UserName == userIdentity.UserName {
			jobRequests = append(jobRequests, structuredJobRequest)
		}
	}

	return jobRequests, nil
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
