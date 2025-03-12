//go:build e2e

package e2e

import (
	"context"
	"testing"

	xpres "github.com/crossplane-contrib/xp-testing/pkg/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

var (
	esoDeploymentList = &appsv1.DeploymentList{
		Items: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets",
					Namespace: "external-secrets",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-cert-controller",
					Namespace: "external-secrets",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-webhook",
					Namespace: "external-secrets",
				},
			},
		},
	}
)

func TestExternalSecretsOperator(t *testing.T) {
	cpName := "cp-e2e-external-secrets-operator"
	feature := features.New("CO-671 Install a Control Plane with External Secrets Operator").
		Setup(SetUpControlPlaneResources("testdata/crs/cp-eso", cpName)).
		Assess(
			"Check External Secrets Operator Installed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				WaitForExternalSecretsOperatorResources(cfg, t)

				// waits for the ESO Component to be healthy
				WaitForComponentStatusToBeHealthy(t, cfg, cpName, "ExternalSecretsOperator")
				return ctx
			},
		).
		Assess(
			"Check External Secrets Operator Updated", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				cpObserved := GetControlPlaneOrError(t, cfg, cpName)

				want := &v1beta1.ComponentsConfig{
					ExternalSecretsOperator: &v1beta1.ExternalSecretsOperatorConfig{
						Version: "0.10.4",
					},
				}

				updateCP := UpdateControlPlaneSpec(cpObserved, want)

				// Update Control Plane with new spec
				UpdateControlPlaneOrError(ctx, t, cfg, updateCP)

				// check if Deployment for External Secrets Operator is created
				WaitForExternalSecretsOperatorResources(cfg, t)

				// waits for the ESO Component to be healthy
				WaitForComponentStatusToBeHealthy(t, cfg, cpName, "ExternalSecretsOperator")
				return ctx
			},
		).
		Assess(
			"Check External Secrets Operator Deleted", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				cpObserved := GetControlPlaneOrError(t, cfg, cpName)

				want := &v1beta1.ComponentsConfig{
					ExternalSecretsOperator: nil,
				}

				updateCP := UpdateControlPlaneSpec(cpObserved, want)

				// Update Control Plane with new spec
				UpdateControlPlaneOrError(ctx, t, cfg, updateCP)

				// check if Deployment for External Secrets Operator is deleted
				checkExternalSecretsOperatorDeploymentDeletedOrError(t, cfg)

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

// WaitForCrossplaneResources waits for the External Secrets Operator Deployment to be ready
func WaitForExternalSecretsOperatorResources(cfg *envconf.Config, t *testing.T) {
	// check if all 3 Deployments for External Secrets Operator are created in namespace external-secrets
	err := wait.For(conditions.New(cfg.Client().Resources()).ResourceListMatchN(esoDeploymentList, 3, func(object k8s.Object) bool {
		d := object.(*appsv1.Deployment)
		return IsStatusDeploymentConditionPresentAndEqual(d.Status.Conditions, "Available", corev1.ConditionTrue) &&
			d.Status.Replicas == 1 // TODO: fix this, this should be availableReplicas >= 1
	}), wait.WithTimeout(timeoutDeploymentsAvailable))
	if err != nil {
		t.Error(err)
	}
}

// checkExternalSecretsOperatorDeploymentDeletedOrError checks if the Deployment for External Secrets Operator is deleted
func checkExternalSecretsOperatorDeploymentDeletedOrError(t *testing.T, cfg *envconf.Config) {
	// check if Deployment for Crossplane is deleted
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-secrets",
			Namespace: "external-secrets",
		},
	}

	// wait for the deployment to be deleted
	err := wait.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(dep), wait.WithTimeout(timeoutDeploymentDeleted))

	if err != nil {
		t.Error(err)
	}
}
