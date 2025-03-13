//go:build e2e

package e2e

import (
	"context"
	"testing"

	xpres "github.com/crossplane-contrib/xp-testing/pkg/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestControlplane(t *testing.T) {
	cpName := "cp-minimal-e2e"
	feature := features.New("CO-671 Install a simple Control Plane").
		Setup(SetUpControlPlaneResources("testdata/crs/controlplane-only", cpName)).
		Assess(
			"Check Minimal ControlPlane Created", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				cpObserved := GetControlPlaneOrError(t, cfg, cpName)
				if !isControlPlaneReady(cpObserved) {
					t.Error("ControlPlane resource is not ready")
				}
				return ctx
			},
		).
		Assess(
			"Check Control Plane Deleted", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				cpObserved := GetControlPlaneOrError(t, cfg, cpName)
				xpres.AwaitResourceDeletionOrFail(ctx, t, cfg, cpObserved)
				return ctx
			},
		).Feature()

	testEnv.Test(t, feature)
}
