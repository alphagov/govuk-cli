package kubernetes

import (
	"context"

	"charm.land/log/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	restclient "k8s.io/client-go/rest"
)

type AppsV1Client struct {
	appsv1Client *appsv1client.AppsV1Client
	ctx          context.Context
}

func CreateAppsV1Client(ctx context.Context, kubeRestClientConfig *restclient.Config) (*AppsV1Client, error) {
	log.Debug("create appsv1 client")

	client, err := appsv1client.NewForConfig(kubeRestClientConfig)
	if err != nil {
		return nil, err
	}

	return &AppsV1Client{
		appsv1Client: client,
		ctx:          ctx,
	}, nil
}

func (c *AppsV1Client) GetDeploymentNames(namespace string, paginationLimit int64) ([]string, error) {
	listOptions := metav1.ListOptions{
		Limit:          paginationLimit,
		TimeoutSeconds: new(int64(3)),
	}

	deploymentNames := []string{}
	for {
		deploymentList, err := c.appsv1Client.Deployments(namespace).List(c.ctx, listOptions)
		if err != nil {
			return deploymentNames, err
		}

		for _, deployment := range deploymentList.Items {
			deploymentNames = append(deploymentNames, deployment.Name)
		}

		if deploymentList.Continue == "" {
			break
		}

		listOptions.Continue = deploymentList.Continue
	}

	return deploymentNames, nil
}

func (c *AppsV1Client) GetContainerNamesInDeployment(namespace, deploymentName string) ([]string, error) {
	containerNames := []string{}

	deployment, err := c.appsv1Client.Deployments(namespace).Get(c.ctx, deploymentName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) || apierrors.IsGone(err) {
		return containerNames, nil
	} else if err != nil {
		return containerNames, err
	}

	containersInDeployment := deployment.Spec.Template.Spec.Containers

	for _, container := range containersInDeployment {
		containerNames = append(containerNames, container.Name)
	}

	return containerNames, nil
}
