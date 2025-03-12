//nolint:dupl,lll
package components

import (
	"context"
	"testing"

	crossplanev1beta1 "github.com/crossplane/crossplane/apis/pkg/v1beta1"
	"github.com/google/go-cmp/cmp"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestCrossplaneDeploymentRuntimConfig(t *testing.T) {
	testCases := []struct {
		desc            string
		enabled         bool
		name            string
		validationFuncs []validationFunc
	}{
		{
			desc:    "should be enabled",
			enabled: true,
			name:    "test",
			validationFuncs: []validationFunc{
				hasName("DeploymentRuntimeConfigProviderTest"),
				isEnabled(true),
				isAllowed(true),
				hasDependencies(1),
				hasPreInstallHook(),
				hasPreUpdateHook(),
				isTargetComponent(
					hasNamespace(""),
				),
				isObjectComponent(
					objectIsType(&crossplanev1beta1.DeploymentRuntimeConfig{}),
					// sample does not matter, since we always assert healthy
					canCheckHealthiness(nil, juggler.ResourceHealthiness{
						Healthy: true,
						Message: "DeploymentRuntimeConfig applied",
					}),
					canBuildAndReconcile(nil),
					implementsOrphanedObjectsDetector(
						listTypeIs(&crossplanev1beta1.DeploymentRuntimeConfigList{}),
						hasFilterCriteria(2),
						canConvert(&crossplanev1beta1.DeploymentRuntimeConfigList{Items: []crossplanev1beta1.DeploymentRuntimeConfig{{
							ObjectMeta: metav1.ObjectMeta{Name: "Test"}}}}, 1),
						canCheckSame(&CrossplaneDeploymentRuntimeConfig{Name: "A"}, &CrossplaneDeploymentRuntimeConfig{Name: "A"}, true),
						canCheckSame(&CrossplaneDeploymentRuntimeConfig{Name: "A"}, &CrossplaneDeploymentRuntimeConfig{Name: "B"}, false),
					),
				),
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			c := &CrossplaneDeploymentRuntimeConfig{Name: tC.name, Enabled: tC.enabled}
			for _, vfn := range tC.validationFuncs {
				vfn(t, context.Background(), c)
			}
		})
	}
}

func TestApplyDeploymentTemplateDefaults(t *testing.T) {
	// Test if the deployment coming from Reconcile is valid
	// At this point we really wanted to build the final Deployment
	// using crossplane and then use k8s native validation.
	// However this is currently not possible due to two reasons
	// 1) Crossplane hides the Deploymentbuilder behind its internal package:
	//    see https://github.com/crossplane/crossplane/blob/f714904bc18d02ca1749e884fd31ab738e792c6f/internal/controller/pkg/revision/runtime.go#L112
	//    The required function is inside an `internal` filepath, making it not possible
	//    to easily import it internally.
	// 2) Kubernetes is semi hiding the deployment validation logic:
	//    More specifically: the validation logic is only available inside the
	//    `k8s.io/kubernetes` package https://github.com/kubernetes/kubernetes/blob/48dce2e9b3def93556cd7694edf22a74ecb34aa9/pkg/apis/apps/validation/validation.go#L560
	//    This would require us to import the whole top-level package and set up required replacements for common modules as
	//    `k8s.io/kubernetes` by default replaces them with local instances (see https://github.com/kubernetes/kubernetes/blob/48dce2e9b3def93556cd7694edf22a74ecb34aa9/go.mod#L225
	//    for reference). This would create a rather large dependency.
	//    Additionally the validation is based on an internal DeploymentSpec type, which we would need to write a converter for, for it to be used with appsv1.Deployment
	var defaultDeployTemplate = func() *crossplanev1beta1.DeploymentTemplate {
		return &crossplanev1beta1.DeploymentTemplate{
			Spec: &appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{},
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "package-runtime",
								Args: []string{},
							},
						},
					},
				},
			},
		}
	}

	testCases := []struct {
		desc  string
		inDt  *crossplanev1beta1.DeploymentTemplate
		expDt func() *crossplanev1beta1.DeploymentTemplate
	}{
		{
			desc:  "if no DeploymentTemplate exist, default should be returned",
			inDt:  nil,
			expDt: defaultDeployTemplate,
		},
		{
			desc:  "on empty DeploymentTemplate, default should be returned",
			inDt:  &crossplanev1beta1.DeploymentTemplate{},
			expDt: defaultDeployTemplate,
		},
		{
			desc: "custom selector should be respected, other fields defaulted",
			inDt: &crossplanev1beta1.DeploymentTemplate{
				Spec: &appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"test": "test"},
					},
				},
			},
			expDt: func() *crossplanev1beta1.DeploymentTemplate {
				dt := defaultDeployTemplate()
				dt.Spec.Selector.MatchLabels = map[string]string{"test": "test"}
				return dt
			},
		},
		{
			desc: "custom spec.template.spec should be respected, other fields defaulted",
			inDt: &crossplanev1beta1.DeploymentTemplate{
				Spec: &appsv1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							DNSPolicy: "test",
						},
					},
				},
			},
			expDt: func() *crossplanev1beta1.DeploymentTemplate {
				dt := defaultDeployTemplate()
				dt.Spec.Template.Spec.DNSPolicy = "test"
				return dt
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			res := applyDeploymentTemplateDefaults(tC.inDt)
			exp := tC.expDt()

			if !cmp.Equal(res, exp) {
				t.Error(cmp.Diff(res, exp))
			}

		})
	}
}

func TestApplyServiceAccountTemplateDefaults(t *testing.T) {
	providerName := "test"
	var defaultServiceAccountTemplate = func() *crossplanev1beta1.ServiceAccountTemplate {
		return &crossplanev1beta1.ServiceAccountTemplate{
			Metadata: &crossplanev1beta1.ObjectMeta{
				Name: &providerName,
			},
		}
	}

	testCases := []struct {
		desc  string
		inSt  *crossplanev1beta1.ServiceAccountTemplate
		expSt func() *crossplanev1beta1.ServiceAccountTemplate
	}{
		{
			desc:  "if no ServiceAccountTemplate exist, default should be returned",
			inSt:  nil,
			expSt: defaultServiceAccountTemplate,
		},
		{
			desc:  "on empty ServiceAccountTemplate, default should be returned",
			inSt:  &crossplanev1beta1.ServiceAccountTemplate{},
			expSt: defaultServiceAccountTemplate,
		},
		{
			desc:  "on empty ServiceAccountTemplate, default should be returned",
			inSt:  &crossplanev1beta1.ServiceAccountTemplate{},
			expSt: defaultServiceAccountTemplate,
		},
		{
			desc: "custom metadata.name is respected",
			inSt: &crossplanev1beta1.ServiceAccountTemplate{
				Metadata: &crossplanev1beta1.ObjectMeta{
					Name: ptr.To("test"),
				},
			},
			expSt: func() *crossplanev1beta1.ServiceAccountTemplate {
				st := defaultServiceAccountTemplate()
				st.Metadata.Name = ptr.To("test")
				return st
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			res := applyServiceAccountTemplateDefaults(tC.inSt, providerName)
			exp := tC.expSt()

			if !cmp.Equal(res, exp) {
				t.Error(cmp.Diff(res, exp))
			}

		})
	}

}
