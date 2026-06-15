package local

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"go.yaml.in/yaml/v4"
	"k8s.io/client-go/rest"
)

type ServiceDefinition struct {
	Name   string         `yaml:"name"`
	Kind   string         `yaml:"kind"`
	Values map[string]any `yaml:"values"`
}

type Application struct {
	Name          string              `yaml:"name"`
	Dependencies  []string            `yaml:"dependencies"`
	Services      []ServiceDefinition `yaml:"services"`
	Values        map[string]any      `yaml:"values"`
	RawChartName  string              `yaml:"chartName"`
	RawImageOwner string              `yaml:"imageOwner"`
	RawImageRepo  string              `yaml:"imageRepo"`
	RawImageTag   string              `yaml:"imageTag"`
}

func (a *Application) ChartName() string {
	if a.RawChartName == "" {
		return "generic"
	}
	return a.RawChartName
}

func (a *Application) ImageOwner() string {
	if a.RawImageOwner == "" {
		return "alphagov"
	}
	return a.RawImageOwner
}

func (a *Application) ImageRepo() string {
	if a.RawImageRepo == "" {
		return fmt.Sprintf("govuk/%s", a.Name)
	}
	return a.RawImageRepo
}

func (a *Application) ImageTag() string {
	if a.RawImageTag != "" {
		return a.RawImageTag
	}
	return ""
}

func readDefinitionFromFile(filePath string) (*Application, error) {
	def := &Application{}

	yamlBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlBytes, def)
	return def, err
}

func LoadDefinitions(dirPath string) (map[string]*Application, error) {
	matches, err := filepath.Glob(filepath.Join(dirPath, "*.yaml"))
	if err != nil {
		return nil, err
	}
	defs := make(map[string]*Application, len(matches))
	for _, path := range matches {
		def, err := readDefinitionFromFile(path)
		if err != nil {
			return nil, err
		}
		defs[def.Name] = def
	}
	return defs, nil
}

func OrderApplications(defs map[string]*Application) ([]*Application, error) {
	seen := make(map[string]bool)
	onPath := make(map[string]bool)
	var out []*Application

	var visit func(name string) error
	visit = func(name string) error {
		app, ok := defs[name]
		if !ok {
			return fmt.Errorf("unknown application %q", name)
		}
		if seen[name] {
			return nil
		}
		if onPath[name] {
			return fmt.Errorf("dependency cycle at %q", name)
		}
		onPath[name] = true
		for _, dep := range app.Dependencies {
			err := visit(dep)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
		delete(onPath, name)
		seen[name] = true
		out = append(out, app)
		return nil
	}

	sortedKeys := slices.Sorted(maps.Keys(defs))

	for _, name := range sortedKeys {
		err := visit(name)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func EnsureApplication(app Application, restConfig *rest.Config) error {
	for _, service := range app.Services {
		err := HelmChartEnsure(
			service.Name,
			"service",
			service.Kind,
			service.Values,
			restConfig,
		)
		if err != nil {
			return err
		}
	}

	err := HelmChartEnsure(
		app.Name,
		"app",
		app.ChartName(),
		app.Values,
		restConfig,
	)
	return err
}
