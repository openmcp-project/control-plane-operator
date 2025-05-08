package crossplane

import (
	"testing"

	crossplanev1 "github.com/crossplane/crossplane/apis/pkg/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

func TestReconcileProvider(t *testing.T) {
	input := v1beta1.CrossplaneProviderConfig{
		Name:              "sample",
		Version:           "v1.5.2",
		Package:           "repo.example.com/provider/sample:v1.5.2",
		PackagePullPolicy: ptr.To(corev1.PullAlways),
		PackagePullSecrets: []corev1.LocalObjectReference{
			{Name: "my-secret"},
		},
		PackageRuntimeSpec: crossplanev1.PackageRuntimeSpec{
			RuntimeConfigReference: &crossplanev1.RuntimeConfigReference{
				Name: "custom-runtime-config",
			},
		},
	}

	expected := &crossplanev1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "provider-sample",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "control-plane-operator",
			},
		},
		Spec: crossplanev1.ProviderSpec{
			PackageSpec: crossplanev1.PackageSpec{
				Package:           "repo.example.com/provider/sample:v1.5.2",
				PackagePullPolicy: ptr.To(corev1.PullAlways),
				PackagePullSecrets: []corev1.LocalObjectReference{
					{Name: "my-secret"},
				},
			},
			PackageRuntimeSpec: crossplanev1.PackageRuntimeSpec{
				RuntimeConfigReference: &crossplanev1.RuntimeConfigReference{
					Name: "provider-sample", // expect to match provider name
				},
			},
		},
	}

	provider, key := EmptyFromConfig(input)
	provider.SetName(key.Name)
	provider.SetNamespace(key.Namespace)

	err := ReconcileProvider(provider, input)
	assert.NoError(t, err)
	assert.Equal(t, expected, provider)
}

func TestAddProviderPrefix(t *testing.T) {
	tt := []struct {
		desc string
		in   string
		exp  string
	}{
		{
			desc: "no prefix",
			in:   "myprovider",
			exp:  providerPrefix + "myprovider",
		},
		{
			desc: "don't double prefix",
			in:   providerPrefix + "myprovider",
			exp:  providerPrefix + "myprovider",
		},
	}

	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			actual := AddProviderPrefix(tc.in)
			assert.Equal(t, tc.exp, actual)
		})
	}

}

func TestTrimProviderPrefix(t *testing.T) {
	tt := []struct {
		desc string
		in   string
		exp  string
	}{
		{
			desc: "prefix",
			in:   providerPrefix + "myprovider",
			exp:  "myprovider",
		},
		{
			desc: "no prefix",
			in:   "myprovider",
			exp:  "myprovider",
		},
	}

	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			actual := TrimProviderPrefix(tc.in)
			assert.Equal(t, tc.exp, actual)
		})
	}
}
