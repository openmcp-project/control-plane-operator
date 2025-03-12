//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"

	xpres "github.com/crossplane-contrib/xp-testing/pkg/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestCrossplaneProviderNotAllowed(t *testing.T) {
	cpWithProviderNotAllowed := "cp-e2e-crossplane-provider-not-allowed"
	feature := features.New("CO-671 Check if Crossplane Provider is not allowed").
		Setup(SetUpControlPlaneResources("testdata/crs/cp-crossplane-provider-not-allowed", cpWithProviderNotAllowed)).
		Assess("Check Crossplane Installed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			WaitForCrossplaneResources(cfg, t)

			// waits for the Crossplane Component to be healthy
			WaitForComponentStatusToBeHealthy(t, cfg, cpWithProviderNotAllowed, "Crossplane")
			return ctx
		}).
		Assess("Check that Crossplane Provider is not allowed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check that Crossplane Provider is not allowed to be installed and info message is shown to the user
			checkIfCrossplaneProviderIsNotAllowed(t, cfg, cpWithProviderNotAllowed)
			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Tear down control plane so that we do other tests; this will also delete the Crossplane
			cpResourceDummy := newControlPlaneResource(cfg, cpWithProviderNotAllowed)
			xpres.AwaitResourceDeletionOrFail(ctx, t, cfg, cpResourceDummy)
			return ctx
		}).Feature()

	testEnv.Test(t, feature)
}

// checkIfCrossplaneProviderIsNotAllowed checks if the Crossplane Provider is not allowed to be installed
func checkIfCrossplaneProviderIsNotAllowed(t *testing.T, cfg *envconf.Config, cpName string) {
	cp := GetControlPlaneOrError(t, cfg, cpName)
	for _, c := range cp.Status.Conditions {
		if c.Type == "ProviderKubernetesAbcxyzReady" && c.Reason == juggler.StatusInstallFailed.Name && strings.Contains(c.Message, "ProviderKubernetesAbcxyz not installable: Component provider-kubernetes-abcxyz with version 1.1.1 not found.") {
			return // Crossplane Provider is not allowed to be installed - test passed
		}
	}
	t.Error("Crossplane Provider is allowed to be installed but should not be!")
}
