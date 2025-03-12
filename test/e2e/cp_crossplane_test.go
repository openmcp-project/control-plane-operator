//go:build e2e

package e2e

import (
	"context"
	"testing"

	xpres "github.com/crossplane-contrib/xp-testing/pkg/resources"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

func TestCrossplane(t *testing.T) {
	cpName := "cp-e2e-crossplane"
	feature := features.New("CO-671 Install a Control Plane with Crossplane").
		Setup(SetUpControlPlaneResources("testdata/crs/cp-crossplane", cpName)).
		Assess(
			"Check Crossplane Installed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				WaitForCrossplaneResources(cfg, t)
				// waits for the Crossplane Component to be healthy
				WaitForComponentStatusToBeHealthy(t, cfg, cpName, "Crossplane")
				return ctx
			},
		).
		Assess(
			"Check Crossplane Updated", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				cpObserved := GetControlPlaneOrError(t, cfg, cpName)

				want := &v1beta1.ComponentsConfig{
					Crossplane: &v1beta1.CrossplaneConfig{
						Version: "1.17.1",
					},
				}
				updateCP := UpdateControlPlaneSpec(cpObserved, want)

				// Update Control Plane with new spec
				UpdateControlPlaneOrError(ctx, t, cfg, updateCP)

				// check if Deployment for Crossplane is created
				WaitForCrossplaneResources(cfg, t)

				// waits for the Crossplane Component to be healthy
				WaitForComponentStatusToBeHealthy(t, cfg, cpName, "Crossplane")
				return ctx
			},
		).
		Assess(
			"Check Crossplane Deleted", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				cpObserved := GetControlPlaneOrError(t, cfg, cpName)

				want := &v1beta1.ComponentsConfig{
					Crossplane: nil,
				}
				updateCP := UpdateControlPlaneSpec(cpObserved, want)

				// Update Control Plane with new spec
				UpdateControlPlaneOrError(ctx, t, cfg, updateCP)

				// check if Deployment for Crossplane is deleted
				checkCrossplaneDeploymentDeletedOrError(t, cfg)

				return ctx
			},
		).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Tear down control plane so that we do other tests
			cpResourceDummy := newControlPlaneResource(cfg, cpName)
			xpres.AwaitResourceDeletionOrFail(ctx, t, cfg, cpResourceDummy)
			return ctx
		}).Feature()

	testEnv.Test(t, feature)
}

// WaitForCrossplaneResources waits for the Crossplane Deployment to be ready
func WaitForCrossplaneResources(cfg *envconf.Config, t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crossplane",
			Namespace: "crossplane-system",
		},
	}
	// check if Deployment for Crossplane is created
	err := wait.For(conditions.New(cfg.Client().Resources()).ResourceMatch(dep, func(object k8s.Object) bool {
		d := object.(*appsv1.Deployment)
		return float64(d.Status.ReadyReplicas)/float64(*d.Spec.Replicas) >= 0.75
	}), wait.WithTimeout(timeoutDeploymentsAvailable))
	if err != nil {
		t.Error(err)
	}
}

// checkCrossplaneDeploymentDeletedOrError checks if the Crossplane Deployment is deleted
func checkCrossplaneDeploymentDeletedOrError(t *testing.T, cfg *envconf.Config) {
	// check if Deployment for Crossplane is deleted
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crossplane",
			Namespace: "crossplane-system",
		},
	}

	// wait for the deployment to be deleted
	err := wait.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(dep), wait.WithTimeout(timeoutDeploymentDeleted))

	if err != nil {
		t.Error(err)
	}
}
