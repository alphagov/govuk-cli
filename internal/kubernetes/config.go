package kubernetes

import (
	"charm.land/log/v2"
	"github.com/spf13/pflag"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func CreateKubeConfig(kubeConfigPathFlag *pflag.Flag) (*restclient.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeConfigPathFlag != nil && kubeConfigPathFlag.Changed {
		loadingRules.ExplicitPath = kubeConfigPathFlag.Value.String()
		log.Debug("creating kubernetes client", "kubeconfig", loadingRules.ExplicitPath)
	} else {
		log.Debug("creating kubernetes client", "kubeconfig", "loaded from environment")
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return kubeConfig.ClientConfig()
}
