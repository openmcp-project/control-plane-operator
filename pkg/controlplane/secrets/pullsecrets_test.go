package secrets

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/openmcp-project/control-plane-operator/pkg/constants"
)

var (
	errFake = errors.New("some error")
)

func Test_AvailablePullSecrets(t *testing.T) {
	testCases := []struct {
		desc             string
		initObjs         []client.Object
		interceptorFuncs interceptor.Funcs
		expectedResult   []types.NamespacedName
		expectedErr      error
	}{
		{
			desc:           "should return an empty result when no secrets are present",
			expectedResult: []types.NamespacedName{},
		},
		{
			desc: "should return error when client fails to list secrets",
			interceptorFuncs: interceptor.Funcs{
				List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					return errFake
				},
			},
			expectedResult: nil,
			expectedErr:    errFake,
		},
		{
			desc: "should return only one secret",
			initObjs: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pull-secret",
						Namespace: corev1.NamespaceDefault,
						Labels: map[string]string{
							constants.LabelCopyToCP: "true",
						},
					},
					Type: corev1.SecretTypeDockerConfigJson,
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pull-secret-without-label",
						Namespace: corev1.NamespaceDefault,
					},
					Type: corev1.SecretTypeDockerConfigJson,
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-other-secret",
						Namespace: corev1.NamespaceDefault,
						Labels: map[string]string{
							constants.LabelCopyToCP: "true",
						},
					},
					Type: corev1.SecretTypeBasicAuth,
				},
			},
			expectedResult: []types.NamespacedName{
				{
					Name:      "pull-secret",
					Namespace: corev1.NamespaceDefault,
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithObjects(tC.initObjs...).WithInterceptorFuncs(tC.interceptorFuncs).Build()
			result, err := AvailablePullSecrets(context.Background(), fakeClient)
			assert.Equal(t, tC.expectedResult, result)
			assert.Equal(t, tC.expectedErr, err)
		})
	}
}
