package helm

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type Chart struct {
	Name       string
	Repository string
	Version    string
	Namespace  string
}

func (c Chart) Download() (*chart.Chart, error) {
	regClient, err := registry.NewClient(registry.ClientOptDebug(true),
		registry.ClientOptWriter(os.Stderr))
	if err != nil {
		return nil, fmt.Errorf("failed to create registry client: %w", err)
	}

	ociGetter, err := getter.NewOCIGetter(getter.WithRegistryClient(regClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI getter: %w", err)
	}

	buf, err := ociGetter.Get(fmt.Sprintf("oci://%s:%s", c.Repository, c.Version))
	if err != nil {
		return nil, fmt.Errorf("failed to get chart: %w", err)
	}

	ch, err := loader.LoadArchive(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	return ch, nil
}

// InstallOrUpdate installs or updates a given chart
func InstallOrUpdate(ctx context.Context, chart Chart, values map[string]interface{}) (*release.Release, error) {
	logger := ctx.Value("logger").(*zap.Logger)
	logger = logger.With(zap.String("service", "helm"))

	configFlags := genericclioptions.NewConfigFlags(true)
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(configFlags, "cert-manager", "secret", logger.Sugar().Debugf); err != nil {
		return nil, err
	}

	listClient := action.NewList(actionConfig)
	listClient.Filter = chart.Name
	listClient.Deployed = true
	rels, err := listClient.Run()
	if err != nil {
		return nil, err
	}

	ch, err := chart.Download()
	if err != nil {
		return nil, err
	}

	if rels != nil && len(rels) > 0 {
		upgradeClient := action.NewUpgrade(actionConfig)
		rel, err := upgradeClient.Run(chart.Name, ch, values)
		if err != nil {
			return nil, err
		}
		return rel, nil
	}

	installClient := action.NewInstall(actionConfig)
	installClient.ReleaseName = chart.Name
	installClient.Namespace = chart.Namespace
	installClient.CreateNamespace = true
	rel, err := installClient.Run(ch, values)
	if err != nil {
		return nil, err
	}
	return rel, nil
}
