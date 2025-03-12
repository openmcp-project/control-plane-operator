package kubeconfiggen

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	corev1beta1 "github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/internal/schemes"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrSANameOrNamespaceEmpty = errors.New("name or namespace in service account reference must not be empty")
	ErrRestConfigNil          = errors.New("rest config must not be nil")
	ErrExpirationInvalid      = errors.New("must not specify a duration less than 10 minutes")
)

type Generator interface {
	ForServiceAccount(
		ctx context.Context,
		cfg *rest.Config,
		svcAccRef corev1beta1.ServiceAccountReference,
		expiration time.Duration,
	) (*clientcmdapi.Config, *time.Time, error)
}

type Default struct{}

func (*Default) ForServiceAccount(
	ctx context.Context,
	cfg *rest.Config,
	svcAccRef corev1beta1.ServiceAccountReference,
	expiration time.Duration,
) (*clientcmdapi.Config, *time.Time, error) {
	if svcAccRef.Name == "" || svcAccRef.Namespace == "" {
		return nil, nil, ErrSANameOrNamespaceEmpty
	}
	if cfg == nil {
		return nil, nil, ErrRestConfigNil
	}
	if expiration < 10*time.Minute {
		return nil, nil, ErrExpirationInvalid
	}

	client, err := client.New(cfg, client.Options{Scheme: schemes.Local})
	if err != nil {
		return nil, nil, err
	}

	sa := &corev1.ServiceAccount{}
	if err := client.Get(ctx, types.NamespacedName{Name: svcAccRef.Name, Namespace: svcAccRef.Namespace}, sa); err != nil {
		return nil, nil, err
	}

	req := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: ptr.To(int64(expiration.Seconds())),
		},
	}
	if err := client.SubResource("token").Create(ctx, sa, req); err != nil {
		return nil, nil, err
	}

	host := cfg.Host
	if svcAccRef.Overrides.Host != "" {
		host = svcAccRef.Overrides.Host
	}

	ctxName := fmt.Sprintf("%s--%s", sa.Name, sa.Namespace)
	kubeconfig := clientcmdapi.NewConfig()
	kubeconfig.CurrentContext = ctxName
	kubeconfig.Clusters[ctxName] = &clientcmdapi.Cluster{
		Server:                   host,
		CertificateAuthorityData: cfg.CAData,
	}
	kubeconfig.AuthInfos[ctxName] = &clientcmdapi.AuthInfo{
		Token: req.Status.Token,
	}
	kubeconfig.Contexts[ctxName] = &clientcmdapi.Context{
		Cluster:  ctxName,
		AuthInfo: ctxName,
	}

	if cfg.CAFile != "" {
		caBytes, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, nil, err
		}
		kubeconfig.Clusters[ctxName].CertificateAuthorityData = caBytes
	}

	return kubeconfig, &req.Status.ExpirationTimestamp.Time, nil
}
