package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/openmcp-project/control-plane-operator/pkg/constants"
)

var (
	secretWithLabel = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-secret",
			Namespace: corev1.NamespaceDefault,
			Labels: map[string]string{
				constants.LabelCopyToCPNamespace: "true",
			},
		},
	}
	secretDeletedWithLabelAndFinalizer = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-secret",
			Namespace: corev1.NamespaceDefault,
			Labels: map[string]string{
				constants.LabelCopyToCPNamespace: "true",
			},
			DeletionTimestamp: ptr.To(metav1.Now()),
			Finalizers: []string{
				finalizerOrphan,
			},
		},
	}
	secretWithoutLabel = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-secret",
			Namespace: corev1.NamespaceDefault,
		},
	}
	secretWithFinalizer = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-secret",
			Namespace: corev1.NamespaceDefault,
			Finalizers: []string{
				finalizerOrphan,
			},
		},
	}
	replicatedSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-secret",
			Namespace: tenantNamespace.Name,
			Labels: map[string]string{
				constants.LabelCopySourceName:      secretWithLabel.Name,
				constants.LabelCopySourceNamespace: secretWithLabel.Namespace,
			},
		},
	}
	unrelatedSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unrelated-secret",
			Namespace: tenantNamespace.Name,
		},
	}
	tenantNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: cpNamespacePrefix + "example",
		},
	}
	otherNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "example-system",
		},
	}

	errTest = errors.New("some error")
)

func Test_SecretReconciler_Reconcile(t *testing.T) {
	testCases := []struct {
		desc             string
		initObjs         []client.Object
		interceptorFuncs interceptor.Funcs
		expectedResult   ctrl.Result
		expectedErr      error
		validate         func(t *testing.T, ctx context.Context, c client.Client) error
	}{
		{
			desc: "should not return error when object is not found",
			initObjs: []client.Object{
				secretWithLabel.DeepCopy(),
				tenantNamespace.DeepCopy(),
			},
			interceptorFuncs: interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					return apierrors.NewNotFound(corev1.Resource("secrets"), "some-secret")
				},
			},
		},
		{
			desc: "should return error when client returns unknown error",
			initObjs: []client.Object{
				secretWithLabel.DeepCopy(),
				tenantNamespace.DeepCopy(),
			},
			interceptorFuncs: interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					return errTest
				},
			},
			expectedErr: errTest,
		},
		{
			desc: "should not do anything when secret has no label",
			initObjs: []client.Object{
				secretWithoutLabel.DeepCopy(),
				tenantNamespace.DeepCopy(),
			},
			validate: func(t *testing.T, ctx context.Context, c client.Client) error {
				secrets := &corev1.SecretList{}
				if err := c.List(ctx, secrets); err != nil {
					return err
				}
				assert.Len(t, secrets.Items, 1)
				return nil
			},
		},
		{
			desc: "should replicate to tenant namespace",
			initObjs: []client.Object{
				secretWithLabel.DeepCopy(),
				tenantNamespace.DeepCopy(),
				otherNamespace.DeepCopy(),
			},
			validate: func(t *testing.T, ctx context.Context, c client.Client) error {
				secrets := &corev1.SecretList{}
				if err := c.List(ctx, secrets); err != nil {
					return err
				}
				assert.Len(t, secrets.Items, 2)
				return nil
			},
			expectedResult: ctrl.Result{
				RequeueAfter: requeueAfter,
			},
		},
		{
			desc: "should delete secret from tenant namespace",
			initObjs: []client.Object{
				secretDeletedWithLabelAndFinalizer.DeepCopy(),
				tenantNamespace.DeepCopy(),
				otherNamespace.DeepCopy(),
				replicatedSecret.DeepCopy(),
				unrelatedSecret.DeepCopy(),
			},
			validate: func(t *testing.T, ctx context.Context, c client.Client) error {
				secrets := &corev1.SecretList{}
				if err := c.List(ctx, secrets); err != nil {
					return err
				}
				assert.Len(t, secrets.Items, 2)

				secretDeleted := &corev1.Secret{}
				if err := c.Get(ctx, client.ObjectKeyFromObject(secretDeletedWithLabelAndFinalizer), secretDeleted); err != nil {
					return err
				}
				// Finalizer will stay until next reconciliation
				assert.Equal(t, secretDeleted.Finalizers, []string{finalizerOrphan})
				return nil
			},
			expectedResult: ctrl.Result{
				RequeueAfter: requeueAfterError,
			},
		},
		{
			desc: "should remove finalizer when no replicated secrets left",
			initObjs: []client.Object{
				secretDeletedWithLabelAndFinalizer.DeepCopy(),
				tenantNamespace.DeepCopy(),
				otherNamespace.DeepCopy(),
				unrelatedSecret.DeepCopy(),
			},
			validate: func(t *testing.T, ctx context.Context, c client.Client) error {
				secrets := &corev1.SecretList{}
				if err := c.List(ctx, secrets); err != nil {
					return err
				}
				// only "unrelatedSecret" should remain
				assert.Len(t, secrets.Items, 1)
				return nil
			},
			expectedResult: ctrl.Result{
				RequeueAfter: 0,
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			c := fake.NewClientBuilder().WithObjects(tC.initObjs...).WithInterceptorFuncs(tC.interceptorFuncs).Build()
			ctx := newContext()
			req := newRequest(tC.initObjs[0])

			sr := &SecretReconciler{
				Client: c,
				Scheme: c.Scheme(),
			}
			result, err := sr.Reconcile(ctx, req)

			assert.Equal(t, tC.expectedResult, result)
			assert.Equal(t, tC.expectedErr, err)

			if tC.validate != nil {
				assert.NoError(t, tC.validate(t, ctx, c))
			}
		})
	}
}

func Test_buildFilterPredicate(t *testing.T) {
	testCases := []struct {
		desc                string
		objectOther, object client.Object
		expected            bool
	}{
		{
			desc:        "should reconcile secret with label",
			object:      secretWithLabel,
			objectOther: secretWithoutLabel,
			expected:    true,
		},
		{
			desc:     "should not reconcile secret without label",
			object:   secretWithoutLabel,
			expected: false,
		},
		{
			desc:        "should reconcile secret with finalizer",
			object:      secretWithFinalizer,
			objectOther: secretWithoutLabel,
			expected:    true,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			if tC.objectOther == nil {
				tC.objectOther = tC.object
			}

			sr := &SecretReconciler{}
			predicate := sr.buildFilterPredicate()
			assert.Equal(t, tC.expected, predicate.Create(event.CreateEvent{Object: tC.object}))
			assert.Equal(t, tC.expected, predicate.Delete(event.DeleteEvent{Object: tC.object}))
			assert.Equal(t, tC.expected, predicate.Generic(event.GenericEvent{Object: tC.object}))
			assert.Equal(t, tC.expected, predicate.Update(event.UpdateEvent{ObjectOld: tC.object, ObjectNew: tC.objectOther}))
			assert.Equal(t, tC.expected, predicate.Update(event.UpdateEvent{ObjectOld: tC.objectOther, ObjectNew: tC.object}))
		})
	}
}
