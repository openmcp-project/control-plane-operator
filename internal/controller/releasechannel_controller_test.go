package controller

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1beta1 "github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/internal/schemes"
)

const localOCMRegistryTestDataPath = "../../test/testdata/ocm_registry.tgz"

func Test_ReleaseChannelReconciler_Reconcile(t *testing.T) {
	ocmTestRegistry, err := os.ReadFile(localOCMRegistryTestDataPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	testCases := []struct {
		desc           string
		initObjs       []client.Object
		expectedResult ctrl.Result
		expectedErr    error
		validate       func(t *testing.T, ctx context.Context, c client.Client) error
	}{
		{
			desc: "should return error when OcmRegistryUrl and OcmRegistrySecretRef are not set",
			initObjs: []client.Object{
				&corev1beta1.ReleaseChannel{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-releasechannel",
					},
				},
			},
			expectedResult: ctrl.Result{
				RequeueAfter: requeueAfterError,
			},
			expectedErr: errors.New("either 'OcmRegistryUrl' or 'OcmRegistrySecretRef & OcmRegistrySecretKey' must be set"),
		}, {
			desc: "should return error, when secret is not found",
			initObjs: []client.Object{
				&corev1beta1.ReleaseChannel{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-releasechannel",
					},
					Spec: corev1beta1.ReleaseChannelSpec{
						OcmRegistryUrl: "https://some.url",
						PullSecretRef: corev1.SecretReference{
							Name: "some-secret",
						},
					},
				},
			},
			expectedResult: ctrl.Result{
				RequeueAfter: requeueAfterError,
			},
			expectedErr: errors.New("unable to fetch Secret"),
		}, {
			desc: "Cant get username from secret",
			initObjs: []client.Object{
				&corev1beta1.ReleaseChannel{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-releasechannel",
					},
					Spec: corev1beta1.ReleaseChannelSpec{
						OcmRegistryUrl: "https://some.url",
						PullSecretRef: corev1.SecretReference{
							Name: "some-secret",
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-secret",
					},
				},
			},
			expectedResult: ctrl.Result{
				RequeueAfter: requeueAfterError,
			},
			expectedErr: errors.New("Failed to get username from secret"),
		}, {
			desc: "Cant get password from secret",
			initObjs: []client.Object{
				&corev1beta1.ReleaseChannel{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-releasechannel",
					},
					Spec: corev1beta1.ReleaseChannelSpec{
						OcmRegistryUrl: "https://some.url",
						PullSecretRef: corev1.SecretReference{
							Name: "some-secret",
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-secret",
					},
					Data: map[string][]byte{
						"username": []byte("some-username"),
					},
				},
			},
			expectedResult: ctrl.Result{
				RequeueAfter: requeueAfterError,
			},
			expectedErr: errors.New("Failed to get password from secret"),
		}, {
			desc: "should return error when unable to get components from remote OCM",
			initObjs: []client.Object{
				&corev1beta1.ReleaseChannel{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-releasechannel",
					},
					Spec: corev1beta1.ReleaseChannelSpec{
						OcmRegistryUrl: "https://some.url",
						PullSecretRef: corev1.SecretReference{
							Name: "some-secret",
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-secret",
					},
					Data: map[string][]byte{
						"username": []byte("some-username"),
						"password": []byte("some-password"),
					},
				},
			},
			expectedResult: ctrl.Result{
				RequeueAfter: requeueAfterError,
			},
			expectedErr: errors.New("unable to get components from remote OCM"),
		},
		{
			desc: "should return an error, if OcmRegistrySecretRef is set but secret is not found",
			initObjs: []client.Object{
				&corev1beta1.ReleaseChannel{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-releasechannel",
					},
					Spec: corev1beta1.ReleaseChannelSpec{
						OcmRegistrySecretRef: corev1.SecretReference{
							Name: "some-secret",
						},
						OcmRegistrySecretKey: "registry.tar.gz",
					},
				},
			},
			expectedResult: ctrl.Result{
				RequeueAfter: requeueAfterError,
			},
			expectedErr: errors.New("unable to fetch Secret"),
		},
		{
			desc: "should return an error, if OcmRegistrySecretRef is set but key is not found in secret",
			initObjs: []client.Object{
				&corev1beta1.ReleaseChannel{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-releasechannel",
					},
					Spec: corev1beta1.ReleaseChannelSpec{
						OcmRegistrySecretRef: corev1.SecretReference{
							Name: "some-secret",
						},
						OcmRegistrySecretKey: "registry.tar.gz",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-secret",
					},
				},
			},
			expectedResult: ctrl.Result{
				RequeueAfter: requeueAfterError,
			},
			expectedErr: errors.New("key registry.tar.gz not found in secret some-secret"),
		}, {
			desc: "should return an error, if unable to get components from local OCM",
			initObjs: []client.Object{
				&corev1beta1.ReleaseChannel{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-releasechannel",
					},
					Spec: corev1beta1.ReleaseChannelSpec{
						OcmRegistrySecretRef: corev1.SecretReference{
							Name: "some-secret",
						},
						OcmRegistrySecretKey: "registry.tar.gz",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-secret",
					},
					Data: map[string][]byte{
						"registry.tar.gz": []byte("some-data"),
					},
				},
			},
			expectedResult: ctrl.Result{
				RequeueAfter: requeueAfterError,
			},
			expectedErr: errors.New("unable to get components from local OCM"),
		}, {
			desc: "success case",
			initObjs: []client.Object{
				&corev1beta1.ReleaseChannel{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-releasechannel",
					},
					Spec: corev1beta1.ReleaseChannelSpec{
						OcmRegistrySecretRef: corev1.SecretReference{
							Name: "some-secret",
						},
						OcmRegistrySecretKey: "registry.tar.gz",
						Interval:             metav1.Duration{Duration: 15 * time.Minute},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-secret",
					},
					Data: map[string][]byte{
						"registry.tar.gz": ocmTestRegistry,
					},
				},
			},
			expectedResult: ctrl.Result{
				RequeueAfter: time.Minute * 15,
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			c := fake.NewClientBuilder().WithObjects(tC.initObjs...).WithStatusSubresource(tC.initObjs[0]).WithScheme(schemes.Local).Build()
			ctx := newContext()
			req := newRequest(tC.initObjs[0])

			sr := &ReleaseChannelReconciler{
				Client: c,
				Scheme: c.Scheme(),
			}
			result, err := sr.Reconcile(ctx, req)

			assert.Equal(t, tC.expectedResult, result)
			if tC.expectedErr != nil {
				assert.Errorf(t, err, tC.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if tC.validate != nil {
				assert.NoError(t, tC.validate(t, ctx, c))
			}
		})
	}
}
