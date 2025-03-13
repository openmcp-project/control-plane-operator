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
	certManagerDeploymentList = &appsv1.DeploymentList{
		Items: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cert-manager",
					Namespace: "cert-manager",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cert-manager-cainjector",
					Namespace: "cert-manager",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cert-manager-webhook",
					Namespace: "cert-manager",
				},
			},
		},
	}
)

func TestCertManager(t *testing.T) {
	cpName := "cp-e2e-cert-manager"
	feature := features.New("CO-671 Install a Control Plane with Cert Manager").
		Setup(SetUpControlPlaneResources("testdata/crs/cp-cert-manager", cpName)).
		Assess(
			"Check Cert Manager Installed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				WaitForCertManagerResources(cfg, t)
				// waits for the Cert Manager Component is healthy
				WaitForComponentStatusToBeHealthy(t, cfg, cpName, "CertManager")
				return ctx
			},
		).
		Assess(
			"Check Cert Manager Deleted", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				cpObserved := GetControlPlaneOrError(t, cfg, cpName)

				want := &v1beta1.ComponentsConfig{
					CertManager: nil,
				}

				updateCP := UpdateControlPlaneSpec(cpObserved, want)

				// Update Control Plane with new spec
				UpdateControlPlaneOrError(ctx, t, cfg, updateCP)

				checkCertManagerDeploymentDeletedOrError(t, cfg)

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
func WaitForCertManagerResources(cfg *envconf.Config, t *testing.T) {
	// check if all 3 Deployments for External Secrets Operator are created in namespace external-secrets
	err := wait.For(conditions.New(cfg.Client().Resources()).ResourceListMatchN(certManagerDeploymentList, 3, func(object k8s.Object) bool {
		d := object.(*appsv1.Deployment)

		// true if the image version is the same as the version we want AND the replicas are 1 AND the deployment is available
		return d.Status.Replicas == 1 &&
			IsStatusDeploymentConditionPresentAndEqual(d.Status.Conditions, "Available", corev1.ConditionTrue)

	}), wait.WithTimeout(timeoutDeploymentsAvailable))
	if err != nil {
		t.Error(err)
	}
}

// checkCertManagerDeploymentDeletedOrError checks if the Deployment for Cert Manager is deleted
func checkCertManagerDeploymentDeletedOrError(t *testing.T, cfg *envconf.Config) {
	// check if Deployment for Cert Manager is deleted
	err := wait.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(&certManagerDeploymentList.Items[0]), wait.WithTimeout(timeoutDeploymentDeleted))

	if err != nil {
		t.Error(err)
	}
}
