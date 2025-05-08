//nolint:dupl
package components

import (
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

func Test_Flux(t *testing.T) {
	testCases := []struct {
		desc            string
		config          *v1beta1.FluxConfig
		versionResolver v1beta1.VersionResolverFn
		validationFuncs []validationFunc
	}{
		{
			desc: "should be disabled",
			validationFuncs: []validationFunc{
				hasName("Flux"),
				isEnabled(false),
			},
		},
		{
			desc: "should not be allowed",
			config: &v1beta1.FluxConfig{
				Version: "1.2.3",
			},
			versionResolver: fakeVersionResolver(true),
			validationFuncs: []validationFunc{
				hasName("Flux"),
				isEnabled(true),
				isAllowed(false),
			},
		},
		{
			desc: "should be enabled",
			config: &v1beta1.FluxConfig{
				Version: "1.2.3",
				Values:  &apiextensionsv1.JSON{Raw: []byte(`{"clusterDomain":"some-other.local"}`)},
			},
			versionResolver: fakeVersionResolver(false),
			validationFuncs: []validationFunc{
				hasName("Flux"),
				isEnabled(true),
				isAllowed(true),
				hasPreUninstallHook(),
				hasDependencies(0),
				isTargetComponent(
					hasNamespace("flux-system"),
				),
				isFluxComponent(
					returnsHelmRepo(),
					returnsHelmRelease(
						hasKubeconfigRef(),
						hasHelmValue("some-other.local", "clusterDomain"),
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
			ctx := newContext(nil, tC.versionResolver)
			c := &Flux{Config: tC.config}
			for _, vfn := range tC.validationFuncs {
				vfn(t, ctx, c)
			}
		})
	}
}
