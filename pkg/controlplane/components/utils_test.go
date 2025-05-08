//nolint:lll
package components

import (
	"context"
	"strings"
	"testing"

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/secretresolver"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/fluxcd"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/object"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
	"github.com/openmcp-project/control-plane-operator/pkg/utils/rcontext"
)

var (
	tenantNamespace = "tenant-namespace"
	fluxSecretRef   = &meta.KubeConfigReference{
		SecretRef: meta.SecretKeyReference{
			Name: "some-secret",
			Key:  "kubeconfig",
		},
	}
)

func fakeVersionResolver(shouldFail bool) v1beta1.VersionResolverFn {
	return func(componentName string, channelName string) (v1beta1.ComponentVersion, error) {
		if shouldFail {
			return v1beta1.ComponentVersion{}, errFake
		}
		return v1beta1.ComponentVersion{
			DockerRef: strings.ToLower(componentName),
			Version:   "v1.0.0",
		}, nil
	}
}

func fakeSecretRefResolver(shouldFail, shouldReturn bool) secretresolver.ResolveFunc {
	return func(urlType secretresolver.UrlSecretType) (*corev1.LocalObjectReference, error) {
		if shouldFail {
			return nil, errFake
		}
		if shouldReturn {
			return &corev1.LocalObjectReference{Name: "some-secret"}, nil
		}
		return nil, nil
	}
}

type validationFunc func(t *testing.T, ctx context.Context, c juggler.Component)
type targetValidationFunc func(t *testing.T, ctx context.Context, c TargetComponent)
type policyRulesValidationFunc func(t *testing.T, ctx context.Context, c PolicyRulesComponent)
type fluxValidationFunc func(t *testing.T, ctx context.Context, c fluxcd.FluxComponent)
type helmReleaseValidationFunc func(t *testing.T, ctx context.Context, h *fluxcd.HelmReleaseManifesto)
type objectValidationFunc func(t *testing.T, ctx context.Context, c object.ObjectComponent)
type orphanedObjectsDetectorValidationFunc func(t *testing.T, ctx context.Context, dc object.DetectorContext)

func hasName(expected string) validationFunc {
	return func(t *testing.T, ctx context.Context, c juggler.Component) {
		assert.Equal(t, expected, c.GetName(), "GetName does not match")
	}
}

func isEnabled(expected bool) validationFunc {
	return func(t *testing.T, ctx context.Context, c juggler.Component) {
		assert.Equal(t, expected, c.IsEnabled(), "IsEnabled does not match")
	}
}

func isAllowed(expected bool) validationFunc {
	return func(t *testing.T, ctx context.Context, c juggler.Component) {
		actual, _ := c.IsInstallable(ctx)
		assert.Equal(t, expected, actual, "IsInstallable does not match")
	}
}

func hasPreUninstallHook() validationFunc {
	return func(t *testing.T, ctx context.Context, c juggler.Component) {
		assert.NotNil(t, c.Hooks().PreUninstall, "PreUninstall hook is nil")
	}
}

func hasPreInstallHook() validationFunc {
	return func(t *testing.T, ctx context.Context, c juggler.Component) {
		assert.NotNil(t, c.Hooks().PreInstall, "PreInstall hook is nil")
	}
}

func hasPreUpdateHook() validationFunc {
	return func(t *testing.T, ctx context.Context, c juggler.Component) {
		assert.NotNil(t, c.Hooks().PreUpdate, "PreUpdate hook is nil")
	}
}

func hasNoHooks() validationFunc {
	return func(t *testing.T, ctx context.Context, c juggler.Component) {
		assert.Equal(t, juggler.ComponentHooks{}, c.Hooks())
	}
}

func hasDependencies(count int) validationFunc {
	return func(t *testing.T, ctx context.Context, c juggler.Component) {
		assert.Len(t, c.GetDependencies(), count, "len(GetDependencies) count does not match")
	}
}

func newContext(fn secretresolver.ResolveFunc, fn2 v1beta1.VersionResolverFn) context.Context {
	ctx := context.Background()
	ctx = rcontext.WithTenantNamespace(ctx, tenantNamespace)
	ctx = rcontext.WithFluxKubeconfigRef(ctx, &corev1.SecretReference{Name: fluxSecretRef.SecretRef.Name})
	ctx = rcontext.WithSecretRefResolver(ctx, fn)
	ctx = rcontext.WithVersionResolver(ctx, fn2)
	return ctx
}

func isTargetComponent(additionalValidations ...targetValidationFunc) validationFunc {
	return func(t *testing.T, ctx context.Context, c juggler.Component) {
		tc, ok := c.(TargetComponent)
		if !assert.True(t, ok, "not a TargetComponent") {
			return
		}

		for _, v := range additionalValidations {
			v(t, ctx, tc)
		}
	}
}

func hasNamespace(namespace string) targetValidationFunc {
	return func(t *testing.T, ctx context.Context, c TargetComponent) {
		assert.Equal(t, namespace, c.GetNamespace(), "GetNamespace does not match")
	}
}

func isFluxComponent(additionalValidations ...fluxValidationFunc) validationFunc {
	return func(t *testing.T, ctx context.Context, c juggler.Component) {
		fc, ok := c.(fluxcd.FluxComponent)
		if !assert.True(t, ok, "not a FluxComponent") {
			return
		}

		for _, v := range additionalValidations {
			v(t, ctx, fc)
		}
	}
}

func returnsHelmRepo() fluxValidationFunc {
	return func(t *testing.T, ctx context.Context, c fluxcd.FluxComponent) {
		s, err := c.BuildSourceRepository(ctx)
		assert.NoError(t, err)

		h, ok := s.(*fluxcd.HelmRepositoryAdapter)
		if !assert.True(t, ok, "not a HelmRepositoryAdapter") {
			return
		}
		assert.NotNil(t, h.Source)
	}
}

func returnsHelmRelease(additionalValidations ...helmReleaseValidationFunc) fluxValidationFunc {
	return func(t *testing.T, ctx context.Context, c fluxcd.FluxComponent) {
		m, err := c.BuildManifesto(ctx)
		assert.NoError(t, err)

		h, ok := m.(*fluxcd.HelmReleaseManifesto)
		if !assert.True(t, ok, "not a HelmReleaseManifesto") {
			return
		}
		assert.NotNil(t, h.Manifest)

		for _, v := range additionalValidations {
			v(t, ctx, h)
		}
	}
}

func hasKubeconfigRef() helmReleaseValidationFunc {
	return func(t *testing.T, ctx context.Context, h *fluxcd.HelmReleaseManifesto) {
		assert.NotNil(t, h.Manifest.Spec.KubeConfig)
		assert.Equal(t, "kubeconfig", h.Manifest.Spec.KubeConfig.SecretRef.Key, "KubeConfig.SecretRef.Key does not match")
	}
}

func hasHelmValue(expected any, path ...string) helmReleaseValidationFunc {
	return func(t *testing.T, ctx context.Context, h *fluxcd.HelmReleaseManifesto) {
		if assert.NotNil(t, h.Manifest.Spec.Values, "values are nil") {
			actual, err := utils.GetNestedValue(h.Manifest.GetValues(), path...)
			assert.NoError(t, err)
			assert.EqualValues(t, expected, actual)
		}
	}
}

func isObjectComponent(additionalValidations ...objectValidationFunc) validationFunc {
	return func(t *testing.T, ctx context.Context, c juggler.Component) {
		oc, ok := c.(object.ObjectComponent)
		if !assert.True(t, ok, "not an ObjectComponent") {
			return
		}

		for _, v := range additionalValidations {
			v(t, ctx, oc)
		}
	}
}

func objectIsType(sample client.Object) objectValidationFunc {
	return func(t *testing.T, ctx context.Context, c object.ObjectComponent) {
		obj, _, err := c.BuildObjectToReconcile(ctx)
		if !assert.NoError(t, err) {
			return
		}

		assert.IsType(t, sample, obj)
	}
}

func implementsOrphanedObjectsDetector(additionalValidations ...orphanedObjectsDetectorValidationFunc) objectValidationFunc {
	return func(t *testing.T, ctx context.Context, c object.ObjectComponent) {
		ood, ok := c.(object.OrphanedObjectsDetector)
		if !assert.True(t, ok, "not a OrphanedObjectsDetector") {
			return
		}

		for _, v := range additionalValidations {
			v(t, ctx, ood.OrphanDetectorContext())
		}
	}
}

func listTypeIs(sample client.ObjectList) orphanedObjectsDetectorValidationFunc {
	return func(t *testing.T, ctx context.Context, dc object.DetectorContext) {
		assert.IsType(t, sample, dc.ListType)
	}
}

func hasFilterCriteria(count int) orphanedObjectsDetectorValidationFunc {
	return func(t *testing.T, ctx context.Context, dc object.DetectorContext) {
		assert.Len(t, dc.FilterCriteria, count)
	}
}

func canConvert(sample client.ObjectList, count int) orphanedObjectsDetectorValidationFunc {
	return func(t *testing.T, ctx context.Context, dc object.DetectorContext) {
		result := dc.ConvertFunc(sample)
		assert.Len(t, result, count)
	}
}

func canCheckSame(configured, detected juggler.Component, expected bool) orphanedObjectsDetectorValidationFunc {
	return func(t *testing.T, ctx context.Context, dc object.DetectorContext) {
		actual := dc.SameFunc(configured, detected)
		assert.Equal(t, expected, actual, "components %s and %s are not the same", configured.GetName(), detected.GetName())
	}
}

func canCheckHealthiness(sample client.Object, expected juggler.ResourceHealthiness) objectValidationFunc {
	return func(t *testing.T, ctx context.Context, c object.ObjectComponent) {
		actual := c.IsObjectHealthy(sample)
		assert.Equal(t, expected, actual)
	}
}

func canBuildAndReconcile(expectedErr error) objectValidationFunc {
	return func(t *testing.T, ctx context.Context, c object.ObjectComponent) {
		obj, _, err := c.BuildObjectToReconcile(ctx)
		if err != nil {
			assert.ErrorIs(t, err, expectedErr)
			return
		}

		err = c.ReconcileObject(ctx, obj)
		assert.Equal(t, expectedErr, err)
	}
}

func isPolicyRulesComponent(additionalValidations ...policyRulesValidationFunc) validationFunc {
	return func(t *testing.T, ctx context.Context, c juggler.Component) {
		oc, ok := c.(PolicyRulesComponent)
		if !assert.True(t, ok, "not a PolicyRulesComponent") {
			return
		}

		for _, v := range additionalValidations {
			v(t, ctx, oc)
		}
	}
}

func hasPolicyRules() policyRulesValidationFunc {
	return func(t *testing.T, ctx context.Context, c PolicyRulesComponent) {
		assert.NotEmpty(t, c.GetPolicyRules().Admin, "Admin policy rules are empty")
		assert.NotEmpty(t, c.GetPolicyRules().View, "View policy rules are empty")
	}
}
