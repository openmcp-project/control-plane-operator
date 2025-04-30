package controller

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	testutils "github.com/openmcp-project/control-plane-operator/test/utils"

	"github.com/openmcp-project/controller-utils/pkg/clientconfig"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	corev1beta1 "github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/cmd/options"
	"github.com/openmcp-project/control-plane-operator/internal/schemes"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/kubeconfiggen"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/secretresolver"
	envtestutil "github.com/openmcp-project/control-plane-operator/pkg/utils/envtest"
)

func TestMain(m *testing.M) {
	if err := envtestutil.Install(); err != nil {
		log.Fatalln(err)
	}
	os.Exit(m.Run())
}

var (
	ocmSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "artifactory-readonly-docker-openmcp",
			Namespace: "co-system",
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{}`),
		},
	}
	coSystemNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "co-system",
		},
	}
)

func TestControlPlaneReconciler_Reconcile(t *testing.T) {
	testCases := []struct {
		desc             string
		initObjs         []client.Object
		interceptorFuncs interceptor.Funcs
		expectedResult   ctrl.Result
		expectedErr      error
		validate         func(t *testing.T, ctx context.Context, c client.Client) error
	}{
		{
			desc: "error - resource not found",
			initObjs: []client.Object{
				&corev1beta1.ControlPlane{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-controlplane",
					},
				},
				coSystemNamespace,
				ocmSecret,
			},
			interceptorFuncs: interceptor.Funcs{Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return apierrors.NewNotFound(schema.GroupResource{Group: corev1beta1.GroupVersion.Group, Resource: "controlplanes"}, key.Name)
			}},
			expectedResult: ctrl.Result{},
			expectedErr:    nil,
		},
		{
			desc: "error - unable to fetch ControlPlane",
			initObjs: []client.Object{
				&corev1beta1.ControlPlane{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-controlplane",
					},
				},
				coSystemNamespace,
				ocmSecret,
			},
			interceptorFuncs: interceptor.Funcs{Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return errTest
			}},
			expectedResult: ctrl.Result{},
			expectedErr:    errTest,
		},
		{
			desc: "ensure Namespace creation failed",
			initObjs: []client.Object{
				&corev1beta1.ControlPlane{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-controlplane",
					},
				},
				coSystemNamespace,
				ocmSecret,
			},
			interceptorFuncs: interceptor.Funcs{
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if _, ok := obj.(*corev1.Namespace); ok {
						return errTest
					}
					return nil
				},
			},
			expectedResult: ctrl.Result{},
			expectedErr:    errors.Join(errFailedToCreateCPNamespace, errTest),
		},
		{
			desc: "failed to apply Flux RBAC for target cluster",
			initObjs: []client.Object{
				&corev1beta1.ControlPlane{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-controlplane",
					},
					Spec: corev1beta1.ControlPlaneSpec{
						Target: corev1beta1.Target{},
					},
				},
				coSystemNamespace,
				ocmSecret,
			},
			expectedResult: ctrl.Result{},
			expectedErr:    errors.Join(errFailedToApplyFluxRBAC, errors.New("name or namespace in service account reference must not be empty")),
		},
		{
			desc: "failed to ensure Flux Kubeconfig for target cluster - Get Secret error (unexpected)",
			initObjs: []client.Object{
				&corev1beta1.ControlPlane{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-controlplane",
					},
					Spec: corev1beta1.ControlPlaneSpec{
						Target: corev1beta1.Target{
							FluxServiceAccount: corev1beta1.ServiceAccountReference{
								Name:      "flux-deployer",
								Namespace: "default",
							},
						},
					},
				},
				coSystemNamespace,
				ocmSecret,
			},
			interceptorFuncs: interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if s, ok := obj.(*corev1.Secret); ok && s.Name == "flux-kubeconfig" {
						return errTest
					}
					return client.Get(ctx, key, obj, opts...)
				},
			},
			expectedResult: ctrl.Result{},
			expectedErr:    errors.Join(errFailedToEnsureFluxKubeconfig, errTest),
		},
		{
			desc: "failed to ensure Finalizer for ControlPlane resource - Update error (unexpected)",
			initObjs: []client.Object{
				&corev1beta1.ControlPlane{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-controlplane",
					},
					Spec: corev1beta1.ControlPlaneSpec{
						Target: corev1beta1.Target{
							FluxServiceAccount: corev1beta1.ServiceAccountReference{
								Name:      "flux-deployer",
								Namespace: "default",
							},
						},
					},
				},
				coSystemNamespace,
				ocmSecret,
			},
			interceptorFuncs: interceptor.Funcs{
				Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					if _, ok := obj.(*corev1beta1.ControlPlane); ok { // can not update ControlPlane to add finalizer
						return errTest
					}
					return nil
				},
			},
			expectedResult: ctrl.Result{},
			expectedErr:    errTest,
		},
		{
			desc: "successful ControlPlane reconciliation - Crossplane installed",
			initObjs: []client.Object{
				&corev1beta1.ControlPlane{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-controlplane",
					},
					Spec: corev1beta1.ControlPlaneSpec{
						Target: corev1beta1.Target{
							FluxServiceAccount: corev1beta1.ServiceAccountReference{
								Name:      "flux-deployer",
								Namespace: "default",
							},
						},
						ComponentsConfig: corev1beta1.ComponentsConfig{
							Crossplane: &corev1beta1.CrossplaneConfig{
								Version: "1.15.0",
							},
						},
					},
				},
				&corev1beta1.ReleaseChannel{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-releasechannel",
					},
					Status: corev1beta1.ReleaseChannelStatus{Components: []corev1beta1.Component{
						{Name: "crossplane", Versions: []corev1beta1.ComponentVersion{{Version: "1.15.0", HelmRepo: "https://charts.crossplane.io/stable", HelmChart: "crossplane"}}},
						{Name: "provider-helm", Versions: []corev1beta1.ComponentVersion{{Version: "0.19.0", DockerRef: "xpkg.upbound.io/crossplane-contrib/provider-helm:v0.19.0"}}},
					}},
				},
				coSystemNamespace,
				ocmSecret,
			},
			validate: func(t *testing.T, ctx context.Context, c client.Client) error {
				cp := &corev1beta1.ControlPlane{}
				if err := c.Get(ctx, client.ObjectKey{Name: "some-controlplane"}, cp); err != nil {
					return err
				}
				expectedComponentsEnabled := 10
				if options.IsDeploymentRuntimeConfigProtectionEnabled() {
					expectedComponentsEnabled += 2
				}
				assert.Equal(t, expectedComponentsEnabled, cp.Status.ComponentsEnabled)
				cond := meta.FindStatusCondition(cp.Status.Conditions, "CrossplaneReady")
				assert.Equal(t, metav1.ConditionFalse, cond.Status)
				assert.Equal(t, "Installed", cond.Reason)
				return nil
			},
			expectedResult: ctrl.Result{RequeueAfter: time.Second * 30},
			expectedErr:    nil,
		},
		{
			desc: "successful ControlPlane deletion - Step 1",
			initObjs: []client.Object{
				&corev1beta1.ControlPlane{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "some-controlplane",
						DeletionTimestamp: ptr.To(metav1.Now()),
						Finalizers: []string{
							corev1beta1.Finalizer,
						},
					},
					Spec: corev1beta1.ControlPlaneSpec{
						Target: corev1beta1.Target{
							FluxServiceAccount: corev1beta1.ServiceAccountReference{
								Name:      "flux-deployer",
								Namespace: "default",
							},
						},
						ComponentsConfig: corev1beta1.ComponentsConfig{
							Crossplane: &corev1beta1.CrossplaneConfig{
								Version: "1.15.0",
							},
						},
					},
				},
				coSystemNamespace,
				ocmSecret,
			},
			validate: func(t *testing.T, ctx context.Context, c client.Client) error {
				cp := &corev1beta1.ControlPlane{}
				if err := c.Get(ctx, client.ObjectKey{Name: "some-controlplane"}, cp); err != nil {
					return err
				}
				assert.Equal(t, 0, cp.Status.ComponentsEnabled)
				cond := meta.FindStatusCondition(cp.Status.Conditions, "clusterRoleAdminReady")
				assert.Equal(t, metav1.ConditionFalse, cond.Status)
				assert.Equal(t, "Uninstalled", cond.Reason)
				return nil
			},
			expectedResult: ctrl.Result{RequeueAfter: 5 * time.Second},
			expectedErr:    nil,
		},
		{
			desc: "successful ControlPlane deletion - Step 2",
			initObjs: []client.Object{
				&corev1beta1.ControlPlane{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "some-controlplane",
						DeletionTimestamp: ptr.To(metav1.Now()),
						Finalizers: []string{
							corev1beta1.Finalizer,
						},
					},
					Spec: corev1beta1.ControlPlaneSpec{
						Target: corev1beta1.Target{
							FluxServiceAccount: corev1beta1.ServiceAccountReference{
								Name:      "flux-deployer",
								Namespace: "default",
							},
						},
						ComponentsConfig: corev1beta1.ComponentsConfig{
							Crossplane: &corev1beta1.CrossplaneConfig{
								Version: "1.15.0",
							},
						},
					},
				},
				coSystemNamespace,
				ocmSecret,
			},
			validate: func(t *testing.T, ctx context.Context, c client.Client) error {
				cp := &corev1beta1.ControlPlane{}
				err := c.Get(ctx, client.ObjectKey{Name: "some-controlplane"}, cp)
				assert.True(t, apierrors.IsNotFound(err))
				return nil
			},
			expectedResult: ctrl.Result{},
			expectedErr:    nil,
		},
	}

	testEnv := &envtest.Environment{}
	testCfg, err := testEnv.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		assert.NoError(t, testEnv.Stop())
	}()

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			c := fake.NewClientBuilder().WithObjects(tC.initObjs...).WithInterceptorFuncs(tC.interceptorFuncs).WithStatusSubresource(tC.initObjs[0]).WithScheme(schemes.Local).Build()
			ctx := newContext()

			testSecretResolver := secretresolver.NewFluxSecretResolver(c)
			_ = testSecretResolver.Start(ctx)

			cpr := &ControlPlaneReconciler{
				Client:             c,
				Scheme:             c.Scheme(),
				Kubeconfiggen:      &kubeconfiggen.Default{},
				FluxSecretResolver: testSecretResolver,
				WebhookMiddleware:  types.NamespacedName{},
				ReconcilePeriod:    time.Second * 30,
				Recorder:           record.NewFakeRecorder(100),
				RemoteConfigBuilder: func(target corev1beta1.Target) (*rest.Config, clientconfig.ReloadFunc, error) {
					return testCfg, nil, nil
				},
			}
			assert.NoError(t, testutils.SetEnvironmentVariableForLocalOCMTar(testutils.LocalOCMRepositoryPathValid))
			req := newRequest(tC.initObjs[0])
			result, err := cpr.Reconcile(ctx, req)

			assert.Equal(t, tC.expectedResult, result)
			assert.Equal(t, tC.expectedErr, err)

			if tC.validate != nil {
				assert.NoError(t, tC.validate(t, ctx, c))
			}
		})
	}
}
