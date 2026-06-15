package local

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/config"
	conf "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	l "github.com/alphagov/govuk-cli/internal/logger"
)

type ClusterConfig struct {
	Name         string
	RegistryPort int
	ApiPort      int
	AppPort      int
}

func (c ClusterConfig) RegistryHost() string {
	return fmt.Sprintf("k3d-%s-registry:%d", c.Name, c.RegistryPort)
}

var rt = runtimes.SelectedRuntime

func k3dClusterCreate(cfg ClusterConfig) (*k3d.Cluster, error) {
	ctx := context.Background()

	c := conf.SimpleConfig{
		Image:   fmt.Sprintf("%s:%s", k3d.DefaultK3sImageRepo, "v1.34.8-k3s1"),
		Servers: 1,
		Agents:  1,
		Options: conf.SimpleConfigOptions{
			K3dOptions: conf.SimpleConfigOptionsK3d{Wait: true},
		},
		ExposeAPI: conf.SimpleExposureOpts{
			HostIP:   "0.0.0.0",
			Host:     "0.0.0.0",
			HostPort: strconv.Itoa(cfg.ApiPort),
		},
		Ports: []conf.PortWithNodeFilters{
			{
				Port:        fmt.Sprintf("%d:80", cfg.AppPort),
				NodeFilters: []string{"loadbalancer"},
			},
		},
	}
	c.Name = cfg.Name
	c.Registries.Create = &conf.SimpleConfigRegistryCreateConfig{
		Name:     "k3d-" + cfg.Name + "-registry",
		HostPort: strconv.Itoa(cfg.RegistryPort),
	}

	clusterCfg, err := config.TransformSimpleToClusterConfig(
		ctx,
		rt,
		c,
		cfg.Name,
	)
	if err != nil {
		return nil, err
	}

	err = client.ClusterRun(ctx, rt, clusterCfg)
	if err != nil {
		return nil, err
	}
	return client.ClusterGet(ctx, rt, &k3d.Cluster{Name: cfg.Name})
}

func k3dClusterDelete(cluster *k3d.Cluster) error {
	return client.ClusterDelete(context.Background(), rt, cluster, k3d.ClusterDeleteOpts{SkipRegistryCheck: false})
}

func k3dClusterStart(cluster *k3d.Cluster) error {
	ctx := context.Background()

	envInfo, err := client.GatherEnvironmentInfo(ctx, rt, cluster)
	if err != nil {
		return err
	}

	startOpts, err := client.GetClusterStartOptsFromLabels(cluster)
	if err != nil {
		return err
	}

	opts := k3d.ClusterStartOpts{
		WaitForServer:   true,
		Intent:          k3d.IntentClusterStart,
		EnvironmentInfo: envInfo,
		HostAliases:     startOpts.HostAliases,
	}

	return client.ClusterStart(ctx, rt, cluster, opts)
}

func K3dClusterEnsure(config ClusterConfig) (*k3d.Cluster, error) {
	l.Log().WithFields(logrus.Fields{
		"name":         config.Name,
		"registryPort": config.RegistryPort,
	}).Info("ensuring cluster")

	ctx := context.Background()
	cluster, err := client.ClusterGet(ctx, rt, &k3d.Cluster{Name: config.Name})
	if err != nil {
		if errors.Is(err, client.ClusterGetNoNodesFoundError) {
			// cluster doesn't exist
			l.Log().WithFields(logrus.Fields{
				"name": config.Name,
			}).Info("creating cluster")
			return k3dClusterCreate(config)
		}
		return nil, err
	}

	_, agentsRunning := cluster.AgentCountRunning()

	if agentsRunning == 0 {
		l.Log().WithFields(logrus.Fields{
			"name": config.Name,
		}).Info("starting stopped cluster")

		err = k3dClusterStart(cluster)

		return cluster, err
	}

	l.Log().WithFields(logrus.Fields{
		"name": config.Name,
	}).Info("cluster already running")

	return cluster, nil
}

func K3dClusterDestroy(config ClusterConfig) error {
	l.Log().WithFields(logrus.Fields{
		"name": config.Name,
	}).Info("destroying cluster")

	ctx := context.Background()
	cluster, err := client.ClusterGet(ctx, rt, &k3d.Cluster{Name: config.Name})
	if err != nil {
		if errors.Is(err, client.ClusterGetNoNodesFoundError) {
			// do nothing if the cluster doesn't exist
			l.Log().WithFields(logrus.Fields{
				"name": config.Name,
			}).Info("no cluster to destroy")

			return nil
		}
		return err
	}
	return k3dClusterDelete(cluster)
}

func RestConfigGet(cluster *k3d.Cluster) (*rest.Config, error) {
	kubeconfig, err := client.KubeconfigGet(context.Background(), rt, cluster)
	if err != nil {
		return nil, err
	}
	clientConfig := clientcmd.NewDefaultClientConfig(
		*kubeconfig,
		&clientcmd.ConfigOverrides{}, //&clientcmd.ConfigOverrides{CurrentContext: "k3d-" + cluster.Name},
	)
	return clientConfig.ClientConfig()
}
