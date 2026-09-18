//nolint:dupl
package components

import (
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

const (
	testVersion123             = "1.2.3"
	testVersion110             = "1.1.0"
	testVersion120             = "1.2.0"
	testExampleObjName         = "example"
	testDescShouldBeDisabled   = "should be disabled"
	testDescShouldNotBeAllowed = "should not be allowed"
	testDescShouldBeEnabled    = "should be enabled"
	testDescAvailVersions      = "returns available versions from context resolver"
	testDescAvailVersionsErr   = "returns error when available versions resolver fails"
	testProviderKubernetes     = "provider-kubernetes"
	testKubernetes             = "kubernetes"
	testStr                    = "test"
)

func Test_BTPServiceOperator(t *testing.T) {
	testCases := []struct {
		desc                      string
		config                    *v1beta1.BTPServiceOperatorConfig
		versionResolver           v1beta1.VersionResolverFn
		availableVersionsResolver v1beta1.AvailableVersionsResolverFn
		validationFuncs           []validationFunc
	}{
		{
			desc: testDescShouldBeDisabled,
			validationFuncs: []validationFunc{
				hasName("BTPServiceOperator"),
				isEnabled(false),
			},
		},
		{
			desc: testDescShouldNotBeAllowed,
			config: &v1beta1.BTPServiceOperatorConfig{
				Version: testVersion123,
			},
			versionResolver: fakeVersionResolver(true),
			validationFuncs: []validationFunc{
				hasName("BTPServiceOperator"),
				isEnabled(true),
				isAllowed(false),
			},
		},
		{
			desc:                      testDescAvailVersions,
			config:                    &v1beta1.BTPServiceOperatorConfig{},
			availableVersionsResolver: fakeAvailableVersionsResolver(false),
			validationFuncs: []validationFunc{
				hasName("BTPServiceOperator"),
				hasAvailableVersions([]string{testVersion110, testVersion120}),
			},
		},
		{
			desc:                      testDescAvailVersionsErr,
			config:                    &v1beta1.BTPServiceOperatorConfig{},
			availableVersionsResolver: fakeAvailableVersionsResolver(true),
			validationFuncs: []validationFunc{
				hasName("BTPServiceOperator"),
				hasAvailableVersionsError(errFake),
			},
		},
		{
			desc: testDescShouldBeEnabled,
			config: &v1beta1.BTPServiceOperatorConfig{
				Version: testVersion123,
				Values:  &apiextensionsv1.JSON{Raw: []byte(`{"manager":{"replica_count":2}}`)},
			},
			versionResolver: fakeVersionResolver(false),
			validationFuncs: []validationFunc{
				hasName("BTPServiceOperator"),
				isEnabled(true),
				isAllowed(true),
				hasPreUninstallHook(),
				hasDependencies(1),
				isTargetComponent(
					hasNamespace("sap-btp-service-operator"),
				),
				isFluxComponent(
					returnsHelmRepo(),
					returnsHelmRelease(
						hasKubeconfigRef(),
						hasHelmValue(2, "manager", "replica_count"),               // override default
						hasHelmValue("sap-btp-service-operator", "cluster", "id"), // default
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
			c := &BTPServiceOperator{Config: tC.config}
			for _, vfn := range tC.validationFuncs {
				vfn(t, ctx, c)
			}
		})
	}
}
