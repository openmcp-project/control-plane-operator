//nolint:dupl
package components

import (
	"testing"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func Test_CertManager(t *testing.T) {
	testCases := []struct {
		desc            string
		config          *v1beta1.CertManagerConfig
		versionResolver v1beta1.VersionResolverFn
		validationFuncs []validationFunc
	}{
		{
			desc: "should be disabled",
			validationFuncs: []validationFunc{
				hasName("CertManager"),
				isEnabled(false),
			},
		},
		{
			desc: "should not be allowed",
			config: &v1beta1.CertManagerConfig{
				Version: "1.2.3",
			},
			versionResolver: fakeVersionResolver(true),
			validationFuncs: []validationFunc{
				hasName("CertManager"),
				isEnabled(true),
				isAllowed(false),
			},
		},
		{
			desc: "should be enabled",
			config: &v1beta1.CertManagerConfig{
				Version: "1.2.3",
				Values:  &apiextensionsv1.JSON{Raw: []byte(`{"global":{"logLevel": 3}}`)},
			},
			versionResolver: fakeVersionResolver(false),
			validationFuncs: []validationFunc{
				hasName("CertManager"),
				isEnabled(true),
				isAllowed(true),
				hasPreUninstallHook(),
				hasDependencies(0),
				isTargetComponent(
					hasNamespace("cert-manager"),
				),
				isFluxComponent(
					returnsHelmRepo(),
					returnsHelmRelease(
						hasKubeconfigRef(),
						hasHelmValue(true, "installCRDs"),     // default
						hasHelmValue(3, "global", "logLevel"), // custom value
					),
				),
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ctx := newContext(nil, tC.versionResolver)
			c := &CertManager{Config: tC.config}
			for _, vfn := range tC.validationFuncs {
				vfn(t, ctx, c)
			}
		})
	}
}
