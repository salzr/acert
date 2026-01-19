package k8s

import (
	"context"
	"fmt"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type k8sObject interface {
	client.Object
}

// NewClient creates a new k8s client.
func NewClient(ctx context.Context, config *rest.Config, builder ...runtime.SchemeBuilder) (client.Client, error) {
	logger := ctx.Value("logger").(*zap.Logger)
	log.SetLogger(zapr.NewLogger(logger))

	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		return nil, fmt.Errorf("failed to add scheme: %w", err)
	}
	for _, b := range builder {
		if err := b.AddToScheme(s); err != nil {
			return nil, fmt.Errorf("failed to add scheme: %w", err)
		}
	}
	k8sClient, err := client.New(config, client.Options{Scheme: s})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}
	return k8sClient, err
}

// FindOrCreate finds an object by name and namespace or creates it if it doesn't exist.
func FindOrCreate[T k8sObject](ctx context.Context, k8sClient client.Client, obj T) error {
	nn := client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}
	if err := k8sClient.Get(ctx, nn, obj); err != nil {
		if errors.IsNotFound(err) {
			return k8sClient.Create(ctx, obj)
		}
		return err
	}
	return nil
}
