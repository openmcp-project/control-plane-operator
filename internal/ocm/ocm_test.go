package ocm

import (
	"context"
	"errors"
	"testing"

	"github.com/openmcp-project/control-plane-operator/internal/schemes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	testutils "github.com/openmcp-project/control-plane-operator/test/utils"

	corev1beta1 "github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestGetOCMComponent(t *testing.T) {
	type input struct {
		componentName    string
		version          string
		dockerconfigjson []byte
		validLocalRepo   bool
	}
	type want struct {
		component corev1beta1.ComponentVersion
		err       error
	}
	tests := []struct {
		name  string
		input input
		want  want
	}{
		{
			name: "Error: Can't find nonexistent component in ocm registry",
			input: input{
				dockerconfigjson: []byte("{}"),
				validLocalRepo:   true,
				componentName:    "invalidComponent",
				version:          "",
			},
			want: want{
				component: corev1beta1.ComponentVersion{},
				err:       errors.New("Component %s with version %s not found."),
			},
		},
		{
			name: "Error: Can't find nonexistent version of valid component in ocm registry",
			input: input{
				dockerconfigjson: []byte("{}"),
				validLocalRepo:   true,
				componentName:    "crossplane",
				version:          "0.0.0",
			},
			want: want{
				component: corev1beta1.ComponentVersion{},
				err:       errors.New("Component %s with version %s not found."),
			},
		},
		{
			name: "Get helm component from ocm registry",
			input: input{
				dockerconfigjson: []byte("{}"),
				validLocalRepo:   true,
				componentName:    "crossplane",
				version:          "1.15.0",
			},
			want: want{
				component: corev1beta1.ComponentVersion{
					Version:   "1.15.0",
					HelmRepo:  "https://charts.crossplane.io/stable",
					HelmChart: "crossplane",
				},
			},
		},
		{
			name: "Get oci component from ocm registry",
			input: input{
				dockerconfigjson: []byte("{}"),
				validLocalRepo:   true,
				componentName:    "provider-helm",
				version:          "0.19.0",
			},
			want: want{
				component: corev1beta1.ComponentVersion{
					Version:   "0.19.0",
					DockerRef: "xpkg.upbound.io/crossplane-contrib/provider-helm:v0.19.0",
				},
			},
		},
	}

	initObjs := []client.Object{
		&corev1beta1.ReleaseChannel{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-releasechannel",
			},
			Status: corev1beta1.ReleaseChannelStatus{Components: []corev1beta1.Component{
				{
					Name: "crossplane",
					Versions: []corev1beta1.ComponentVersion{
						{Version: "1.15.0", HelmRepo: "https://charts.crossplane.io/stable", HelmChart: "crossplane"},
					},
				},
				{
					Name: "provider-helm",
					Versions: []corev1beta1.ComponentVersion{
						{Version: "0.19.0", DockerRef: "xpkg.upbound.io/crossplane-contrib/provider-helm:v0.19.0"},
					},
				},
			}},
		},
	}

	c := fake.NewClientBuilder().WithObjects(initObjs...).WithStatusSubresource(initObjs[0]).WithScheme(schemes.Local).Build() //nolint:lll

	ctx := newContext()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.validLocalRepo {
				assert.NoError(t, testutils.SetEnvironmentVariableForLocalOCMTar(testutils.LocalOCMRepositoryPathValid))
			} else {
				assert.NoError(t, testutils.SetEnvironmentVariableForLocalOCMTar(testutils.RepositoryPathInvalid))
			}

			got, err := GetOCMComponent(ctx, c, tt.input.componentName, tt.input.version)

			if tt.want.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Errorf(t, err, tt.want.err.Error())
			}
			assert.Equal(t, got, tt.want.component)
		})
	}
}

func newContext() context.Context {
	ctx := context.Background()
	ctx = log.IntoContext(ctx, log.Log)
	return ctx
}
