//nolint:lll,dupl
package fluxcd

import (
	"context"
	"errors"
	"reflect"
	"testing"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

var errBoom = errors.New("boom")

const testLabelComponentName = "flux.juggler.test.io/component"

func TestNewFluxReconciler(t *testing.T) {
	tests := []struct {
		name         string
		logger       logr.Logger
		localClient  client.Client
		remoteClient client.Client
		expected     *FluxReconciler
	}{
		{
			name:         "New empty FluxReconciler",
			localClient:  nil,
			remoteClient: nil,
			logger:       logr.Logger{},
			expected: &FluxReconciler{
				logger:       logr.Logger{},
				localClient:  nil,
				remoteClient: nil,
				knownTypes:   sets.Set[reflect.Type]{},
			},
		},
		{
			name:         "New FluxReconciler with localClient and remoteClient",
			localClient:  fake.NewFakeClient(),
			remoteClient: fake.NewFakeClient(),
			logger:       logr.Logger{},
			expected: &FluxReconciler{
				logger:       logr.Logger{},
				localClient:  fake.NewFakeClient(),
				remoteClient: fake.NewFakeClient(),
				knownTypes:   sets.Set[reflect.Type]{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := NewFluxReconciler(tt.logger, tt.localClient, tt.remoteClient, "")
			if !assert.Equal(t, actual, tt.expected) {
				t.Errorf("NewReconciler() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestFluxReconciler_Observe(t *testing.T) {
	tests := []struct {
		name                string
		obj                 juggler.Component
		localObjects        []client.Object
		expectedObservation juggler.ComponentObservation
		expectedError       error
	}{
		{
			name:                "Error not a FluxComponent",
			obj:                 FakeComponent{},
			localObjects:        nil,
			expectedObservation: juggler.ComponentObservation{},
			expectedError:       errNotFluxComponent,
		},
		{
			name: "Error Source",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return nil, errBoom
				}},
			localObjects: nil,
			expectedObservation: juggler.ComponentObservation{
				ResourceExists: false,
			},
			expectedError: errBoom,
		},
		{
			name: "Source and Manifesto not found - resource does not exist, not healthy",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return &HelmRepositoryAdapter{
						Source: &sourcev1.HelmRepository{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
				BuildManifestoFunc: func(ctx context.Context) (Manifesto, error) {
					return &HelmReleaseManifesto{
						Manifest: &helmv2.HelmRelease{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
			},
			localObjects: []client.Object{}, // no objects in the cluster
			expectedObservation: juggler.ComponentObservation{
				ResourceExists: false,
				ResourceHealthiness: juggler.ResourceHealthiness{
					Healthy: false,
					Message: "",
				},
			},
			expectedError: nil,
		},
		{
			name: "Source exists, BuildManifesto error",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return &HelmRepositoryAdapter{
						Source: &sourcev1.HelmRepository{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
				BuildManifestoFunc: func(ctx context.Context) (Manifesto, error) {
					return nil, errBoom

				}},
			localObjects: []client.Object{
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
			},
			expectedObservation: juggler.ComponentObservation{},
			expectedError:       errBoom,
		},
		{
			name: "Source exists, Manifesto not found - resource exists, not healthy",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return &HelmRepositoryAdapter{
						Source: &sourcev1.HelmRepository{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
				BuildManifestoFunc: func(ctx context.Context) (Manifesto, error) {
					return &HelmReleaseManifesto{
						Manifest: &helmv2.HelmRelease{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil

				}},
			localObjects: []client.Object{
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
			},
			expectedObservation: juggler.ComponentObservation{
				ResourceExists: true,
				ResourceHealthiness: juggler.ResourceHealthiness{
					Healthy: false,
					Message: "Unable to check healthiness. Ready condition is not present.",
				},
			},
			expectedError: nil,
		},
		{
			name: "Source exists, Manifesto exists - unable to check healthiness",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return &HelmRepositoryAdapter{
						Source: &sourcev1.HelmRepository{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
				BuildManifestoFunc: func(ctx context.Context) (Manifesto, error) {
					return &HelmReleaseManifesto{
						Manifest: &helmv2.HelmRelease{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
			},
			localObjects: []client.Object{
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
				&helmv2.HelmRelease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
			},
			expectedObservation: juggler.ComponentObservation{
				ResourceExists: true,
				ResourceHealthiness: juggler.ResourceHealthiness{
					Healthy: false,
					Message: "Unable to check healthiness. Ready condition is not present.\nUnable to check healthiness. Ready condition is not present.",
				},
			},
			expectedError: nil,
		},
		{
			name: "Source exists, Manifesto exists, Source is out of date, Manifesto is up to date - resource exists, not up to date and not healthy",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return &HelmRepositoryAdapter{
						Source: &sourcev1.HelmRepository{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
				BuildManifestoFunc: func(ctx context.Context) (Manifesto, error) {
					return &HelmReleaseManifesto{
						Manifest: &helmv2.HelmRelease{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
			},
			localObjects: []client.Object{
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						URL: "out-of-date-maker",
					},
					Status: sourcev1.HelmRepositoryStatus{
						Conditions: []metav1.Condition{
							{
								Type:   "Ready",
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
				&helmv2.HelmRelease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
			},
			expectedObservation: juggler.ComponentObservation{
				ResourceExists: true,
				ResourceHealthiness: juggler.ResourceHealthiness{
					Healthy: false,
					Message: "Unable to check healthiness. Ready condition is not present.", // expected: because manifesto is not ready in status
				},
			},
			expectedError: nil,
		},
		{
			name: "Source exists, Manifesto exists, Source is up to date, Manifesto is out of date - resource exists, not up to date and not healthy",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return &HelmRepositoryAdapter{
						Source: &sourcev1.HelmRepository{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
				BuildManifestoFunc: func(ctx context.Context) (Manifesto, error) {
					return &HelmReleaseManifesto{
						Manifest: &helmv2.HelmRelease{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
			},
			localObjects: []client.Object{
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
				&helmv2.HelmRelease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
					Spec: helmv2.HelmReleaseSpec{
						TargetNamespace: "out-of-date-maker",
					},
					Status: helmv2.HelmReleaseStatus{
						Conditions: []metav1.Condition{
							{
								Type:   "Ready",
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			expectedObservation: juggler.ComponentObservation{
				ResourceExists: true,
				ResourceHealthiness: juggler.ResourceHealthiness{
					Healthy: false,
					Message: "Unable to check healthiness. Ready condition is not present.", // expected: because source is not ready in status
				},
			},
			expectedError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			r := NewFluxReconciler(logr.Logger{}, fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.localObjects...).Build(), nil, testLabelComponentName)
			actualObservation, actualError := r.Observe(context.TODO(), tt.obj)
			if !assert.Equal(t, tt.expectedObservation, actualObservation) {
				t.Errorf("ObjectReconciler.Observe() = %v, want %v", actualObservation, tt.expectedObservation)
			}
			if !assert.Equal(t, tt.expectedError, actualError) {
				t.Errorf("ObjectReconciler.Observe() = %v, want %v", actualError, tt.expectedError)
			}
		})
	}
}

func TestFluxReconciler_PreUninstall(t *testing.T) {
	tests := []struct {
		name     string
		obj      juggler.Component
		expected error
	}{
		{
			name: "FakeFluxComponent no PreUninstall Hooks - no error",
			obj: FakeFluxComponent{
				HookFunc: juggler.ComponentHooks{
					PreUninstall: nil,
				},
			},
			expected: nil,
		},
		{
			name: "FakeFluxComponent PreUninstall error",
			obj: FakeFluxComponent{
				HookFunc: juggler.ComponentHooks{
					PreUninstall: func(ctx context.Context, client client.Client) error {
						return errBoom
					},
				},
			},
			expected: errBoom,
		},
		{
			name: "FakeFluxComponent PreUninstall no error",
			obj: FakeFluxComponent{
				HookFunc: juggler.ComponentHooks{
					PreUninstall: func(ctx context.Context, client client.Client) error {
						return nil
					},
				},
			},
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &FluxReconciler{}
			actual := r.PreUninstall(context.TODO(), tt.obj)
			if !errors.Is(actual, tt.expected) {
				t.Errorf("ObjectReconciler.PreUninstall() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestFluxReconciler_PreInstall(t *testing.T) {
	tests := []struct {
		name     string
		obj      juggler.Component
		expected error
	}{
		{
			name: "FakeFluxComponent no PreInstall Hooks - no error",
			obj: FakeFluxComponent{
				HookFunc: juggler.ComponentHooks{
					PreInstall: nil,
				},
			},
			expected: nil,
		},
		{
			name: "FakeFluxComponent PreInstall error",
			obj: FakeFluxComponent{
				HookFunc: juggler.ComponentHooks{
					PreInstall: func(ctx context.Context, client client.Client) error {
						return errBoom
					},
				},
			},
			expected: errBoom,
		},
		{
			name: "FakeFluxComponent PreInstall no error",
			obj: FakeFluxComponent{
				HookFunc: juggler.ComponentHooks{
					PreInstall: func(ctx context.Context, client client.Client) error {
						return nil
					},
				},
			},
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &FluxReconciler{}
			actual := r.PreInstall(context.TODO(), tt.obj)
			if !errors.Is(actual, tt.expected) {
				t.Errorf("ObjectReconciler.PreInstall() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestFluxReconciler_PreUpdate(t *testing.T) {
	tests := []struct {
		name     string
		obj      juggler.Component
		expected error
	}{
		{
			name: "FakeFluxComponent no PreUpdate Hooks - no error",
			obj: FakeFluxComponent{
				HookFunc: juggler.ComponentHooks{
					PreUpdate: nil,
				},
			},
			expected: nil,
		},
		{
			name: "FakeFluxComponent PreUpdate error",
			obj: FakeFluxComponent{
				HookFunc: juggler.ComponentHooks{
					PreUpdate: func(ctx context.Context, client client.Client) error {
						return errBoom
					},
				},
			},
			expected: errBoom,
		},
		{
			name: "FakeFluxComponent PreUpdate no error",
			obj: FakeFluxComponent{
				HookFunc: juggler.ComponentHooks{
					PreUpdate: func(ctx context.Context, client client.Client) error {
						return nil
					},
				},
			},
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &FluxReconciler{}
			actual := r.PreUpdate(context.TODO(), tt.obj)
			if !errors.Is(actual, tt.expected) {
				t.Errorf("ObjectReconciler.PreUpdate() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestFluxReconciler_Uninstall(t *testing.T) {
	tests := []struct {
		name         string
		obj          juggler.Component
		localObjects []client.Object
		expected     error
	}{
		{
			name:         "Not a FluxComponent",
			obj:          FakeComponent{},
			localObjects: nil,
			expected:     errNotFluxComponent,
		},
		{
			name: "Error Build Source Repository",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return nil, errBoom
				},
			},
			localObjects: nil,
			expected:     errBoom,
		},
		{
			name: "Build Source Repository successful - delete Source successful, Build Manifesto error",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return &HelmRepositoryAdapter{
						Source: &sourcev1.HelmRepository{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
				BuildManifestoFunc: func(ctx context.Context) (Manifesto, error) {
					return nil, errBoom
				},
			},
			localObjects: []client.Object{
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
			},
			expected: errBoom,
		},
		{
			name: "Build Manifesto successful - delete Source successful, delete Manifesto successful",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return &HelmRepositoryAdapter{
						Source: &sourcev1.HelmRepository{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
				BuildManifestoFunc: func(ctx context.Context) (Manifesto, error) {
					return &HelmReleaseManifesto{
						Manifest: &helmv2.HelmRelease{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
						},
					}, nil
				},
			},
			localObjects: []client.Object{
				&sourcev1.HelmRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
				&helmv2.HelmRelease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
			},
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewFluxReconciler(logr.Logger{}, fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.localObjects...).Build(), nil, testLabelComponentName)
			actual := r.Uninstall(context.TODO(), tt.obj)
			if !errors.Is(actual, tt.expected) {
				t.Errorf("ObjectReconciler.Uninstall() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestFluxReconciler_Install(t *testing.T) {
	tests := []struct {
		name         string
		obj          juggler.Component
		validateFunc func(ctx context.Context, c client.Client, component juggler.Component) error
		expected     error
	}{
		{
			name:     "Not a FluxComponent",
			obj:      FakeComponent{},
			expected: errNotFluxComponent,
		},
		{
			name: "Error Build Source Repository",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return nil, errBoom
				},
			},
			expected: errBoom,
		},
		{
			name: "Build Source Repository successful - create Source successful, Build Manifesto error",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return &HelmRepositoryAdapter{
						Source: &sourcev1.HelmRepository{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
							Spec: sourcev1.HelmRepositorySpec{
								URL: "test-url",
							},
						},
					}, nil
				},
				BuildManifestoFunc: func(ctx context.Context) (Manifesto, error) {
					return nil, errBoom
				},
				GetNameFunc: "FakeFluxComponent",
			},
			validateFunc: func(ctx context.Context, c client.Client, component juggler.Component) error {
				helmRepo := &sourcev1.HelmRepository{}
				err := c.Get(ctx, client.ObjectKey{Name: "test", Namespace: "default"}, helmRepo)
				if err != nil {
					return err
				}
				if !assert.Equal(t, helmRepo.GetLabels(), map[string]string{
					"app.kubernetes.io/managed-by": "control-plane-operator",
					testLabelComponentName:         component.GetName(),
				}) {
					return errors.New("labels not equal")
				}
				return nil
			},
			expected: errBoom, // because of Build Manifesto error
		},
		{
			name: "Build Manifesto successful, create Manifesto successful",
			obj: FakeFluxComponent{
				BuildSourceRepositoryFunc: func(ctx context.Context) (SourceAdapter, error) {
					return &HelmRepositoryAdapter{
						Source: &sourcev1.HelmRepository{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
							Spec: sourcev1.HelmRepositorySpec{
								URL: "test-url",
							},
						},
					}, nil
				},
				BuildManifestoFunc: func(ctx context.Context) (Manifesto, error) {
					return &HelmReleaseManifesto{
						Manifest: &helmv2.HelmRelease{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "default",
							},
							Spec: helmv2.HelmReleaseSpec{ReleaseName: "test-name"},
						},
					}, nil
				},
				GetNameFunc: "FakeFluxComponent",
			},
			validateFunc: func(ctx context.Context, c client.Client, component juggler.Component) error {
				helmRelease := &helmv2.HelmRelease{}
				err := c.Get(ctx, client.ObjectKey{Name: "test", Namespace: "default"}, helmRelease)
				if err != nil {
					return err
				}
				if !assert.Equal(t, helmRelease.GetLabels(), map[string]string{
					"app.kubernetes.io/managed-by": "control-plane-operator",
					testLabelComponentName:         component.GetName(),
				}) {
					return errors.New("labels not equal")
				}
				return nil
			},
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeLocalClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			r := NewFluxReconciler(logr.Logger{}, fakeLocalClient, nil, testLabelComponentName)
			ctx := context.TODO()
			actual := r.Install(ctx, tt.obj)

			if !errors.Is(actual, tt.expected) {
				t.Errorf("ObjectReconciler.Install() = %v, want %v", actual, tt.expected)
			}
			// validates if the object was created correctly
			if tt.validateFunc != nil {
				if err := tt.validateFunc(ctx, fakeLocalClient, tt.obj); err != nil {
					t.Errorf("ObjectReconciler.Install() = %v", err)
				}
			}
		})
	}
}

func Test_FluxReconciler_Types(t *testing.T) {
	r := NewFluxReconciler(logr.Logger{}, nil, nil, "")
	r.RegisterType(FakeFluxComponent{}, FakeFluxComponent{})
	assert.Len(t, r.KnownTypes(), 1)
	assert.Equal(t, r.KnownTypes()[0], reflect.TypeOf(FakeFluxComponent{}))
}
