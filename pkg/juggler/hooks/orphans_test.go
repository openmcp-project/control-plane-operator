package hooks

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func Test_PreventOrphanedResources(t *testing.T) {
	testCases := []struct {
		desc             string
		gvks             []schema.GroupVersionKind
		initObjs         []client.Object
		interceptorFuncs interceptor.Funcs
		expectedErr      *string
	}{
		{
			desc: "should not return error when list of GVKs is empty",
		},
		{
			desc: "should not return error when CRD is not installed",
			gvks: []schema.GroupVersionKind{
				{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"},
				{Group: "cert-manager.io", Version: "v1", Kind: "Issuer"},
				{Group: "cert-manager.io", Version: "v1", Kind: "ClusterIssuer"},
			},
			interceptorFuncs: interceptor.Funcs{
				List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					return &apiutil.ErrResourceDiscoveryFailed{
						list.GetObjectKind().GroupVersionKind().GroupVersion(): apierrors.NewNotFound(schema.GroupResource{}, ""),
					}
				},
			},
		},
		{
			desc: "should not return error when no resource of type exists",
			gvks: []schema.GroupVersionKind{
				{Group: "", Version: "v1", Kind: "Secret"},
			},
		},
		{
			desc: "should return error when API server returns unknown error",
			gvks: []schema.GroupVersionKind{
				{Group: "", Version: "v1", Kind: "Secret"},
			},
			interceptorFuncs: interceptor.Funcs{
				List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					return errors.New("some unknown error")
				},
			},
			expectedErr: ptr.To("some unknown error"),
		},
		{
			desc: "should return error when resource of type exists",
			gvks: []schema.GroupVersionKind{
				{Group: "", Version: "v1", Kind: "Secret"},
			},
			initObjs: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-secret",
						Namespace: metav1.NamespaceDefault,
					},
				},
			},
			expectedErr: ptr.To("cannot uninstall because there is a least one object of /v1, Kind=Secret remaining"),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			c := fake.NewClientBuilder().WithObjects(tC.initObjs...).WithInterceptorFuncs(tC.interceptorFuncs).Build()
			fn := PreventOrphanedResources(tC.gvks)
			err := fn(context.Background(), c)

			if tC.expectedErr != nil {
				assert.EqualError(t, err, *tC.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
