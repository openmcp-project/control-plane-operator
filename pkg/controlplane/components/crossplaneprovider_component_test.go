//nolint:dupl,lll
package components

import (
	"errors"
	"fmt"
	"testing"

	commonv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	crossplanev1 "github.com/crossplane/crossplane/apis/pkg/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/crossplane"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/secretresolver"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

var (
	errFake = errors.New("some error")

	providerInstallPending = &crossplanev1.Provider{}
	providerHealthy        = &crossplanev1.Provider{
		Status: crossplanev1.ProviderStatus{
			ConditionedStatus: commonv1.ConditionedStatus{
				Conditions: []commonv1.Condition{
					{
						Type:   crossplanev1.TypeInstalled,
						Status: corev1.ConditionTrue,
					},
					{
						Type:    crossplanev1.TypeHealthy,
						Status:  corev1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Healthy",
					},
				},
			},
		},
	}
	providerA = &crossplanev1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "provider-a",
		},
		Spec: crossplanev1.ProviderSpec{
			PackageSpec: crossplanev1.PackageSpec{
				Package: "xpkg.example.com/example/provider-example:v1.0.0",
			},
		},
	}
	providerB = &crossplanev1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "provider-b",
		},
		Spec: crossplanev1.ProviderSpec{
			PackageSpec: crossplanev1.PackageSpec{
				Package: "xpkg.example.com/example/provider-example:v1.0.0",
				// irrelevant field, should still be equal to providerA.
				RevisionHistoryLimit: ptr.To[int64](5),
			},
		},
	}
)

func Test_formatProviderName(t *testing.T) {
	testCases := []struct {
		providerName string
		expected     string
	}{
		{
			providerName: "provider-kubernetes",
			expected:     "ProviderKubernetes",
		},
		{
			providerName: "kubernetes",
			expected:     "ProviderKubernetes",
		},
		{
			providerName: "a",
			expected:     "ProviderA",
		},
		{
			providerName: "provider-btp-account",
			expected:     "ProviderBtpAccount",
		},
		{
			providerName: "btp-account",
			expected:     "ProviderBtpAccount",
		},
	}
	for _, tC := range testCases {
		tName := fmt.Sprintf("%s -> %s", tC.providerName, tC.expected)
		t.Run(tName, func(t *testing.T) {
			actual := formatProviderName(tC.providerName)
			assert.Equal(t, tC.expected, actual)
		})
	}
}

func Test_CrossplaneProvider(t *testing.T) {
	testCases := []struct {
		desc              string
		enabled           bool
		config            *v1beta1.CrossplaneProviderConfig
		versionResolver   v1beta1.VersionResolverFn
		secretRefResolver secretresolver.ResolveFunc
		validationFuncs   []validationFunc
	}{
		{
			desc:    "should be disabled",
			enabled: false,
			validationFuncs: []validationFunc{
				isEnabled(false),
			},
		},
		{
			desc:    "should not be allowed",
			enabled: true,
			config: &v1beta1.CrossplaneProviderConfig{
				Name: "kubernetes",
			},
			versionResolver: fakeVersionResolver(true),
			validationFuncs: []validationFunc{
				hasName("ProviderKubernetes"),
				isEnabled(true),
				isAllowed(false),
			},
		},
		{
			desc:    "should be allowed with prefix",
			enabled: true,
			config: &v1beta1.CrossplaneProviderConfig{
				Name: "provider-kubernetes",
			},
			versionResolver: func(componentName string, channelName string) (v1beta1.ComponentVersion, error) {
				if componentName == "provider-kubernetes" {
					return v1beta1.ComponentVersion{}, nil
				}
				return v1beta1.ComponentVersion{}, errFake
			},
			validationFuncs: []validationFunc{
				hasName("ProviderKubernetes"),
				isEnabled(true),
				isAllowed(true),
			},
		},
		{
			desc:    "should be allowed without prefix",
			enabled: true,
			config: &v1beta1.CrossplaneProviderConfig{
				Name: "kubernetes",
			},
			versionResolver: func(componentName string, channelName string) (v1beta1.ComponentVersion, error) {
				if componentName == "provider-kubernetes" {
					return v1beta1.ComponentVersion{}, nil
				}
				return v1beta1.ComponentVersion{}, errFake
			},
			validationFuncs: []validationFunc{
				hasName("ProviderKubernetes"),
				isEnabled(true),
				isAllowed(true),
			},
		},
		{
			desc:    "should be enabled",
			enabled: true,
			config: &v1beta1.CrossplaneProviderConfig{
				Name: "kubernetes",
			},
			versionResolver:   fakeVersionResolver(false),
			secretRefResolver: fakeSecretRefResolver(false, true),
			validationFuncs: []validationFunc{
				hasName("ProviderKubernetes"),
				isEnabled(true),
				isAllowed(true),
				hasDependencies(1),
				hasPreInstallHook(),
				hasPreUpdateHook(),
				isTargetComponent(
					hasNamespace("crossplane-system"),
				),
				isObjectComponent(
					objectIsType(&crossplanev1.Provider{}),
					canCheckHealthiness(providerInstallPending, juggler.ResourceHealthiness{
						Healthy: false,
						Message: "Provider installation is pending (). ",
					}),
					canCheckHealthiness(providerHealthy, juggler.ResourceHealthiness{
						Healthy: true,
						Message: "Healthy: Healthy",
					}),
					canBuildAndReconcile(nil),
					implementsOrphanedObjectsDetector(
						listTypeIs(&crossplanev1.ProviderList{}),
						hasFilterCriteria(2),
						canConvert(&crossplanev1.ProviderList{Items: []crossplanev1.Provider{*providerA}}, 1),
						canCheckSame(
							&CrossplaneProvider{Config: &v1beta1.CrossplaneProviderConfig{Name: crossplane.TrimProviderPrefix(providerA.Name)}},
							&CrossplaneProvider{Config: &v1beta1.CrossplaneProviderConfig{Name: crossplane.TrimProviderPrefix(providerA.Name)}},
							true),
						canCheckSame(
							&CrossplaneProvider{Config: &v1beta1.CrossplaneProviderConfig{Name: providerA.Name}},
							&CrossplaneProvider{Config: &v1beta1.CrossplaneProviderConfig{Name: crossplane.TrimProviderPrefix(providerA.Name)}},
							true),
						canCheckSame(
							&CrossplaneProvider{Config: &v1beta1.CrossplaneProviderConfig{Name: crossplane.TrimProviderPrefix(providerA.Name)}},
							&CrossplaneProvider{Config: &v1beta1.CrossplaneProviderConfig{Name: crossplane.TrimProviderPrefix(providerB.Name)}},
							false),
					),
				),
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ctx := newContext(tC.secretRefResolver, tC.versionResolver)
			c := &CrossplaneProvider{Config: tC.config, Enabled: tC.enabled}
			for _, vfn := range tC.validationFuncs {
				vfn(t, ctx, c)
			}
		})
	}
}
