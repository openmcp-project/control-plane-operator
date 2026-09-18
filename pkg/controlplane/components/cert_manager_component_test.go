//nolint:dupl
package components

import (
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

func Test_CertManager(t *testing.T) {
	testCases := []struct {
		desc                      string
		config                    *v1beta1.CertManagerConfig
		versionResolver           v1beta1.VersionResolverFn
		availableVersionsResolver v1beta1.AvailableVersionsResolverFn
		validationFuncs           []validationFunc
	}{
		{
			desc: testDescShouldBeDisabled,
			validationFuncs: []validationFunc{
				hasName("CertManager"),
				isEnabled(false),
			},
		},
		{
			desc: testDescShouldNotBeAllowed,
			config: &v1beta1.CertManagerConfig{
				Version: testVersion123,
			},
			versionResolver: fakeVersionResolver(true),
			validationFuncs: []validationFunc{
				hasName("CertManager"),
				isEnabled(true),
				isAllowed(false),
			},
		},
		{
			desc:                      testDescAvailVersions,
			config:                    &v1beta1.CertManagerConfig{},
			availableVersionsResolver: fakeAvailableVersionsResolver(false),
			validationFuncs: []validationFunc{
				hasName("CertManager"),
				hasAvailableVersions([]string{testVersion110, testVersion120}),
			},
		},
		{
			desc:                      testDescAvailVersionsErr,
			config:                    &v1beta1.CertManagerConfig{},
			availableVersionsResolver: fakeAvailableVersionsResolver(true),
			validationFuncs: []validationFunc{
				hasName("CertManager"),
				hasAvailableVersionsError(errFake),
			},
		},
		{
			desc: testDescShouldBeEnabled,
			config: &v1beta1.CertManagerConfig{
				Version: testVersion123,
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
			ctx := newContext(nil, tC.versionResolver, tC.availableVersionsResolver)
			c := &CertManager{Config: tC.config}
			for _, vfn := range tC.validationFuncs {
				vfn(t, ctx, c)
			}
		})
	}
}
