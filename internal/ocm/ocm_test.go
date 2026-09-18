package ocm

import (
	"context"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openmcp-project/control-plane-operator/internal/schemes"

	testutils "github.com/openmcp-project/control-plane-operator/test/utils"

	"github.com/stretchr/testify/assert"

	corev1beta1 "github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

const (
	testCrossplaneName        = "crossplane"
	testVersion1150           = "1.15.0"
	testCrossplaneHelmRepo    = "https://charts.crossplane.io/stable"
	testProviderHelm          = "provider-helm"
	testVersion0190           = "0.19.0"
	testProviderHelmDockerRef = "xpkg.upbound.io/crossplane-contrib/provider-helm:v0.19.0"
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
				err:       fmt.Errorf("%w: component %s with version %s", ErrComponentVersionNotFound, "invalidComponent", ""),
			},
		},
		{
			name: "Error: Can't find nonexistent version of valid component in ocm registry",
			input: input{
				dockerconfigjson: []byte("{}"),
				validLocalRepo:   true,
				componentName:    testCrossplaneName,
				version:          "0.0.0",
			},
			want: want{
				component: corev1beta1.ComponentVersion{},
				err:       fmt.Errorf("%w: component %s with version %s", ErrComponentVersionNotFound, testCrossplaneName, "0.0.0"),
			},
		},
		{
			name: "Get helm component from ocm registry",
			input: input{
				dockerconfigjson: []byte("{}"),
				validLocalRepo:   true,
				componentName:    testCrossplaneName,
				version:          testVersion1150,
			},
			want: want{
				component: corev1beta1.ComponentVersion{
					Version:   testVersion1150,
					HelmRepo:  testCrossplaneHelmRepo,
					HelmChart: testCrossplaneName,
				},
			},
		},
		{
			name: "Get oci component from ocm registry",
			input: input{
				dockerconfigjson: []byte("{}"),
				validLocalRepo:   true,
				componentName:    testProviderHelm,
				version:          testVersion0190,
			},
			want: want{
				component: corev1beta1.ComponentVersion{
					Version:   testVersion0190,
					DockerRef: testProviderHelmDockerRef,
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
					Name: testCrossplaneName,
					Versions: []corev1beta1.ComponentVersion{
						{Version: testVersion1150, HelmRepo: testCrossplaneHelmRepo, HelmChart: testCrossplaneName},
					},
				},
				{
					Name: testProviderHelm,
					Versions: []corev1beta1.ComponentVersion{
						{Version: testVersion0190, DockerRef: testProviderHelmDockerRef},
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
				assert.Error(t, err)
				assert.EqualError(t, err, tt.want.err.Error())
				assert.ErrorIs(t, err, ErrComponentVersionNotFound)
			}
			assert.Equal(t, got, tt.want.component)
		})
	}
}

func TestGetOCMComponentAvailableVersions(t *testing.T) {
	type input struct {
		componentName    string
		dockerconfigjson []byte
		validLocalRepo   bool
	}
	type want struct {
		versions []string
		err      error
	}
	tests := []struct {
		name  string
		input input
		want  want
	}{
		{
			name: "Can not find versions for component",
			input: input{
				dockerconfigjson: []byte("{}"),
				validLocalRepo:   true,
				componentName:    "invalidComponent",
			},
			want: want{
				versions: nil,
				err:      fmt.Errorf("no versions found for component %s", "invalidComponent"),
			},
		},
		{
			name: "Get single version for component",
			input: input{
				dockerconfigjson: []byte("{}"),
				validLocalRepo:   true,
				componentName:    testCrossplaneName,
			},
			want: want{
				versions: []string{testVersion1150},
				err:      nil,
			},
		},
		{
			name: "Get multiple versions for component sorted by semver",
			input: input{
				dockerconfigjson: []byte("{}"),
				validLocalRepo:   true,
				componentName:    testProviderHelm,
			},
			want: want{
				versions: []string{testVersion0190, "0.20.0"},
				err:      nil,
			},
		},
		{
			name: "Get multiple versions for component not semver",
			input: input{
				dockerconfigjson: []byte("{}"),
				validLocalRepo:   true,
				componentName:    "provider-notsemver",
			},
			want: want{
				versions: []string{"version-b", "version-a"},
				err:      nil,
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
					Name: testCrossplaneName,
					Versions: []corev1beta1.ComponentVersion{
						{Version: testVersion1150, HelmRepo: testCrossplaneHelmRepo, HelmChart: testCrossplaneName},
					},
				},
				{
					Name: testProviderHelm,
					Versions: []corev1beta1.ComponentVersion{
						{Version: testVersion0190, DockerRef: testProviderHelmDockerRef},
						{Version: "0.20.0", DockerRef: "xpkg.upbound.io/crossplane-contrib/provider-helm:v0.20.0"},
					},
				},
				{
					Name: "provider-notsemver",
					Versions: []corev1beta1.ComponentVersion{
						{Version: "version-b", DockerRef: "xpkg.upbound.io/crossplane-contrib/provider-notsemver:version-b"},
						{Version: "version-a", DockerRef: "xpkg.upbound.io/crossplane-contrib/provider-notsemver:version-a"},
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

			got, err := GetOCMComponentAvailableVersions(ctx, c, tt.input.componentName)

			if tt.want.err == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.want.err.Error())
			}
			assert.Equal(t, tt.want.versions, got)
		})
	}
}

func newContext() context.Context {
	ctx := context.Background()
	ctx = log.IntoContext(ctx, log.Log)
	return ctx
}
