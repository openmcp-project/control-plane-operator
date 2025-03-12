package kubeconfiggen

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	envtestutil "github.com/openmcp-project/control-plane-operator/pkg/utils/envtest"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	testSvcAcc = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeconfiggen-test",
			Namespace: metav1.NamespaceDefault,
		},
	}
)

func TestMain(m *testing.M) {
	if err := envtestutil.Install(); err != nil {
		log.Fatalln(err)
	}
	os.Exit(m.Run())
}

func Test_ForServiceAccount(t *testing.T) {
	testCases := []struct {
		desc                  string
		svcAccRef             v1beta1.ServiceAccountReference
		tryConnect            bool
		writeCACertToTempFile bool
		hostOverride          *string
	}{
		{
			desc:       "should generate kubeconfig and be able to connect to API server",
			tryConnect: true,
			svcAccRef: v1beta1.ServiceAccountReference{
				Name:      "kubeconfiggen-test",
				Namespace: metav1.NamespaceDefault,
			},
		},
		{
			desc: "should generate kubeconfig with host override",
			svcAccRef: v1beta1.ServiceAccountReference{
				Name:      "kubeconfiggen-test",
				Namespace: metav1.NamespaceDefault,
				Overrides: v1beta1.KubeconfigOverrides{
					Host: "http://custom-host.example.com",
				},
			},
			hostOverride: ptr.To("http://custom-host.example.com"),
		},
		{
			desc: "should generate kubeconfig with host override",
			svcAccRef: v1beta1.ServiceAccountReference{
				Name:      "kubeconfiggen-test",
				Namespace: metav1.NamespaceDefault,
			},
			writeCACertToTempFile: true,
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

	ctx := context.Background()
	if err := setupTestServiceAcc(ctx, testCfg); err != nil {
		t.Fatal(err)
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			if tC.writeCACertToTempFile {
				cleanupFn, err := writeCACertToTempFile(testCfg)
				assert.NoError(t, err)
				defer func() {
					assert.NoError(t, cleanupFn())
				}()
			}

			d := &Default{}
			apiConfig, expTime, err := d.ForServiceAccount(ctx, testCfg, tC.svcAccRef, 1*time.Hour)

			assert.NoError(t, err)
			// check if exp time is in 55-65 min (1h +/- 10min)
			assert.WithinRange(t, *expTime, time.Now().Add(55*time.Minute), time.Now().Add(65*time.Minute))

			saClientConfig := clientcmd.NewDefaultClientConfig(*apiConfig, nil)
			saRestConfig, err := saClientConfig.ClientConfig()
			assert.NoError(t, err)

			if tC.tryConnect {
				testClient, err := client.New(saRestConfig, client.Options{Scheme: clientgoscheme.Scheme})
				assert.NoError(t, err)

				// try to list namespaces, expect authorization error
				err = testClient.List(ctx, &corev1.NamespaceList{})
				//nolint:lll
				assert.ErrorContains(t, err, "User \"system:serviceaccount:default:kubeconfiggen-test\" cannot list resource \"namespaces\"")
			}

			if tC.hostOverride != nil {
				assert.Equal(t, *tC.hostOverride, saRestConfig.Host)
			} else {
				assert.Equal(t, testCfg.Host, saRestConfig.Host)
			}
		})
	}
}

func Test_ForServiceAccount_Validation(t *testing.T) {
	testCases := []struct {
		desc       string
		cfg        *rest.Config
		svcAccRef  v1beta1.ServiceAccountReference
		expiration time.Duration
		expected   error
	}{
		{
			desc:       "should fail with invalid service account name",
			cfg:        &rest.Config{},
			svcAccRef:  v1beta1.ServiceAccountReference{Name: "", Namespace: "some-namespace"},
			expiration: time.Hour,
			expected:   ErrSANameOrNamespaceEmpty,
		},
		{
			desc:       "should fail with invalid service account namespace",
			cfg:        &rest.Config{},
			svcAccRef:  v1beta1.ServiceAccountReference{Name: "some-name", Namespace: ""},
			expiration: time.Hour,
			expected:   ErrSANameOrNamespaceEmpty,
		},
		{
			desc:       "should fail with invalid rest config",
			cfg:        nil,
			svcAccRef:  v1beta1.ServiceAccountReference{Name: "some-name", Namespace: "some-namespace"},
			expiration: time.Hour,
			expected:   ErrRestConfigNil,
		},
		{
			desc:       "should fail with invalid expiration - negative",
			cfg:        &rest.Config{},
			svcAccRef:  v1beta1.ServiceAccountReference{Name: "some-name", Namespace: "some-namespace"},
			expiration: -time.Hour,
			expected:   ErrExpirationInvalid,
		},
		{
			desc:       "should fail with invalid expiration - too short",
			cfg:        &rest.Config{},
			svcAccRef:  v1beta1.ServiceAccountReference{Name: "some-name", Namespace: "some-namespace"},
			expiration: time.Minute,
			expected:   ErrExpirationInvalid,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			d := &Default{}
			_, _, actual := d.ForServiceAccount(context.Background(), tC.cfg, tC.svcAccRef, tC.expiration)
			assert.ErrorIs(t, actual, tC.expected)
		})
	}
}

func setupTestServiceAcc(ctx context.Context, cfg *rest.Config) error {
	c, err := client.New(cfg, client.Options{Scheme: clientgoscheme.Scheme})
	if err != nil {
		return err
	}

	return c.Create(ctx, testSvcAcc)
}

func writeCACertToTempFile(cfg *rest.Config) (func() error, error) {
	file, err := os.CreateTemp("", "kubeconfiggen-test")
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(file, bytes.NewReader(cfg.CAData)); err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}

	cfg.CAFile = file.Name()
	cfg.CAData = nil

	return func() error {
		return os.Remove(file.Name())
	}, nil
}
