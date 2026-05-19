//nolint:dupl
package components

import (
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

func Test_Kyverno(t *testing.T) {
	testCases := []struct {
		desc                      string
		config                    *v1beta1.KyvernoConfig
		versionResolver           v1beta1.VersionResolverFn
		availableVersionsResolver v1beta1.AvailableVersionsResolverFn
		validationFuncs           []validationFunc
	}{
		{
			desc: "should be disabled",
			validationFuncs: []validationFunc{
				hasName("Kyverno"),
				isEnabled(false),
			},
		},
		{
			desc: "should not be allowed",
			config: &v1beta1.KyvernoConfig{
				Version: "1.2.3",
			},
			versionResolver: fakeVersionResolver(true),
			validationFuncs: []validationFunc{
				hasName("Kyverno"),
				isEnabled(true),
				isAllowed(false),
			},
		},
		{
			desc:                      "returns available versions from context resolver",
			config:                    &v1beta1.KyvernoConfig{},
			availableVersionsResolver: fakeAvailableVersionsResolver(false),
			validationFuncs: []validationFunc{
				hasName("Kyverno"),
				hasAvailableVersions([]string{"1.1.0", "1.2.0"}),
			},
		},
		{
			desc:                      "returns error when available versions resolver fails",
			config:                    &v1beta1.KyvernoConfig{},
			availableVersionsResolver: fakeAvailableVersionsResolver(true),
			validationFuncs: []validationFunc{
				hasName("Kyverno"),
				hasAvailableVersionsError(errFake),
			},
		},
		{
			desc: "should be enabled",
			config: &v1beta1.KyvernoConfig{
				Version: "1.2.3",
				Values:  &apiextensionsv1.JSON{Raw: []byte(`{"crds":{"install":true}}`)},
			},
			versionResolver: fakeVersionResolver(false),
			validationFuncs: []validationFunc{
				hasName("Kyverno"),
				isEnabled(true),
				isAllowed(true),
				hasPreUninstallHook(),
				hasDependencies(0),
				isTargetComponent(
					hasNamespace("kyverno-system"),
				),
				isFluxComponent(
					returnsHelmRepo(),
					returnsHelmRelease(
						hasKubeconfigRef(),
						hasHelmValue(true, "crds", "install"),
					),
				),
				isPolicyRulesComponent(
					hasPolicyRules(),
				),
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ctx := newContext(nil, tC.versionResolver, tC.availableVersionsResolver)
			c := &Kyverno{Config: tC.config}
			for _, vfn := range tC.validationFuncs {
				vfn(t, ctx, c)
			}
		})
	}
}
