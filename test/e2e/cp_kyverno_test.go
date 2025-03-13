//go:build e2e

package e2e

import (
	"context"
	"testing"

	xpres "github.com/crossplane-contrib/xp-testing/pkg/resources"
	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	// from kyverno chart 3.1.4+ (hmm pi)
	kyvernoDeploymentList = &appsv1.DeploymentList{
		Items: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kyverno-admission-controller",
					Namespace: "kyverno-system",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kyverno-background-controller",
					Namespace: "kyverno-system",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kyverno-reports-controller",
					Namespace: "kyverno-system",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kyverno-cleanup-controller",
					Namespace: "kyverno-system",
				},
			},
		},
	}
)

func TestKyverno(t *testing.T) {
	cpName := "cp-e2e-kyverno"
	feature := features.New("CO-671 Install a Control Plane with Kyverno").
		Setup(SetUpControlPlaneResources("testdata/crs/cp-kyverno", cpName)).
		Assess(
			"Check Kyverno Installed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				WaitForKyvernoResources(cfg, t)
				WaitForComponentStatusToBeHealthy(t, cfg, cpName, "Kyverno")
				return ctx
			},
		).
		// Assess(
		// 	"Check Kyverno Updated", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		// 		cpObserved := GetControlPlaneOrError(t, cfg, cpName)

		// 		want := &v1beta1.ComponentsConfig{
		// 			Kyverno: &v1beta1.KyvernoConfig{
		// 				Version: "latest",
		// 			},
		// 		}
		// 		updateCP := UpdateControlPlaneSpec(cpObserved, want)
		// 		UpdateControlPlaneOrError(ctx, t, cfg, updateCP)
		// 		WaitForKyvernoResources(cfg, t)
		// 		WaitForComponentStatusToBeHealthy(t, cfg, cpName, "Kyverno")
		// 		return ctx
		// 	},
		// ).
		Assess(
			"Check Kyverno Deleted", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				cpObserved := GetControlPlaneOrError(t, cfg, cpName)

				want := &v1beta1.ComponentsConfig{
					Kyverno: nil,
				}
				updateCP := UpdateControlPlaneSpec(cpObserved, want)
				UpdateControlPlaneOrError(ctx, t, cfg, updateCP)
				checkKyvernoDeletedOrError(t, cfg)
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

// WaitForKyvernoResources waits for the Kyverno Deployment to be ready
func WaitForKyvernoResources(cfg *envconf.Config, t *testing.T) {
	// check if all deployments from kyverno to be available in kyverno-system namespace
	err := wait.For(conditions.New(cfg.Client().Resources()).ResourceListMatchN(kyvernoDeploymentList, 3, func(object k8s.Object) bool {
		d := object.(*appsv1.Deployment)

		return IsStatusDeploymentConditionPresentAndEqual(d.Status.Conditions, "Available", corev1.ConditionTrue) &&
			d.Status.AvailableReplicas >= 1
	}), wait.WithTimeout(timeoutDeploymentsAvailable))
	if err != nil {
		t.Error(err)
	}
}

// checkKyvernoDeletedOrError checks if one of the Kyverno Deployment is deleted
func checkKyvernoDeletedOrError(t *testing.T, cfg *envconf.Config) {
	// check if kyverno admin controller is deleted, if it is, then other kyverno deployments should also be gone (as there are more than 1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kyverno-admission-controller",
			Namespace: "kyverno-system",
		},
	}

	// wait for the kyverno resources to be deleted
	err := wait.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(deployment), wait.WithTimeout(timeoutDeploymentDeleted))

	if err != nil {
		t.Error(err)
	}
}
