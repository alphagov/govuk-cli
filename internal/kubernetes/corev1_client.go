package kubernetes

import (
	"context"

	"charm.land/log/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
)

type CoreV1Client struct {
	corev1Client *corev1client.CoreV1Client
	ctx          context.Context
}

func CreateCoreV1Client(ctx context.Context, kubeRestClientConfig *restclient.Config) (*CoreV1Client, error) {
	log.Debug("create corev1 client")

	client, err := corev1client.NewForConfig(kubeRestClientConfig)
	if err != nil {
		return nil, err
	}

	return &CoreV1Client{
		corev1Client: client,
		ctx:          ctx,
	}, nil
}

func (c *CoreV1Client) GetNamespaceNames(paginationLimit int64) ([]string, error) {
	listOptions := metav1.ListOptions{
		Limit:          paginationLimit,
		TimeoutSeconds: new(int64(3)),
	}

	namespaceNames := []string{}
	for {
		namespaceList, err := c.corev1Client.Namespaces().List(c.ctx, listOptions)
		if err != nil {
			return namespaceNames, err
		}

		for _, namespace := range namespaceList.Items {
			namespaceNames = append(namespaceNames, namespace.Name)
		}

		if namespaceList.Continue == "" {
			break
		}

		listOptions.Continue = namespaceList.Continue
	}

	return namespaceNames, nil
}
