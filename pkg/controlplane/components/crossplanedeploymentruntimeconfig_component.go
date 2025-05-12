package components

import (
	"context"
	"fmt"
	"strings"

	crossplanev1beta1 "github.com/crossplane/crossplane/apis/pkg/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/crossplane"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/object"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
)

var _ object.ObjectComponent = &CrossplaneDeploymentRuntimeConfig{}
var _ TargetComponent = &CrossplaneDeploymentRuntimeConfig{}
var _ object.OrphanedObjectsDetector = &CrossplaneDeploymentRuntimeConfig{}
var _ juggler.StatusVisibility = &CrossplaneDeploymentRuntimeConfig{}

type CrossplaneDeploymentRuntimeConfig struct {
	Name    string
	Enabled bool
}

// BuildObjectToReconcile implements object.ObjectComponent.
func (c *CrossplaneDeploymentRuntimeConfig) BuildObjectToReconcile(ctx context.Context) (client.Object,
	types.NamespacedName, error) {
	nsn := types.NamespacedName{
		Name:      c.Name,
		Namespace: "", // we can leave this empty, because DeploymentRungimeConfigs are cluster scoped
	}
	return &crossplanev1beta1.DeploymentRuntimeConfig{}, nsn, nil
}

// ReconcileObject implements object.ObjectComponent.
func (c *CrossplaneDeploymentRuntimeConfig) ReconcileObject(ctx context.Context, obj client.Object) error {
	cdrc := obj.(*crossplanev1beta1.DeploymentRuntimeConfig)

	cdrc.Spec.ServiceAccountTemplate = applyServiceAccountTemplateDefaults(cdrc.Spec.ServiceAccountTemplate, c.Name)

	// We need to set defaults here on creation, because otherwise we would need
	// to relax our policy to allow for them to be modified by the end-user, which
	// we don't want to do. See
	// https://docs.crossplane.io/latest/concepts/providers/#runtime-configuration
	// for reference.
	cdrc.Spec.DeploymentTemplate = applyDeploymentTemplateDefaults(cdrc.Spec.DeploymentTemplate)

	return nil
}

// IsObjectHealthy implements object.ObjectComponent.
func (c *CrossplaneDeploymentRuntimeConfig) IsObjectHealthy(obj client.Object) juggler.ResourceHealthiness {
	return juggler.ResourceHealthiness{
		Healthy: true,
		Message: "DeploymentRuntimeConfig applied",
	}
}

// IsInstallable implements Component.
func (c *CrossplaneDeploymentRuntimeConfig) IsInstallable(ctx context.Context) (bool, error) {
	// CrossplaneDeploymentRuntimeConfigs are always installable
	return true, nil
}

// GetName implements Component.
func (c *CrossplaneDeploymentRuntimeConfig) GetName() string {
	return "DeploymentRuntimeConfig" + formatProviderName(c.Name)
}

// GetDependencies implements Component.
func (c *CrossplaneDeploymentRuntimeConfig) GetDependencies() []juggler.Component {
	return []juggler.Component{&Crossplane{}}
}

// IsEnabled implements Component.
func (c *CrossplaneDeploymentRuntimeConfig) IsEnabled() bool {
	return c.Enabled
}

// Hooks implements Component.
func (*CrossplaneDeploymentRuntimeConfig) Hooks() juggler.ComponentHooks {
	return juggler.ComponentHooks{
		PreInstall: crossplane.CheckIfPolicyIsInstalled(crossplane.Providers),
		PreUpdate:  crossplane.CheckIfPolicyIsInstalled(crossplane.Providers),
	}
}

// GetNamespace implements TargetComponent.
func (c *CrossplaneDeploymentRuntimeConfig) GetNamespace() string {
	// CrossplaneDeploymentRuntimeConfig is cluster-scoped.
	return ""
}

// OrphanDetectorContext implements object.OrphanedObjectsDetector.
func (c *CrossplaneDeploymentRuntimeConfig) OrphanDetectorContext() object.DetectorContext {
	return object.DetectorContext{
		ListType: &crossplanev1beta1.DeploymentRuntimeConfigList{},
		FilterCriteria: object.FilterCriteria{
			utils.IsManaged(),
			object.HasComponentLabel(),
		},
		ConvertFunc: func(list client.ObjectList) []juggler.Component {
			cdrcs := []juggler.Component{}
			for _, role := range (list.(*crossplanev1beta1.DeploymentRuntimeConfigList)).Items {
				name, _ := strings.CutPrefix(role.Name, fmt.Sprintf("%s:", v1beta1.GroupVersion.Group))
				cdrcs = append(cdrcs, &CrossplaneDeploymentRuntimeConfig{Name: name})
			}
			return cdrcs
		},
		SameFunc: func(configured, detected juggler.Component) bool {
			configuredCR := configured.(*CrossplaneDeploymentRuntimeConfig)
			detectedCR := detected.(*CrossplaneDeploymentRuntimeConfig)
			return strings.EqualFold(configuredCR.Name, detectedCR.Name)
		},
	}
}

// IsStatusInternal implements juggler.StatusVisibility
func (c *CrossplaneDeploymentRuntimeConfig) IsStatusInternal() bool {
	return true
}

// applyDeploymentTemplateDefaults makes sure that all required fields are set
// on a DeploymentTemplate. Specifically this means that we need to set
// template.spec, template.spec.selector, and template.spec.template.containers
// to be empty, but not nil values. This is because spec.DeploymentTemplate is
// validated against the k8s deployment type, which requires these fields to be
// set (even though they will  be properly filled with values by the crossplane
// controller later).
func applyDeploymentTemplateDefaults(in *crossplanev1beta1.DeploymentTemplate) *crossplanev1beta1.DeploymentTemplate {
	out := &crossplanev1beta1.DeploymentTemplate{}
	if in != nil {
		*out = *in
	}

	if out.Spec == nil {
		out.Spec = &appsv1.DeploymentSpec{}
	}

	if out.Spec.Selector == nil {
		// we don't need to set this to anything meaningful, crossplane is going to
		// do this for us when creating the deployment. However we still need to set
		// it to non-nil so the validation on the DeploymentRuntimeConfig does not
		// fail
		out.Spec.Selector = &metav1.LabelSelector{}
	}

	if len(out.Spec.Template.Spec.Containers) == 0 {
		out.Spec.Template.Spec.Containers = []v1.Container{
			{
				// we need to hardcode this, since crossplane keeps it private.
				// It is the default, and only, name a provider container can have
				// nolint:lll
				// https://github.com/crossplane/crossplane/blob/bee7c095b2c8b2e157a3154cbb85bfc8e54ace6f/internal/controller/pkg/revision/runtime.go#L35
				Name: "package-runtime",
				Args: []string{},
			},
		}
	}

	return out
}

// applyServiceAccountTemplateDefaults makes sure that the name of a
// serviceAccount matches the name of the passed in provider
func applyServiceAccountTemplateDefaults(
	in *crossplanev1beta1.ServiceAccountTemplate,
	providerName string,
) *crossplanev1beta1.ServiceAccountTemplate {
	out := &crossplanev1beta1.ServiceAccountTemplate{}
	if in != nil {
		*out = *in
	}

	if out.Metadata == nil {
		out.Metadata = &crossplanev1beta1.ObjectMeta{}
	}

	if out.Metadata.Name == nil {
		out.Metadata.Name = &providerName
	}

	return out
}
