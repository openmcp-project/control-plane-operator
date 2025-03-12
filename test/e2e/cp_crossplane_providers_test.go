//go:build e2e

package e2e

import (
	"context"
	"testing"

	xpres "github.com/crossplane-contrib/xp-testing/pkg/resources"
	xcommonv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	crossplanev1 "github.com/crossplane/crossplane/apis/pkg/v1"
	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/crossplane"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	res "sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	providerVersionInManifest = "0.14.1"
	providerVersionNew        = "0.15.0"
	providerName              = "kubernetes"
)

func TestCrossplaneProviders(t *testing.T) {
	cpName := "cp-e2e-crossplane-provider"
	feature := features.New("CO-671 Install a Control Plane with Crossplane and a Crossplane Provider").
		Setup(SetUpControlPlaneResources("testdata/crs/cp-crossplane-provider", cpName)).
		Assess(
			"Check Crossplane Installed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				WaitForCrossplaneResources(cfg, t)

				// waits for the Crossplane Component to be healthy
				WaitForComponentStatusToBeHealthy(t, cfg, cpName, "Crossplane")
				return ctx
			},
		).
		Assess(
			"Check Crossplane Provider Installed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				WaitForCrossplaneProviderResources(cfg, t, providerKubernetesComponentConfig(providerVersionInManifest))
				// waits for the Crossplane and ProviderKubernetes Component to be healthy
				WaitForComponentStatusToBeHealthy(t, cfg, cpName, "Crossplane", "ProviderKubernetes")
				return ctx
			},
		).
		Assess(
			"Check Crossplane Provider Updated", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				cpObserved := GetControlPlaneOrError(t, cfg, cpName)

				want := providerKubernetesComponentConfig(providerVersionNew)

				updateCP := UpdateControlPlaneSpec(cpObserved, want)

				// Update Control Plane with new spec
				UpdateControlPlaneOrError(ctx, t, cfg, updateCP)

				// check if Deployment for Crossplane is created
				WaitForCrossplaneProviderResources(cfg, t, want)

				// waits for the Crossplane and ProviderKubernetes Component to be healthy
				WaitForComponentStatusToBeHealthy(t, cfg, cpName, "Crossplane", "ProviderKubernetes")

				return ctx
			},
		).
		Assess(
			"Check Crossplane Provider Deleted", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				cpObserved := GetControlPlaneOrError(t, cfg, cpName)

				want := &v1beta1.ComponentsConfig{
					Crossplane: &v1beta1.CrossplaneConfig{
						Version: "1.17.1",
					},
				}
				updateCP := UpdateControlPlaneSpec(cpObserved, want)

				// Update Control Plane with new spec
				UpdateControlPlaneOrError(ctx, t, cfg, updateCP)

				// check if Deployment for Crossplane Provider is deleted
				checkCrossplaneProviderDeploymentDeletedOrError(t, cfg)
				return ctx
			},
		).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Tear down control plane so that we do other tests; this will also delete the Crossplane
			cpResourceDummy := newControlPlaneResource(cfg, cpName)
			xpres.AwaitResourceDeletionOrFail(ctx, t, cfg, cpResourceDummy)
			return ctx
		}).Feature()

	testEnv.Test(t, feature)
}

func providerKubernetesComponentConfig(providerVersion string) *v1beta1.ComponentsConfig {
	return &v1beta1.ComponentsConfig{
		Crossplane: &v1beta1.CrossplaneConfig{
			Version: "1.17.1",
			Providers: []*v1beta1.CrossplaneProviderConfig{
				{
					Name:    providerName,
					Version: providerVersion,
				},
			},
		},
	}
}

// WaitForCrossplaneProviderResources waits for the Crossplane Provider to be ready
func WaitForCrossplaneProviderResources(cfg *envconf.Config, t *testing.T, want *v1beta1.ComponentsConfig) {
	client := getResourcesWithCrossplaneSchemeOrError(cfg, t)

	provider := &crossplanev1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name: crossplane.AddProviderPrefix(want.Crossplane.Providers[0].Name),
		},
	}

	err := wait.For(conditions.New(client).ResourceMatch(provider, func(object k8s.Object) bool {
		prov := object.(*crossplanev1.Provider)
		// true if correct package with version and if Provider is healthy
		return IsStatusConditionPresentAndEqual(prov.Status.Conditions, "Healthy", corev1.ConditionTrue)
	}), wait.WithTimeout(timeoutDeploymentsAvailable))
	if err != nil {
		t.Error(err)
	}
}

// getResourcesWithCrossplaneSchemeOrError returns a res.Resources with registered Crossplane scheme
func getResourcesWithCrossplaneSchemeOrError(cfg *envconf.Config, t *testing.T) *res.Resources {
	client, err := cfg.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	clientres := client.Resources()
	_ = crossplanev1.AddToScheme(clientres.GetScheme())

	return clientres
}

// IsStatusConditionPresentAndEqual returns true when conditionType is present and equal to status.
func IsStatusConditionPresentAndEqual(conditions []xcommonv1.Condition, conditionType string, status corev1.ConditionStatus) bool {
	for _, condition := range conditions {
		if string(condition.Type) == conditionType {
			return condition.Status == status
		}
	}
	return false
}

// checkCrossplaneProviderDeploymentDeletedOrError checks if the Crossplane Provider Deployment is deleted
func checkCrossplaneProviderDeploymentDeletedOrError(t *testing.T, cfg *envconf.Config) {
	client := getResourcesWithCrossplaneSchemeOrError(cfg, t)

	// check if Crossplane Provider is deleted
	provider := &crossplanev1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name: crossplane.AddProviderPrefix(providerName),
		},
	}

	// wait for the Crossplane Provider to be deleted
	err := wait.For(conditions.New(client).ResourceDeleted(provider), wait.WithTimeout(timeoutDeploymentDeleted))

	if err != nil {
		t.Error(err)
	}
}
