package targetrbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	errSANameOrNamespaceEmpty = errors.New("name or namespace in service account reference must not be empty")
)

// checkServiceAccountReference checks if the ServiceAccountReference is valid
func checkServiceAccountReference(svcAccRef v1beta1.ServiceAccountReference) error {
	if svcAccRef.Name == "" || svcAccRef.Namespace == "" {
		return errSANameOrNamespaceEmpty
	}
	return nil
}

// serviceAccount creates a ServiceAccount for the given ServiceAccountReference
func serviceAccount(svcAccRef v1beta1.ServiceAccountReference) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcAccRef.Name,
			Namespace: svcAccRef.Namespace,
		},
		AutomountServiceAccountToken: ptr.To(false),
	}
}

// clusterRoleBinding creates a ClusterRoleBinding for the given ServiceAccountReference
func clusterRoleBinding(svcAccRef v1beta1.ServiceAccountReference) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s:%s", svcAccRef.Namespace, svcAccRef.Name),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      svcAccRef.Name,
				Namespace: svcAccRef.Namespace,
			},
		},
	}
}

// Apply creates or updates the ServiceAccount and ClusterRoleBinding for the given ServiceAccountReference
func Apply(ctx context.Context, c client.Client, svcAccRef v1beta1.ServiceAccountReference) error {
	if err := checkServiceAccountReference(svcAccRef); err != nil {
		return err
	}
	sa := serviceAccount(svcAccRef)
	_, err := controllerutil.CreateOrUpdate(ctx, c, sa, func() error {
		desired := serviceAccount(svcAccRef)

		sa.AutomountServiceAccountToken = desired.AutomountServiceAccountToken

		utils.SetManagedBy(sa)
		return nil
	})
	if err != nil {
		return err
	}

	crb := clusterRoleBinding(svcAccRef)
	_, err = controllerutil.CreateOrUpdate(ctx, c, crb, func() error {
		desired := clusterRoleBinding(svcAccRef)
		crb.RoleRef = desired.RoleRef
		crb.Subjects = desired.Subjects

		utils.SetManagedBy(crb)
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func Delete(ctx context.Context, c client.Client) error {
	if err := c.DeleteAllOf(ctx, &rbacv1.ClusterRoleBinding{}, utils.IsManaged()); err != nil {
		return err
	}

	// DeleteAllOf does not work across namespaces
	// https://github.com/kubernetes-sigs/controller-runtime/issues/1842#issuecomment-1244857876
	svcAccs := &corev1.ServiceAccountList{}
	if err := c.List(ctx, svcAccs, utils.IsManaged()); err != nil {
		return err
	}
	for _, sa := range svcAccs.Items {
		if err := c.Delete(ctx, &sa); err != nil {
			return err
		}
	}

	return nil
}
