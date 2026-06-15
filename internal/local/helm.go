package local

import (
	"embed"
	"io/fs"
	"path"
	"strings"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart/loader/archive"
	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/chart/v2/loader"
	"helm.sh/helm/v4/pkg/release"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"

	l "github.com/alphagov/govuk-cli/internal/logger"
	"github.com/sirupsen/logrus"
)

//go:embed charts
var charts embed.FS

func embeddedChartLoad(kind string, name string) (*chartv2.Chart, error) {
	l.Log().WithFields(logrus.Fields{
		"name": name,
		"kind": kind,
	}).Debug("loading embedded chart")

	chartPath := path.Join("charts", kind, name)
	var files []*archive.BufferedFile
	err := fs.WalkDir(charts, chartPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := charts.ReadFile(path)
		if err != nil {
			return err
		}
		fileName := strings.TrimPrefix(path, chartPath+"/")
		files = append(files, &archive.BufferedFile{Name: fileName, Data: data})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return loader.LoadFiles(files)
}

func helmConfig(restConfig *rest.Config) *action.Configuration {
	cf := genericclioptions.NewConfigFlags(false)
	cf.WrapConfigFn = func(*rest.Config) *rest.Config { return restConfig }

	cfg := new(action.Configuration)
	cfg.Init(cf, "default", "secret")

	return cfg
}

func helmListInstalled(cfg *action.Configuration) ([]release.Releaser, error) {
	client := action.NewList(cfg)
	client.Deployed = true
	client.Failed = true
	client.Pending = true
	client.SetStateMask()

	return client.Run()
}

func helmIsInstalled(releaseName string, releases []release.Releaser) (bool, error) {
	for _, r := range releases {
		acc, err := release.NewAccessor(r)
		if err != nil {
			return false, err
		}
		if acc.Name() == releaseName {
			return true, nil
		}
	}
	return false, nil
}

func helmUpgrade(releaseName string, chart *chartv2.Chart, values map[string]any, cfg *action.Configuration) error {
	l.Log().WithFields(logrus.Fields{
		"release": releaseName,
		"chart":   chart.Metadata.Name,
		"version": chart.Metadata.Version,
	}).Debug("performing Helm upgrade")

	client := action.NewUpgrade(cfg)
	client.Namespace = "default"
	client.WaitStrategy = "watcher"
	client.DryRunStrategy = "none"
	client.ForceConflicts = true

	_, err := client.Run(releaseName, chart, values)
	return err
}

func helmInstall(releaseName string, chart *chartv2.Chart, values map[string]any, cfg *action.Configuration) error {
	l.Log().WithFields(logrus.Fields{
		"release": releaseName,
		"chart":   chart.Metadata.Name,
		"version": chart.Metadata.Version,
	}).Debug("performing Helm install")

	client := action.NewInstall(cfg)
	client.ReleaseName = releaseName
	client.Namespace = "default"
	client.CreateNamespace = true
	client.WaitStrategy = "watcher"
	client.DryRunStrategy = "none"

	_, err := client.Run(chart, values)
	return err
}

func HelmChartEnsure(releaseName string, chartKind string, chartName string, values map[string]any, restConfig *rest.Config) error {
	cfg := helmConfig(restConfig)

	chart, err := embeddedChartLoad(chartKind, chartName)
	if err != nil {
		return err
	}

	results, err := helmListInstalled(cfg)

	if err != nil {
		return err
	}

	isInstalled, err := helmIsInstalled(releaseName, results)

	if err != nil {
		return err
	}

	if isInstalled {
		return helmUpgrade(
			releaseName,
			chart,
			values,
			cfg,
		)
	} else {
		return helmInstall(
			releaseName,
			chart,
			values,
			cfg,
		)
	}
}
