package components

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/object"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
)

var _ object.ObjectComponent = &ClusterRole{}
var _ object.OrphanedObjectsDetector = &ClusterRole{}
var _ TargetComponent = &ClusterRole{}
var _ juggler.KeepOnUninstall = &ClusterRole{}
var _ juggler.StatusVisibility = &ClusterRole{}

type ClusterRole struct {
	Name          string
	Rules         []rbacv1.PolicyRule
	Enabled       bool
	KeepInstalled bool
}

// KeepOnUninstall implements juggler.KeepOnUninstall.
func (c *ClusterRole) KeepOnUninstall() bool {
	return c.KeepInstalled
}

// BuildObjectToReconcile implements object.ObjectComponent.
func (c *ClusterRole) BuildObjectToReconcile(ctx context.Context) (client.Object, types.NamespacedName, error) {
	return &rbacv1.ClusterRole{}, types.NamespacedName{
		Name: fmt.Sprintf("%s:%s", v1beta1.GroupVersion.Group, strings.ToLower(c.Name)),
	}, nil
}

// ReconcileObject implements object.ObjectComponent.
func (c *ClusterRole) ReconcileObject(ctx context.Context, obj client.Object) error {
	objCR := obj.(*rbacv1.ClusterRole)

	aggregateLabel := fmt.Sprintf("%s/aggregate-to-%s", v1beta1.GroupVersion.Group, strings.ToLower(c.Name))
	metav1.SetMetaDataLabel(&objCR.ObjectMeta, aggregateLabel, "true")

	objCR.Rules = c.Rules
	return nil
}

// OrphanDetectorContext implements object.OrphanedObjectsDetector.
func (*ClusterRole) OrphanDetectorContext() object.DetectorContext {
	return object.DetectorContext{
		ListType: &rbacv1.ClusterRoleList{},
		FilterCriteria: object.FilterCriteria{
			utils.IsManaged(),
			object.HasComponentLabel(),
		},
		ConvertFunc: func(list client.ObjectList) []juggler.Component {
			clusterRoles := []juggler.Component{}
			for _, role := range (list.(*rbacv1.ClusterRoleList)).Items {
				name, _ := strings.CutPrefix(role.Name, fmt.Sprintf("%s:", v1beta1.GroupVersion.Group))
				clusterRoles = append(clusterRoles, &ClusterRole{Name: name, Rules: role.Rules})
			}
			return clusterRoles
		},
		SameFunc: func(configured, detected juggler.Component) bool {
			configuredCR := configured.(*ClusterRole)
			detectedCR := detected.(*ClusterRole)
			return strings.EqualFold(configuredCR.Name, detectedCR.Name)
		},
	}
}

// GetDependencies implements object.ObjectComponent.
func (c *ClusterRole) GetDependencies() []juggler.Component {
	return []juggler.Component{}
}

// GetName implements object.ObjectComponent.
func (c *ClusterRole) GetName() string {
	name := cases.Title(language.English).String(c.Name)
	return "ClusterRole" + name
}

// Hooks implements object.ObjectComponent.
func (c *ClusterRole) Hooks() juggler.ComponentHooks {
	return juggler.ComponentHooks{}
}

func (c *ClusterRole) IsInstallable(_ context.Context) (bool, error) {
	return true, nil
}

// IsEnabled implements object.ObjectComponent.
func (c *ClusterRole) IsEnabled() bool {
	return c.Enabled
}

// IsObjectHealthy implements object.ObjectComponent.
func (c *ClusterRole) IsObjectHealthy(obj client.Object) juggler.ResourceHealthiness {
	return juggler.ResourceHealthiness{
		// ClusterRole has no status field.
		Healthy: obj.GetDeletionTimestamp() == nil,
	}
}

// GetNamespace implements TargetComponent.
func (c *ClusterRole) GetNamespace() string {
	// ClusterRole is cluster-scoped.
	return ""
}

// IsStatusInternal implements StatusVisibility interface.
func (c *ClusterRole) IsStatusInternal() bool {
	return true
}
