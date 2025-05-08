package secretresolver

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/pkg/constants"
)

type ResolveFunc func(urlType UrlSecretType) (*corev1.LocalObjectReference, error)

// SecretResolver defines methods on resolving fluxSecrets
type SecretResolver interface {
	// Start starts an initial run in scanning the Secrets
	Start(ctx context.Context) error

	// Resolve returns the LocalObjectReference to a Secret
	Resolve(urlType UrlSecretType) (*corev1.LocalObjectReference, error)
}

// NewFluxSecretResolver creates a new FluxSecretResolver
func NewFluxSecretResolver(c client.Client) SecretResolver {
	return &FluxSecretResolver{
		client:  c,
		secrets: map[UrlSecretType]string{},
	}
}

// FluxSecretResolver is a struct that implements the SecretResolver interface.
// It resolves the fluxSecrets for the Helm and Docker repositories.
type FluxSecretResolver struct {
	client  client.Client
	secrets map[UrlSecretType]string
}

// UrlSecretType is a struct that holds the URL and the SecretType
type UrlSecretType struct {
	URL        string
	SecretType corev1.SecretType
}

var _ SecretResolver = &FluxSecretResolver{}

// Start implements SecretResolver
func (f *FluxSecretResolver) Start(ctx context.Context) error {
	matchingLabels := client.MatchingLabels{
		constants.LabelCopyToCPNamespace: "true",
	}
	secretList := &corev1.SecretList{}

	// get Secrets with matching labels
	err := f.client.List(ctx, secretList, matchingLabels)
	if err != nil {
		return err
	}

	for _, secret := range secretList.Items {
		joinedURLs, hasURL := secret.Annotations[constants.AnnotationCredentialsForUrl]
		if !hasURL {
			continue
		}

		urls := strings.Split(joinedURLs, ",")
		for _, url := range urls {
			urlType := UrlSecretType{
				URL:        url,
				SecretType: secret.Type,
			}

			f.secrets[urlType] = secret.Name
		}
	}
	return nil
}

// Resolve implements SecretResolver
func (f *FluxSecretResolver) Resolve(urlType UrlSecretType) (*corev1.LocalObjectReference, error) {
	secretName, hasSecret := f.secrets[urlType]
	if hasSecret {
		return &corev1.LocalObjectReference{Name: secretName}, nil
	}
	return nil, nil
}
