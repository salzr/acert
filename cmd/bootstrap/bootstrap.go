package bootstrap

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/salzr/acert/helm"
	"github.com/salzr/acert/k8s"
)

type options struct {
	acertNamespace          string
	acertCertificateInstall bool

	certManagerInstall     bool
	certManagerReleaseName string
	certManagerRepository  string
	certManagerVersion     string
	certManagerNamespace   string

	configFlags *genericclioptions.ConfigFlags
}

func defaultOptions() options {
	configFlags := genericclioptions.NewConfigFlags(true)

	return options{
		certManagerInstall:     true,
		certManagerReleaseName: "cert-manager",
		certManagerRepository:  "quay.io/jetstack/charts/cert-manager",
		certManagerVersion:     "v1.19.2",
		certManagerNamespace:   "cert-manager",

		acertNamespace:          "acert-system",
		acertCertificateInstall: true,

		configFlags: configFlags,
	}
}

func Command() *cobra.Command {
	opts := defaultOptions()

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstraps acert and dependencies",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.WithValue(cmd.Context(), "configFlags", opts.configFlags)

			logger := ctx.Value("logger").(*zap.Logger)
			logger = logger.With(zap.String("service", "bootstrap"))

			// TODO: Maybe move this to a helper function or maybe initialize it in the options field
			cfg, err := opts.configFlags.ToRESTConfig()
			if err != nil {
				logger.Fatal("Failed to create rest config", zap.Error(err))
			}

			k8sClient, err := k8s.NewClient(ctx, cfg, certmanagerv1.SchemeBuilder)
			if err != nil {
				logger.Fatal("Failed to create k8s client", zap.Error(err))
			}

			if opts.certManagerInstall {
				ch := helm.Chart{
					Name:       opts.certManagerReleaseName,
					Repository: opts.certManagerRepository,
					Version:    opts.certManagerVersion,
					Namespace:  opts.certManagerNamespace,
				}
				rel, err := helm.InstallOrUpdate(ctx, ch,
					map[string]any{
						"crds": map[string]any{
							"enabled": true,
						},
					})
				if err != nil {
					logger.Fatal("Failed to install chart", zap.Error(err))
				}
				logger.Info("Chart installed successfully", zap.String("release", rel.Name))
			}

			if opts.acertCertificateInstall {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: opts.acertNamespace,
					},
				}
				if err := k8s.FindOrCreate[*corev1.Namespace](ctx, k8sClient, ns); err != nil {
					logger.Fatal("Failed to find or create acert namespace", zap.Error(err))
				}

				issuer := &certmanagerv1.ClusterIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "selfsigned-ca-issuer",
					},
					Spec: certmanagerv1.IssuerSpec{
						IssuerConfig: certmanagerv1.IssuerConfig{
							SelfSigned: &certmanagerv1.SelfSignedIssuer{},
						},
					},
				}
				if err := k8s.FindOrCreate[*certmanagerv1.ClusterIssuer](ctx, k8sClient, issuer); err != nil {
					logger.Fatal("Failed to find or create self-signed issuer", zap.Error(err))
				}

				serverCA := &certmanagerv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "acert-server-ca",
						Namespace: ns.Name,
					},
				}
				if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: serverCA.Namespace, Name: serverCA.Name}, serverCA); err != nil {
					if errors.IsNotFound(err) {
						serverCA.Spec = certmanagerv1.CertificateSpec{
							IsCA:       true,
							CommonName: "acert-server-ca",
							SecretName: "acert-server-ca",
							Usages: []certmanagerv1.KeyUsage{
								certmanagerv1.UsageCertSign,
							},
							PrivateKey: &certmanagerv1.CertificatePrivateKey{
								Algorithm: certmanagerv1.RSAKeyAlgorithm,
								Size:      4096,
							},
							IssuerRef: certmanagermetav1.IssuerReference{
								Kind: "ClusterIssuer",
								Name: issuer.Name,
							},
						}

						serverCA.Spec.Duration = func() *metav1.Duration {
							prompt := promptui.Prompt{
								Label:   "Enter server CA duration",
								Default: "87600h",
							}
							v, _ := prompt.Run()
							duration, err := time.ParseDuration(v)
							if err != nil {
								logger.Fatal("Failed to parse duration", zap.Error(err))
							}
							return &metav1.Duration{Duration: duration}
						}()

						serverCA.Spec.Subject = fillOutPrompt(&certmanagerv1.X509Subject{}).(*certmanagerv1.X509Subject)

						logger.Info("Creating server CA certificate", zap.Any("serverCA", serverCA))
						if err := k8sClient.Create(ctx, serverCA); err != nil {
							logger.Fatal("Failed to create server CA certificate", zap.Error(err))
						}
					} else {
						logger.Fatal("Failed to get server CA certificate", zap.Error(err))
					}
				}

				serverIssuer := &certmanagerv1.Issuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "acert-cluster-certificate-issuer",
						Namespace: opts.acertNamespace,
					},
					Spec: certmanagerv1.IssuerSpec{
						IssuerConfig: certmanagerv1.IssuerConfig{
							CA: &certmanagerv1.CAIssuer{
								SecretName: serverCA.Spec.SecretName,
							},
						},
					},
				}
				if err := k8s.FindOrCreate[*certmanagerv1.Issuer](ctx, k8sClient, serverIssuer); err != nil {
					logger.Fatal("Failed to find or create issuer", zap.Error(err))
				}

				if err := k8s.FindOrCreate[*certmanagerv1.Certificate](ctx, k8sClient, &certmanagerv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "acert-grpc-server-cert",
						Namespace: opts.acertNamespace,
					},
					Spec: certmanagerv1.CertificateSpec{
						IsCA: true,
						Duration: func() *metav1.Duration {
							d := 87600 * time.Hour
							return &metav1.Duration{Duration: d}
						}(),
						CommonName: "acert-agent-ca-cert",
						SecretName: "acert-agent-ca-cert",
						Subject:    fillOutPrompt(&certmanagerv1.X509Subject{}).(*certmanagerv1.X509Subject),
						DNSNames:   []string{"server.acert.salzr.localhost"},
						PrivateKey: &certmanagerv1.CertificatePrivateKey{
							Algorithm: certmanagerv1.RSAKeyAlgorithm,
							Size:      4096,
						},
						SignatureAlgorithm: certmanagerv1.SHA256WithRSA,
						Usages: []certmanagerv1.KeyUsage{
							certmanagerv1.UsageDigitalSignature,
							certmanagerv1.UsageKeyEncipherment,
							certmanagerv1.UsageKeyAgreement,
						},
						IssuerRef: certmanagermetav1.IssuerReference{
							Name: serverIssuer.Name,
							Kind: "Issuer",
						},
					},
				}); err != nil {
					logger.Fatal("Failed to find or create certificate", zap.Error(err))
				}

				if err := k8s.FindOrCreate[*certmanagerv1.Certificate](ctx, k8sClient, &certmanagerv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "acert-agent-ca",
						Namespace: opts.acertNamespace,
					},
					Spec: certmanagerv1.CertificateSpec{
						IsCA:       true,
						CommonName: "acert-agent-ca",
						SecretName: "acert-agent-ca",
						Subject:    fillOutPrompt(&certmanagerv1.X509Subject{}).(*certmanagerv1.X509Subject),
						Usages: []certmanagerv1.KeyUsage{
							certmanagerv1.UsageCertSign,
						},
						PrivateKey: &certmanagerv1.CertificatePrivateKey{
							Algorithm: certmanagerv1.RSAKeyAlgorithm,
							Size:      4096,
						},
						IssuerRef: certmanagermetav1.IssuerReference{
							Kind: "ClusterIssuer",
							Name: issuer.Name,
						},
					},
				}); err != nil {
					logger.Fatal("Failed to find or create certificate", zap.Error(err))
				}
			}
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

	opts.configFlags.AddFlags(cmd.PersistentFlags())

	return cmd
}

// TODO: Expand this to be a method with configurable options
func fillOutPrompt(obj any) any {
	ptr := reflect.ValueOf(obj)
	val := ptr.Elem()
	for i := 0; i < val.NumField(); i++ {
		fName := val.Type().Field(i).Name
		fVal := val.Field(i)
		switch fVal.Kind() {
		case reflect.Slice:
			answers := make([]string, 0)
			for {
				label := "Enter " + fName + "(optional, multiple values accepted, exit entry mode by entering empty)"
				if len(answers) > 0 {
					label = "Enter " + fName + fmt.Sprintf("(%s)", strings.Join(answers, ", "))
				}
				prompt := promptui.Prompt{
					Label: label,
				}
				v, _ := prompt.Run()
				if v == "" {
					break
				}
				answers = append(answers, v)
			}
			if len(answers) > 0 {
				fVal.Set(reflect.ValueOf(answers))
			}
		default:
			prompt := promptui.Prompt{
				Label: "Enter " + fName,
			}
			v, _ := prompt.Run()
			if len(v) > 0 {
				fVal.SetString(v)
			}
		}
	}
	return ptr.Interface()
}
