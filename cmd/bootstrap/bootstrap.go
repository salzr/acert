package bootstrap

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/salzr/acert/helm"
)

type options struct {
	certManagerReleaseName string
	certManagerRepository  string
	certManagerVersion     string
	certManagerNamespace   string
}

func defaultOptions() options {
	return options{
		certManagerReleaseName: "cert-manager",
		certManagerRepository:  "quay.io/jetstack/charts/cert-manager",
		certManagerVersion:     "v1.19.2",
		certManagerNamespace:   "cert-manager",
	}
}

func Command() *cobra.Command {
	opts := defaultOptions()

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstraps the cluster for cert-manager",
		Run: func(cmd *cobra.Command, args []string) {
			logger := cmd.Context().Value("logger").(*zap.Logger)
			logger = logger.With(zap.String("service", "bootstrap"))

			ch := helm.Chart{
				Name:       opts.certManagerReleaseName,
				Repository: opts.certManagerRepository,
				Version:    opts.certManagerVersion,
				Namespace:  opts.certManagerNamespace,
			}
			rel, err := helm.InstallOrUpdate(cmd.Context(), ch,
				map[string]interface{}{
					"crds": map[string]interface{}{
						"enabled": true,
					},
				})
			if err != nil {
				logger.Fatal("Failed to install chart", zap.Error(err))
			}

			logger.Info("Chart installed successfully", zap.String("release", rel.Name))
		},
	}
	cmd.PersistentFlags().StringVar(&opts.certManagerReleaseName, "cert-manager-release-name",
		opts.certManagerReleaseName, "cert-manager chart release name")
	cmd.PersistentFlags().StringVar(&opts.certManagerRepository, "cert-manager-repository",
		opts.certManagerRepository, "cert-manager chart repository")
	cmd.PersistentFlags().StringVar(&opts.certManagerVersion, "cert-manager-version",
		opts.certManagerVersion, "cert-manager chart version")
	cmd.PersistentFlags().StringVar(&opts.certManagerNamespace, "cert-manager-namespace",
		opts.certManagerNamespace, "cert-manager chart namespace")

	return cmd
}
