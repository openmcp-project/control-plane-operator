package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1beta1 "github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
)

const (
	keyKubeconfig = "kubeconfig"
	keyExpiration = "expiresAt"
)

func (r *ControlPlaneReconciler) ensureKubeconfig(ctx context.Context, remoteCfg *rest.Config, namespace string, secretName string, svcaccountRef corev1beta1.ServiceAccountReference) (*corev1.SecretReference, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	if expirationStr, ok := secret.Data[keyExpiration]; ok {
		expiration, err := time.Parse(time.RFC3339, string(expirationStr))
		if err != nil {
			return nil, err
		}

		remainingLifetime := time.Until(expiration)

		// check if token would expire before next planned reconciliation
		// or less than a third of the desired lifetime is left
		if remainingLifetime < r.ReconcilePeriod || remainingLifetime < r.FluxTokenLifetime/3 {
			// kubeconfig is still valid
			return &corev1.SecretReference{Name: secret.Name, Namespace: secret.Namespace}, nil
		}
	}

	kubeconfig, expiration, err := r.Kubeconfiggen.ForServiceAccount(ctx, remoteCfg, svcaccountRef, r.FluxTokenLifetime)
	if err != nil {
		return nil, err
	}

	kubeconfigBytes, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return nil, err
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		utils.SetManagedBy(secret)

		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}

		secret.Data[keyKubeconfig] = kubeconfigBytes
		secret.Data[keyExpiration] = []byte(expiration.Format(time.RFC3339))
		return nil
	})

	return &corev1.SecretReference{Name: secret.Name, Namespace: secret.Namespace}, err
}
