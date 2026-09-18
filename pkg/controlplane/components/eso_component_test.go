//nolint:dupl
package components

import (
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

func Test_ExternalSecretsOperator(t *testing.T) {
	testCases := []struct {
		desc                      string
		config                    *v1beta1.ExternalSecretsOperatorConfig
		versionResolver           v1beta1.VersionResolverFn
		availableVersionsResolver v1beta1.AvailableVersionsResolverFn
		validationFuncs           []validationFunc
	}{
		{
			desc: testDescShouldBeDisabled,
			validationFuncs: []validationFunc{
				hasName("ExternalSecretsOperator"),
				isEnabled(false),
			},
		},
		{
			desc: testDescShouldNotBeAllowed,
			config: &v1beta1.ExternalSecretsOperatorConfig{
				Version: testVersion123,
			},
			versionResolver: fakeVersionResolver(true),
			validationFuncs: []validationFunc{
				hasName("ExternalSecretsOperator"),
				isEnabled(true),
				isAllowed(false),
			},
		},
		{
			desc:                      testDescAvailVersions,
			config:                    &v1beta1.ExternalSecretsOperatorConfig{},
			availableVersionsResolver: fakeAvailableVersionsResolver(false),
			validationFuncs: []validationFunc{
				hasName("ExternalSecretsOperator"),
				hasAvailableVersions([]string{testVersion110, testVersion120}),
			},
		},
		{
			desc:                      testDescAvailVersionsErr,
			config:                    &v1beta1.ExternalSecretsOperatorConfig{},
			availableVersionsResolver: fakeAvailableVersionsResolver(true),
			validationFuncs: []validationFunc{
				hasName("ExternalSecretsOperator"),
				hasAvailableVersionsError(errFake),
			},
		},
		{
			desc: testDescShouldBeEnabled,
			config: &v1beta1.ExternalSecretsOperatorConfig{
				Version: testVersion123,
				Values:  &apiextensionsv1.JSON{Raw: []byte(`{"replicaCount":2}`)},
			},
			versionResolver: fakeVersionResolver(false),
			validationFuncs: []validationFunc{
				hasName("ExternalSecretsOperator"),
				isEnabled(true),
				isAllowed(true),
				hasPreUninstallHook(),
				hasDependencies(0),
				isTargetComponent(
					hasNamespace("external-secrets"),
				),
				isFluxComponent(
					returnsHelmRepo(),
					returnsHelmRelease(
						hasKubeconfigRef(),
						hasHelmValue(2, "replicaCount"),
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
			c := &ExternalSecretsOperator{Config: tC.config}
			for _, vfn := range tC.validationFuncs {
				vfn(t, ctx, c)
			}
		})
	}
}
