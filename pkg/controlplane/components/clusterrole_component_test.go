//nolint:dupl
package components

import (
	"testing"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var (
	clusterRoleHealthy = &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-healthy",
		},
	}
	clusterRoleUnhealthy = &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-unhealthy",
			DeletionTimestamp: ptr.To(metav1.Now()),
		},
	}
	clusterRoleA = &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-a",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{corev1.GroupName},
				Resources: []string{"secrets"},
				Verbs:     VerbsAdmin,
			},
		},
	}
)

func Test_ClusterRole(t *testing.T) {
	testCases := []struct {
		desc            string
		enabled         bool
		name            string
		rules           []rbacv1.PolicyRule
		validationFuncs []validationFunc
	}{
		{
			desc:    "should be disabled",
			enabled: false,
			name:    "Admin",
			validationFuncs: []validationFunc{
				hasName("ClusterRoleAdmin"),
				isEnabled(false),
			},
		},
		{
			desc:    "should be enabled",
			enabled: true,
			name:    "Admin",
			rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{corev1.GroupName},
					Resources: []string{"secrets"},
					Verbs:     VerbsAdmin,
				},
			},
			validationFuncs: []validationFunc{
				hasName("ClusterRoleAdmin"),
				isEnabled(true),
				isAllowed(true),
				hasDependencies(0),
				hasNoHooks(),
				isTargetComponent(
					hasNamespace(""),
				),
				isObjectComponent(
					objectIsType(&rbacv1.ClusterRole{}),
					canCheckHealthiness(clusterRoleUnhealthy, juggler.ResourceHealthiness{
						Healthy: false,
					}),
					canCheckHealthiness(clusterRoleHealthy, juggler.ResourceHealthiness{
						Healthy: true,
					}),
					canBuildAndReconcile(nil),
					implementsOrphanedObjectsDetector(
						listTypeIs(&rbacv1.ClusterRoleList{}),
						hasFilterCriteria(2),
						canConvert(&rbacv1.ClusterRoleList{Items: []rbacv1.ClusterRole{*clusterRoleA}}, 1),
						canCheckSame(&ClusterRole{Name: "A"}, &ClusterRole{Name: "A"}, true),
						canCheckSame(&ClusterRole{Name: "A"}, &ClusterRole{Name: "B"}, false),
					),
				),
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ctx := newContext(nil, nil)
			c := &ClusterRole{Name: tC.name, Rules: tC.rules, Enabled: tC.enabled}
			for _, vfn := range tC.validationFuncs {
				vfn(t, ctx, c)
			}
		})
	}
}
