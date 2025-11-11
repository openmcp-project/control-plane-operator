//nolint:dupl
package components

import (
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

func Test_Kyverno(t *testing.T) {
	testCases := []struct {
		desc            string
		config          *v1beta1.KyvernoConfig
		enableDefaults  bool
		versionResolver v1beta1.VersionResolverFn
		validationFuncs []validationFunc
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
		{
			desc:           "should use default values instad of values from config when enabled via env",
			enableDefaults: true,
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
						hasHelmValue(false, "config", "preserve"),
						hasHelmValue([]interface{}{
							"[*/*,kyverno,*]",
							"[*/*,istio-system,*]",
							"[*/*,kyma-system,*]",
							"[*/*,kube-system,*]",
							"[*/*,kube-public,*]",
							"[*/*,neo-core,*]",
						}, "config", "resourceFilters"),
						hasHelmValue(5000, "config", "updateRequestThreshold"),
						hasHelmValue([]interface{}{
							"system:nodes",
						}, "config", "excludeGroups"),
						hasHelmValue(map[string]interface{}{
							"matchExpressions": []interface{}{
								map[string]interface{}{
									"key":      "kubernetes.io/metadata.name",
									"operator": "NotIn",
									"values": []interface{}{
										"kube-system",
										"kyverno",
										"istio-system",
										"kube-public",
										"kyma-system",
										"neo-core",
									},
								},
							},
						}, "config", "webhooks", "namespaceSelector"),
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
			if tC.enableDefaults {
				t.Setenv(EnvEnableKyvernoDefaultValues, "true")
			}
			c := &Kyverno{Config: tC.config}
			for _, vfn := range tC.validationFuncs {
				vfn(t, ctx, c)
			}
		})
	}
}
