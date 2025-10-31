//nolint:lll,dupl
package object

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

var errBoom = errors.New("boom")

const (
	testLabelComponentKey   = "object.juggler.test.io/component"
	testLabelManagedByKey   = "object.juggler.test.io/managedBy"
	testLabelManagedByValue = "object.juggler.test.io/control-plane-operator"
)

func TestObjectReconciler_Install(t *testing.T) {
	tests := []struct {
		name          string
		obj           juggler.Component
		remoteObjects []client.Object
		labelFunc     juggler.LabelFunc
		error         error
		validateFunc  func(ctx context.Context, c client.Client, comp juggler.Component) error
	}{
		{
			name: "nil",
			obj:  nil,

			error: errNotObjectComponent,
		},
		{
			name: "nil",
			obj:  FakeComponent{},

			error: errNotObjectComponent,
		},
		{
			name: "ObjectComponent BuildObject error",
			obj: FakeObjectComponent{
				BuildObjectToReconcileFunc: func(ctx context.Context) (client.Object, types.NamespacedName, error) {
					return nil, types.NamespacedName{}, errBoom
				},
			},
			error: errBoom,
		},
		{
			name: "ObjectComponent BuildObject successful - Creation successful",
			obj: FakeObjectComponent{
				BuildObjectToReconcileFunc: func(ctx context.Context) (client.Object, types.NamespacedName, error) {
					return &corev1.Secret{}, types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					}, nil
				},
				ReconcileObjectFunc: func(ctx context.Context, obj client.Object) error {
					return nil
				},
				name: "FakeObjectComponent",
			},
			validateFunc: func(ctx context.Context, c client.Client, comp juggler.Component) error {
				secret := &corev1.Secret{}
				err := c.Get(ctx, client.ObjectKey{Name: "test", Namespace: "default"}, secret)
				if err != nil {
					return err
				}
				if !assert.Equal(t, secret.GetLabels(), map[string]string{
					"app.kubernetes.io/managed-by": "control-plane-operator",
					testLabelComponentKey:          comp.GetName(),
				}) {
					return errors.New("labels not equal")
				}
				return nil
			},
		},
		{
			name: "ObjectComponent BuildObject successful - Object already there - Update successful",
			obj: FakeObjectComponent{
				BuildObjectToReconcileFunc: func(ctx context.Context) (client.Object, types.NamespacedName, error) {
					return &corev1.Secret{}, types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					}, nil
				},
				ReconcileObjectFunc: func(ctx context.Context, obj client.Object) error {
					secret := obj.(*corev1.Secret)
					secret.Type = corev1.SecretTypeDockerConfigJson
					return nil
				},
				name: "FakeObjectComponent",
			},
			remoteObjects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
					Type: corev1.SecretTypeOpaque, // different type
				},
			},
			validateFunc: func(ctx context.Context, c client.Client, comp juggler.Component) error {
				secret := &corev1.Secret{}
				err := c.Get(ctx, client.ObjectKey{Name: "test", Namespace: "default"}, secret)
				if err != nil {
					return err
				}
				if !assert.Equal(t, secret.GetLabels(), map[string]string{
					"app.kubernetes.io/managed-by": "control-plane-operator",
					testLabelComponentKey:          comp.GetName(),
				}) {
					return errors.New("labels not equal")
				}
				if !assert.Equal(t, secret.Type, corev1.SecretTypeDockerConfigJson) {
					return errors.New("type not equal")
				}
				return nil
			},
		},
		{
			name: "ObjectReconciler with custom label func - creation successful",
			obj: FakeObjectComponent{
				BuildObjectToReconcileFunc: func(ctx context.Context) (client.Object, types.NamespacedName, error) {
					return &corev1.Secret{}, types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					}, nil
				},
				ReconcileObjectFunc: func(ctx context.Context, obj client.Object) error {
					return nil
				},
				name: "FakeObjectComponent",
			},
			labelFunc: func(comp juggler.Component) map[string]string {
				return map[string]string{
					testLabelComponentKey: comp.GetName(),
					testLabelManagedByKey: testLabelManagedByValue,
				}
			},
			validateFunc: func(ctx context.Context, c client.Client, comp juggler.Component) error {
				secret := &corev1.Secret{}
				if err := c.Get(ctx, client.ObjectKey{Name: "test", Namespace: "default"}, secret); err != nil {
					return err
				}
				if !assert.Equal(t, secret.GetLabels(), map[string]string{
					testLabelComponentKey: comp.GetName(),
					testLabelManagedByKey: testLabelManagedByValue,
				}) {
					return errors.New("labels not equal")
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRemoteClient := fake.NewClientBuilder().WithObjects(tt.remoteObjects...).Build()
			r := NewReconciler(logr.Logger{}, fakeRemoteClient, testLabelComponentKey).
				WithLabelFunc(tt.labelFunc)
			ctx := context.TODO()
			actual := r.Install(ctx, tt.obj)
			if !errors.Is(actual, tt.error) {
				t.Errorf("ObjectReconciler.Install() = %v, want %v", actual, tt.error)
			}
			// validates if the object was created correctly
			if tt.validateFunc != nil {
				if err := tt.validateFunc(ctx, fakeRemoteClient, tt.obj); err != nil {
					t.Errorf("ObjectReconciler.Install() = %v, want %v", err, nil)
				}
			}
		})
	}
}

func TestObjectReconciler_PreUninstall(t *testing.T) {
	tests := []struct {
		name     string
		obj      juggler.Component
		expected error
	}{
		{
			name: "ObjectComponent no PreUninstall Hooks - no error",
			obj: FakeObjectComponent{
				hooks: juggler.ComponentHooks{
					PreUninstall: nil,
				},
			},
			expected: nil,
		},
		{
			name: "ObjectComponent PreUninstall error",
			obj: FakeObjectComponent{
				hooks: juggler.ComponentHooks{
					PreUninstall: func(ctx context.Context, client client.Client) error {
						return errBoom
					},
				},
			},
			expected: errBoom,
		},
		{
			name: "ObjectComponent PreUninstall no error",
			obj: FakeObjectComponent{
				hooks: juggler.ComponentHooks{
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
			r := &ObjectReconciler{}
			actual := r.PreUninstall(context.TODO(), tt.obj)
			if !errors.Is(actual, tt.expected) {
				t.Errorf("ObjectReconciler.PreUninstall() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestObjectReconciler_PreInstall(t *testing.T) {
	tests := []struct {
		name     string
		obj      juggler.Component
		expected error
	}{
		{
			name: "ObjectComponent no PreInstall Hooks - no error",
			obj: FakeObjectComponent{
				hooks: juggler.ComponentHooks{
					PreInstall: nil,
				},
			},
			expected: nil,
		},
		{
			name: "ObjectComponent PreInstall error",
			obj: FakeObjectComponent{
				hooks: juggler.ComponentHooks{
					PreInstall: func(ctx context.Context, client client.Client) error {
						return errBoom
					},
				},
			},
			expected: errBoom,
		},
		{
			name: "ObjectComponent PreInstall no error",
			obj: FakeObjectComponent{
				hooks: juggler.ComponentHooks{
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
			r := &ObjectReconciler{}
			actual := r.PreInstall(context.TODO(), tt.obj)
			if !errors.Is(actual, tt.expected) {
				t.Errorf("ObjectReconciler.PreInstall() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestObjectReconciler_PreUpdate(t *testing.T) {
	tests := []struct {
		name     string
		obj      juggler.Component
		expected error
	}{
		{
			name: "ObjectComponent no PreUpdate Hooks - no error",
			obj: FakeObjectComponent{
				hooks: juggler.ComponentHooks{
					PreUpdate: nil,
				},
			},
			expected: nil,
		},
		{
			name: "ObjectComponent PreUpdate error",
			obj: FakeObjectComponent{
				hooks: juggler.ComponentHooks{
					PreUpdate: func(ctx context.Context, client client.Client) error {
						return errBoom
					},
				},
			},
			expected: errBoom,
		},
		{
			name: "ObjectComponent PreUpdate no error",
			obj: FakeObjectComponent{
				hooks: juggler.ComponentHooks{
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
			r := &ObjectReconciler{}
			actual := r.PreUpdate(context.TODO(), tt.obj)
			if !errors.Is(actual, tt.expected) {
				t.Errorf("ObjectReconciler.PreUpdate() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestObjectReconciler_Uninstall(t *testing.T) {
	tests := []struct {
		name          string
		obj           juggler.Component
		remoteObjects []client.Object
		validateFunc  func(ctx context.Context, c client.Client, obj client.Object) error
		expected      error
	}{
		{
			name:          "nil",
			obj:           nil,
			remoteObjects: nil,
			expected:      errNotObjectComponent,
		},
		{
			name:          "Component - error",
			obj:           FakeComponent{},
			remoteObjects: nil,
			expected:      errNotObjectComponent,
		},
		{
			name: "ObjectComponent BuildObject error",
			obj: FakeObjectComponent{
				BuildObjectToReconcileFunc: func(ctx context.Context) (client.Object, types.NamespacedName, error) {
					return nil, types.NamespacedName{}, errBoom
				},
			},
			remoteObjects: nil,
			expected:      errBoom,
		},
		{
			name: "Uninstall not successful - Object (Secret) Not Found",
			obj: FakeObjectComponent{
				BuildObjectToReconcileFunc: func(ctx context.Context) (client.Object, types.NamespacedName, error) {
					return &corev1.Secret{}, types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					}, nil
				},
			},
			remoteObjects: []client.Object{},
			expected:      nil,
		},
		{
			name: "Uninstall not successful - Object found and deleted",
			obj: FakeObjectComponent{
				BuildObjectToReconcileFunc: func(ctx context.Context) (client.Object, types.NamespacedName, error) {
					return &corev1.Secret{}, types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					}, nil
				},
			},
			remoteObjects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
			},
			validateFunc: func(ctx context.Context, c client.Client, obj client.Object) error {
				// checks whether the object is deleted
				if err := c.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
					return nil
				}
				return errors.New("object not deleted")
			},
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(tt.remoteObjects...).Build()
			r := NewReconciler(logr.Logger{}, fakeClient, testLabelComponentKey)
			ctx := context.TODO()
			actual := r.Uninstall(ctx, tt.obj)
			if !errors.Is(actual, tt.expected) {
				t.Errorf("ObjectReconciler.Uninstall() = %v, want %v", actual, tt.expected)
			}
			if tt.validateFunc != nil {
				if err := tt.validateFunc(ctx, fakeClient, tt.remoteObjects[0]); err != nil {
					t.Errorf("ObjectReconciler.Uninstall() = %v, want %v", err, nil)
				}
			}
		})
	}
}

func TestNewReconciler(t *testing.T) {
	fakeClient := fake.NewFakeClient()
	tests := []struct {
		name         string
		logger       logr.Logger
		remoteClient client.Client
		expected     *ObjectReconciler
	}{
		{
			name:         "New ObjectReconciler",
			logger:       logr.Logger{},
			remoteClient: nil,
			expected: &ObjectReconciler{
				remoteClient: nil,
				logger:       logr.Logger{},
				knownTypes:   sets.Set[reflect.Type]{},
			},
		},
		{
			name:         "New ObjectReconciler with remoteClient",
			logger:       logr.Logger{},
			remoteClient: fakeClient,
			expected: &ObjectReconciler{
				remoteClient: fakeClient,
				logger:       logr.Logger{},
				knownTypes:   sets.Set[reflect.Type]{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := NewReconciler(tt.logger, tt.remoteClient, "")
			diff := cmp.Diff(actual, tt.expected, cmp.Comparer(func(a, b ObjectReconciler) bool {
				return actual.remoteClient == tt.remoteClient &&
					actual.logger == tt.logger &&
					actual.knownTypes.Equal(tt.expected.knownTypes)
			}))
			if !assert.Empty(t, diff) {
				t.Errorf("NewReconciler() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestObjectReconciler_Observe(t *testing.T) {
	tests := []struct {
		name                string
		obj                 juggler.Component
		remoteObjects       []client.Object
		expectedObservation juggler.ComponentObservation
		expectedError       error
	}{
		{
			name:                "Error not a ObjectComponent",
			obj:                 FakeComponent{},
			remoteObjects:       nil,
			expectedObservation: juggler.ComponentObservation{},
			expectedError:       errNotObjectComponent,
		},
		{
			name: "ObjectComponent BuildObject Error",
			obj: FakeObjectComponent{
				BuildObjectToReconcileFunc: func(ctx context.Context) (client.Object, types.NamespacedName, error) {
					return nil, types.NamespacedName{}, errBoom
				},
			},
			remoteObjects:       nil,
			expectedObservation: juggler.ComponentObservation{},
			expectedError:       errBoom,
		},
		{
			name: "ObjectComponent BuildObject successful - Object not found",
			obj: FakeObjectComponent{
				BuildObjectToReconcileFunc: func(ctx context.Context) (client.Object, types.NamespacedName, error) {
					return &corev1.Secret{}, types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					}, nil
				},
			},
			remoteObjects:       nil,
			expectedObservation: juggler.ComponentObservation{ResourceExists: false},
			expectedError:       nil,
		},
		{
			name: "ObjectComponent BuildObject successful - Object found - Resource is not healthy",
			obj: FakeObjectComponent{
				BuildObjectToReconcileFunc: func(ctx context.Context) (client.Object, types.NamespacedName, error) {
					return &corev1.Secret{}, types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					}, nil
				},
				IsObjectHealthyFunc: func(obj client.Object) juggler.ResourceHealthiness {
					return juggler.ResourceHealthiness{ // not healthy
						Healthy: false,
						Message: "not healthy",
					}
				},
			},
			remoteObjects: []client.Object{
				&corev1.Secret{
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
					Message: "not healthy",
				},
			},
		},
		{
			name: "ObjectComponent BuildObject successful - Object found - Resource is healthy",
			obj: FakeObjectComponent{
				BuildObjectToReconcileFunc: func(ctx context.Context) (client.Object, types.NamespacedName, error) {
					return &corev1.Secret{}, types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					}, nil
				},
				IsObjectHealthyFunc: func(obj client.Object) juggler.ResourceHealthiness {
					return juggler.ResourceHealthiness{ // healthy
						Healthy: true,
						Message: "",
					}
				},
			},
			remoteObjects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
				},
			},
			expectedObservation: juggler.ComponentObservation{
				ResourceExists: true,
				ResourceHealthiness: juggler.ResourceHealthiness{
					Healthy: true,
					Message: "",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReconciler(logr.Logger{}, fake.NewClientBuilder().WithObjects(tt.remoteObjects...).Build(), testLabelComponentKey)
			ctx := context.TODO()
			actualObservation, actualError := r.Observe(ctx, tt.obj)
			if !assert.Equal(t, tt.expectedObservation, actualObservation) {
				t.Errorf("ObjectReconciler.Observe() = %v, want %v", actualObservation, tt.expectedObservation)
			}
			if !assert.Equal(t, tt.expectedError, actualError) {
				t.Errorf("ObjectReconciler.Observe() = %v, want %v", actualError, tt.expectedError)
			}
		})
	}
}

func Test_ObjectReconciler_Types(t *testing.T) {
	r := NewReconciler(logr.Logger{}, nil, "")
	r.RegisterType(FakeObjectComponent{}, FakeObjectComponent{})
	assert.Len(t, r.KnownTypes(), 1)
	assert.Equal(t, r.KnownTypes()[0], reflect.TypeOf(FakeObjectComponent{}))
}

func Test_ObjectReconciler_DetectOrphanedComponents(t *testing.T) {
	testCases := []struct {
		desc                 string
		initObjs             []client.Object
		interceptorFuncs     interceptor.Funcs
		configuredComponents []juggler.Component
		expectedComps        []juggler.Component
		expectedErr          error
	}{
		{
			desc: "should find exactly one orphaned component",
			initObjs: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "orphaned-cm",
						Labels: map[string]string{
							fakeFilterLabel: "true",
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "configured-cm",
						Labels: map[string]string{
							fakeFilterLabel: "true",
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-unrelated-cm",
					},
				},
			},
			configuredComponents: []juggler.Component{
				FakeObjectComponent{name: "configured-cm", enabled: true},
			},
			expectedComps: []juggler.Component{
				FakeObjectComponent{name: "orphaned-cm", enabled: false},
			},
			expectedErr: nil,
		},
		{
			desc: "should not return error when CRD is not installed",
			configuredComponents: []juggler.Component{
				FakeObjectComponent{name: "configured-cm", enabled: true},
			},
			interceptorFuncs: interceptor.Funcs{
				List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					return &apiutil.ErrResourceDiscoveryFailed{
						schema.GroupVersion{Group: corev1.GroupName, Version: "v1"}: apierrors.NewNotFound(corev1.Resource("configmaps"), "ConfigMap"),
					}
				},
			},
			expectedComps: []juggler.Component{},
			expectedErr:   nil,
		},
		{
			desc: "should return error when unexpected error happens",
			configuredComponents: []juggler.Component{
				FakeObjectComponent{name: "configured-cm", enabled: true},
			},
			interceptorFuncs: interceptor.Funcs{
				List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					return errBoom
				},
			},
			expectedComps: nil,
			expectedErr:   errBoom,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(tC.initObjs...).WithInterceptorFuncs(tC.interceptorFuncs).Build()
			r := NewReconciler(logr.Logger{}, fakeClient, testLabelComponentKey)
			for _, cc := range tC.configuredComponents {
				r.RegisterType(cc.(ObjectComponent))
			}
			actualComps, actualErr := r.DetectOrphanedComponents(context.Background(), tC.configuredComponents)
			assert.Equal(t, tC.expectedErr, actualErr)
			if tC.expectedComps == nil {
				assert.Nil(t, actualComps)
				return
			}
			if !assert.Len(t, actualComps, len(tC.expectedComps)) {
				return
			}
			for i, ac := range actualComps {
				ec := tC.expectedComps[i]
				assert.Equal(t, ac.GetName(), ec.GetName())
				assert.Equal(t, ac.IsEnabled(), ec.IsEnabled())
			}
		})
	}
}
