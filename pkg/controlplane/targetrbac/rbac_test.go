//nolint:lll
package targetrbac

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
)

func TestApply(t *testing.T) {
	tests := []struct {
		name                    string
		serviceAccountReference v1beta1.ServiceAccountReference
		expectedError           *string
		validateFunc            func(ctx context.Context, client client.Client) error
		interceptorFuncs        interceptor.Funcs
	}{
		{
			name: "Check ServiceAccountReference - invalid - error",
			serviceAccountReference: v1beta1.ServiceAccountReference{
				Namespace: "",
				Name:      "",
			},
			expectedError: ptr.To(errSANameOrNamespaceEmpty.Error()),
		},
		{
			name: "Check ServiceAccountReference - valid - no error",
			serviceAccountReference: v1beta1.ServiceAccountReference{
				Namespace: "default",
				Name:      "test",
			},

			expectedError: nil,
		},
		{
			name: "Check ServiceAccountReference - valid - Create ServiceAccount error",
			serviceAccountReference: v1beta1.ServiceAccountReference{
				Namespace: "default",
				Name:      "test",
			},
			interceptorFuncs: interceptor.Funcs{
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if _, ok := obj.(*corev1.ServiceAccount); ok {
						return errors.New("some create error")
					}
					return nil
				},
			},
			expectedError: ptr.To("some create error"),
		},
		{
			name: "Check ServiceAccountReference - valid, Create ClusterRoleBinding error",
			serviceAccountReference: v1beta1.ServiceAccountReference{
				Namespace: "default",
				Name:      "test",
			},
			interceptorFuncs: interceptor.Funcs{
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if _, ok := obj.(*rbacv1.ClusterRoleBinding); ok {
						return errors.New("some create error")
					}
					return nil
				},
			},
			expectedError: ptr.To("some create error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			c := fake.NewClientBuilder().WithInterceptorFuncs(tt.interceptorFuncs).Build()
			actualError := Apply(ctx, c, tt.serviceAccountReference)

			if tt.expectedError != nil {
				assert.EqualError(t, actualError, *tt.expectedError)
			}

			if tt.validateFunc != nil {
				if err := tt.validateFunc(ctx, c); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name                    string
		serviceAccountReference v1beta1.ServiceAccountReference
		objs                    []client.Object
		interceptorFuncs        interceptor.Funcs
		expectedError           *string
		validateFunc            func(ctx context.Context, client client.Client) error
	}{
		{
			name:                    "Check all ClusterRoleBindings deleted with IsManaged label",
			serviceAccountReference: v1beta1.ServiceAccountReference{},
			objs: []client.Object{
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "default:managed",
						Labels: utils.IsManaged(),
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default:not-managed",
					},
				},
			},
			expectedError: nil,
			validateFunc: func(ctx context.Context, client client.Client) error {
				crbList := &rbacv1.ClusterRoleBindingList{}
				err := client.List(ctx, crbList)
				if err != nil {
					return err
				}

				if len(crbList.Items) != 1 {
					return errors.New("expected 1 ClusterRoleBinding")
				}
				return nil
			},
		},
		{
			name:                    "Check all ServiceAccounts deleted with IsManaged label",
			serviceAccountReference: v1beta1.ServiceAccountReference{},
			objs: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "managed",
						Labels: utils.IsManaged(),
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name: "not-managed",
					},
				},
			},
			expectedError: nil,
			validateFunc: func(ctx context.Context, client client.Client) error {
				crbList := &corev1.ServiceAccountList{}
				err := client.List(ctx, crbList)
				if err != nil {
					return err
				}

				if len(crbList.Items) != 1 {
					return errors.New("expected 1 ServiceAccount")
				}
				return nil
			},
		},
		{
			name:                    "Check all ClusterRoleBindings - some delete error",
			serviceAccountReference: v1beta1.ServiceAccountReference{},
			objs: []client.Object{
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "default:managed",
						Labels: utils.IsManaged(),
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default:not-managed",
					},
				},
			},
			interceptorFuncs: interceptor.Funcs{
				DeleteAllOf: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteAllOfOption) error {
					if _, ok := obj.(*rbacv1.ClusterRoleBinding); ok {
						return errors.New("some delete error")
					}
					return nil
				},
			},
			expectedError: ptr.To("some delete error"),
			validateFunc:  nil,
		},
		{
			name:                    "List all ServiceAccounts - error",
			serviceAccountReference: v1beta1.ServiceAccountReference{},
			objs: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "managed",
						Labels: utils.IsManaged(),
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name: "not-managed",
					},
				},
			},
			interceptorFuncs: interceptor.Funcs{
				List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					if _, ok := list.(*corev1.ServiceAccountList); ok {
						return errors.New("some list error")
					}
					return nil
				},
			},
			expectedError: ptr.To("some list error"),
			validateFunc:  nil,
		},
		{
			name:                    "Delete all ServiceAccounts - error",
			serviceAccountReference: v1beta1.ServiceAccountReference{},
			objs: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "managed",
						Labels: utils.IsManaged(),
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name: "not-managed",
					},
				},
			},
			interceptorFuncs: interceptor.Funcs{
				Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					if _, ok := obj.(*corev1.ServiceAccount); ok {
						return errors.New("some delete error")
					}
					return nil
				},
			},
			expectedError: ptr.To("some delete error"),
			validateFunc:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			c := fake.NewClientBuilder().WithInterceptorFuncs(tt.interceptorFuncs).WithObjects(tt.objs...).Build()
			actualError := Delete(ctx, c)
			if tt.expectedError != nil {
				assert.EqualError(t, actualError, *tt.expectedError)
			}
			if tt.validateFunc != nil {
				if err := tt.validateFunc(ctx, c); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}
