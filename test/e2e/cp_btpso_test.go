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

func TestBTPServiceOperator(t *testing.T) {
	cpName := "cp-e2e-btpso"
	feature := features.New("CO-671 Install a Control Plane with BTP Service Operator").
		Setup(SetUpControlPlaneResources("testdata/crs/cp-btpso", cpName)).
		Assess(
			"Check BTP Service Operator Installed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				WaitForBTPSOResources(cfg, t)

				// waits for the Cert Manager and BTP Service Operator Component to be healthy
				WaitForComponentStatusToBeHealthy(t, cfg, cpName, "CertManager", "BTPServiceOperator")
				return ctx
			},
		).
		Assess(
			"Check BTP Service Operator Updated", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				cpObserved := GetControlPlaneOrError(t, cfg, cpName)
				want := &v1beta1.ComponentsConfig{
					BTPServiceOperator: &v1beta1.BTPServiceOperatorConfig{
						Version: "0.6.8", // version increase for update
					},
					CertManager: &v1beta1.CertManagerConfig{
						Version: "1.16.1",
					},
				}
				updateCP := UpdateControlPlaneSpec(cpObserved, want)

				// Update Control Plane with new spec
				UpdateControlPlaneOrError(ctx, t, cfg, updateCP)

				// check if Deployment for BTP Service Operator is updated
				WaitForBTPSOResources(cfg, t)

				// waits for the Cert Manager and BTP Service Operator Component to be healthy
				WaitForComponentStatusToBeHealthy(t, cfg, cpName, "CertManager", "BTPServiceOperator")

				return ctx
			},
		).
		Assess(
			"Check BTP Service Operator Deleted", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				cpObserved := GetControlPlaneOrError(t, cfg, cpName)

				want := &v1beta1.ComponentsConfig{
					BTPServiceOperator: nil,
					CertManager: &v1beta1.CertManagerConfig{
						Version: "1.16.1",
					},
				}
				updateCP := UpdateControlPlaneSpec(cpObserved, want)

				// Update Control Plane with new spec
				UpdateControlPlaneOrError(ctx, t, cfg, updateCP)

				// check if Deployment for BTP Service Operator is deleted
				checkBTPSODeploymentDeletedOrError(t, cfg)

				return ctx
			},
		).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			cpObserved := GetControlPlaneOrError(t, cfg, cpName)

			updateCP := UpdateControlPlaneSpec(cpObserved, &v1beta1.ComponentsConfig{
				CertManager: nil,
			})

			// Update Control Plane with new spec
			UpdateControlPlaneOrError(ctx, t, cfg, updateCP)

			checkCertManagerDeploymentDeletedOrError(t, cfg)

			// Tear down control plane so that we do other tests
			cpResourceDummy := newControlPlaneResource(cfg, cpName)
			xpres.AwaitResourceDeletionOrFail(ctx, t, cfg, cpResourceDummy)
			return ctx
		}).Feature()

	testEnv.Test(t, feature)
}

// WaitForCrossplaneResources waits for the External Secrets Operator Deployment to be ready
func WaitForBTPSOResources(cfg *envconf.Config, t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sap-btp-operator-controller-manager",
			Namespace: "sap-btp-service-operator",
		},
	}
	// check if Deployment for BTP Service Operator is created
	err := wait.For(conditions.New(cfg.Client().Resources()).ResourceMatch(dep, func(object k8s.Object) bool {
		d := object.(*appsv1.Deployment)
		return float64(d.Status.ReadyReplicas)/float64(*d.Spec.Replicas) >= 0.75
	}), wait.WithTimeout(timeoutDeploymentsAvailable))
	if err != nil {
		t.Error(err)
	}
}

// checkBTPSODeploymentDeletedOrError checks if the BTP Service Operator Deployment is deleted
func checkBTPSODeploymentDeletedOrError(t *testing.T, cfg *envconf.Config) {
	// check if Deployment for BTP Service Operator is deleted
	btpsoDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sap-btp-operator-controller-manager",
			Namespace: "sap-btp-service-operator",
		},
	}

	// wait for the deployments to be deleted
	err := wait.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(btpsoDeployment), wait.WithTimeout(timeoutDeploymentDeleted))

	if err != nil {
		t.Error(err)
	}
}
