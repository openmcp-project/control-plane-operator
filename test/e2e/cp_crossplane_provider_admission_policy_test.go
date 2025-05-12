//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"

	xpres "github.com/crossplane-contrib/xp-testing/pkg/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestCrossplaneProviderAdmissionPolicy(t *testing.T) {
	cpWithProviderAdmissionPolicy := "cp-e2e-crossplane-provider-admission-policy"
	feature := features.New("CO-671 Check if Crossplane Provider is not allowed").
		Setup(SetUpControlPlaneResources("testdata/crs/cp-crossplane-provider-admission-policy", cpWithProviderAdmissionPolicy)).
		Assess("Check Crossplane Installed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			WaitForCrossplaneResources(cfg, t)

			// waits for the Crossplane Component to be healthy
			WaitForComponentStatusToBeHealthy(t, cfg, cpWithProviderAdmissionPolicy, "Crossplane")
			return ctx
		}).
		Assess("Check that Crossplane Provider installation is prevented by policy", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check that Crossplane Provider is not allowed to be installed and info message is shown to the user
			checkIfCrossplaneProviderFailsDueToPolicy(t, cfg, cpWithProviderAdmissionPolicy)
			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Tear down control plane so that we do other tests; this will also delete the Crossplane
			cpResourceDummy := newControlPlaneResource(cfg, cpWithProviderAdmissionPolicy)
			xpres.AwaitResourceDeletionOrFail(ctx, t, cfg, cpResourceDummy)
			return ctx
		}).Feature()

	testEnv.Test(t, feature)
}

// checkIfCrossplaneProviderFailsDueToPolicy checks if the Crossplane Provider is not allowed to be installed (due to Validation Admission Policy)
func checkIfCrossplaneProviderFailsDueToPolicy(t *testing.T, cfg *envconf.Config, cpName string) {
	cp := GetControlPlaneOrError(t, cfg, cpName)
	for _, c := range cp.Status.Conditions {
		if c.Type == "ProviderKubernetesReady" && c.Reason == juggler.StatusInstallFailed.Name && strings.Contains(c.Message, "ValidatingAdmissionPolicy") {
			return // Crossplane Provider is not allowed to be installed - test passed
		}
	}
	t.Errorf("Crossplane Provider is allowed to be installed but should not be! Status: %+v", meta.FindStatusCondition(cp.Status.Conditions, "ProviderKubernetesReady"))
}
