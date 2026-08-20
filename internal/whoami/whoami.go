package whoami

import (
	"context"

	"charm.land/log/v2"
	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	authv1client "k8s.io/client-go/kubernetes/typed/authentication/v1"
	restclient "k8s.io/client-go/rest"
)

type WhoAmIClient struct {
	authv1Client *authv1client.AuthenticationV1Client
	ctx          context.Context
}

func CreateWhoAmIClient(kubeRestClientConfig *restclient.Config) (*WhoAmIClient, error) {
	log.Debug("create whoami client")

	client, err := authv1client.NewForConfig(kubeRestClientConfig)
	if err != nil {
		return nil, err
	}

	return &WhoAmIClient{
		authv1Client: client,
		ctx:          context.Background(),
	}, nil
}

func (c *WhoAmIClient) WhoAmI() (*jrv1.UserIdentity, error) {
	result, err := c.authv1Client.SelfSubjectReviews().Create(
		c.ctx,
		&authv1.SelfSubjectReview{},
		metav1.CreateOptions{},
	)
	if err != nil {
		return nil, err
	}

	userIdentity, err := jrv1.ParseUserIdentityFromARN(result.Status.UserInfo.Username)
	if err != nil {
		return nil, err
	}

	return userIdentity, nil
}
