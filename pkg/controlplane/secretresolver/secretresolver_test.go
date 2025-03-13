package secretresolver

import (
	"context"
	"testing"

	"github.com/openmcp-project/control-plane-operator/pkg/constants"
	"github.com/stretchr/testify/assert"
	assert2 "gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	helmSecret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "helm-secret",
			Namespace: "default",
			Annotations: map[string]string{
				constants.AnnotationCredentialsForUrl: "https://test.com",
			},
			Labels: map[string]string{
				constants.LabelCopyToCPNamespace: "true",
			},
		},
		Type: corev1.SecretTypeBasicAuth,
	}
	dockerSecret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docker-secret",
			Namespace: "default",
			Annotations: map[string]string{
				constants.AnnotationCredentialsForUrl: "https://test.com",
			},
			Labels: map[string]string{
				constants.LabelCopyToCPNamespace: "true",
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}
)

//nolint:lll
func TestFluxSecretResolver_Start(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(ctx context.Context, c client.Client) error
		validateStart func(ctx context.Context, t *testing.T, c client.Client, r SecretResolver, err error) error
	}{
		{
			name: "Start - two secrets found and secrets map is filled - no errors",
			setup: func(ctx context.Context, c client.Client) error {
				secret1 := helmSecret.DeepCopy()
				if err := c.Create(ctx, secret1); err != nil {
					return err
				}
				secret2 := dockerSecret.DeepCopy()
				if err := c.Create(ctx, secret2); err != nil {
					return err
				}
				return nil
			},
			validateStart: func(ctx context.Context, t *testing.T, c client.Client, r SecretResolver, err error) error {
				assert.Len(t, r.(*FluxSecretResolver).secrets, 2)
				assert.Equal(t, "helm-secret", r.(*FluxSecretResolver).secrets[UrlSecretType{URL: "https://test.com", SecretType: corev1.SecretTypeBasicAuth}])
				assert.Equal(t, "docker-secret", r.(*FluxSecretResolver).secrets[UrlSecretType{URL: "https://test.com", SecretType: corev1.SecretTypeDockerConfigJson}])
				assert.NoError(t, err)
				return nil
			},
		},
		{
			name: "Start - no secrets found - no errors",
			setup: func(ctx context.Context, c client.Client) error {
				return nil
			},
			validateStart: func(ctx context.Context, t *testing.T, c client.Client, r SecretResolver, err error) error {
				assert.Len(t, r.(*FluxSecretResolver).secrets, 0)
				assert.NoError(t, err)
				return nil
			},
		},
		{
			name: "Start - one secret found - no errors",
			setup: func(ctx context.Context, c client.Client) error {
				secret1 := helmSecret.DeepCopy()
				if err := c.Create(ctx, secret1); err != nil {
					return err
				}
				return nil
			},
			validateStart: func(ctx context.Context, t *testing.T, c client.Client, r SecretResolver, err error) error {
				assert.Len(t, r.(*FluxSecretResolver).secrets, 1)
				assert.Equal(t, "helm-secret", r.(*FluxSecretResolver).secrets[UrlSecretType{URL: "https://test.com", SecretType: corev1.SecretTypeBasicAuth}])
				assert.NoError(t, err)
				return nil
			},
		},
		{
			name: "Start - one secret found with no url annotation - error",
			setup: func(ctx context.Context, c client.Client) error {
				secret := helmSecret.DeepCopy()
				secret.Annotations = map[string]string{}
				if err := c.Create(ctx, secret); err != nil {
					return err
				}
				return nil
			},
			validateStart: func(ctx context.Context, t *testing.T, c client.Client, r SecretResolver, err error) error {
				assert.Len(t, r.(*FluxSecretResolver).secrets, 0)
				assert.NoError(t, err)
				return nil
			},
		},
	}
	for _, tC := range tests {
		t.Run(tC.name, func(t *testing.T) {
			c := fake.NewClientBuilder().Build()
			ctx := context.Background()
			if err := tC.setup(ctx, c); err != nil {
				t.Fatal(err)
			}

			r := NewFluxSecretResolver(c)

			testErr := r.Start(ctx)

			if err := tC.validateStart(ctx, t, c, r, testErr); err != nil {
				t.Fatal(err)
			}
		})
	}
}

//nolint:lll
func TestFluxSecretResolver_Resolve(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(ctx context.Context, c client.Client) error
		input           UrlSecretType
		validateResolve func(ctx context.Context, t *testing.T, c client.Client, r SecretResolver, actual *corev1.LocalObjectReference, err error) error
	}{
		{
			name: "Resolve - secret found - no errors",
			setup: func(ctx context.Context, c client.Client) error {
				secret1 := helmSecret.DeepCopy()
				if err := c.Create(ctx, secret1); err != nil {
					return err
				}
				secret2 := dockerSecret.DeepCopy()
				if err := c.Create(ctx, secret2); err != nil {
					return err
				}
				return nil
			},
			input: UrlSecretType{URL: "https://test.com", SecretType: corev1.SecretTypeBasicAuth},
			validateResolve: func(ctx context.Context, t *testing.T, c client.Client, r SecretResolver, actual *corev1.LocalObjectReference, err error) error {
				assert2.DeepEqual(t, &corev1.LocalObjectReference{Name: "helm-secret"}, actual)
				assert.NoError(t, err)
				return nil
			},
		},
		{
			name: "Resolve - secret not found - no errors",
			setup: func(ctx context.Context, c client.Client) error {
				return nil
			},
			input: UrlSecretType{URL: "https://test.com", SecretType: corev1.SecretTypeBasicAuth},
			validateResolve: func(ctx context.Context, t *testing.T, c client.Client, r SecretResolver, actual *corev1.LocalObjectReference, err error) error {
				assert.Nil(t, actual)
				assert.NoError(t, err)
				return nil
			},
		},
	}
	for _, tC := range tests {
		t.Run(tC.name, func(t *testing.T) {
			c := fake.NewClientBuilder().Build()
			ctx := context.Background()
			if err := tC.setup(ctx, c); err != nil {
				t.Fatal(err)
			}

			r := NewFluxSecretResolver(c)

			if err := r.Start(ctx); err != nil {
				t.Fatal(err)
			}

			// testing the Resolve function
			actual, err := r.Resolve(tC.input)

			if err := tC.validateResolve(ctx, t, c, r, actual, err); err != nil {
				t.Fatal(err)
			}
		})
	}
}
