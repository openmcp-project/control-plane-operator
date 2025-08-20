package components

import (
	"context"
	"fmt"
	"strings"

	crossplanev1 "github.com/crossplane/crossplane/apis/pkg/v1"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/crossplane"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/object"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
	"github.com/openmcp-project/control-plane-operator/pkg/utils/rcontext"
)

var _ object.ObjectComponent = &CrossplaneProvider{}
var _ object.OrphanedObjectsDetector = &CrossplaneProvider{}
var _ TargetComponent = &CrossplaneProvider{}

type CrossplaneProvider struct {
	Config      *v1beta1.CrossplaneProviderConfig
	Enabled     bool
	PullSecrets []corev1.LocalObjectReference
}

// BuildObjectToReconcile implements object.ObjectComponent.
func (c *CrossplaneProvider) BuildObjectToReconcile(ctx context.Context) (client.Object, types.NamespacedName, error) {
	obj, key := crossplane.EmptyFromConfig(*c.Config)
	return obj, key, nil
}

// ReconcileObject implements object.ObjectComponent.
func (c *CrossplaneProvider) ReconcileObject(ctx context.Context, obj client.Object) error {
	versionResolveFn := rcontext.VersionResolver(ctx)
	copy := *c.Config

	// When uninstalling a provider, we don't need to resolve the version.
	if c.IsEnabled() {
		// Resolve package and version by provider name
		comp, err := versionResolveFn(crossplane.ProviderNameForProviderConfig(c.Config), c.Config.Version)
		if err != nil {
			return err
		}

		copy.Package = comp.DockerRef
		copy.Version = comp.Version
	}

	objProvider := obj.(*crossplanev1.Provider)
	return crossplane.ReconcileProvider(objProvider, copy)
}

// OrphanDetectorContext implements object.OrphanedObjectsDetector.
func (*CrossplaneProvider) OrphanDetectorContext() object.DetectorContext {
	return object.DetectorContext{
		ListType: &crossplanev1.ProviderList{},
		FilterCriteria: object.FilterCriteria{
			utils.IsManaged(),
			utils.HasComponentLabel(),
		},
		ConvertFunc: func(list client.ObjectList) []juggler.Component {
			providers := []juggler.Component{}
			for _, provider := range (list.(*crossplanev1.ProviderList)).Items {
				// since we only need the name for the SameFunc, there is no need to copy the whole object
				cp := &CrossplaneProvider{Config: &v1beta1.CrossplaneProviderConfig{
					Name: crossplane.TrimProviderPrefix(provider.Name),
				}}
				providers = append(providers, cp)
			}
			return providers
		},
		SameFunc: func(configured, detected juggler.Component) bool {
			configuredP := configured.(*CrossplaneProvider)
			detectedP := detected.(*CrossplaneProvider)
			return crossplane.TrimProviderPrefix(configuredP.Config.Name) == detectedP.Config.Name
		},
	}
}

// IsObjectHealthy implements object.ObjectComponent.
func (c *CrossplaneProvider) IsObjectHealthy(obj client.Object) juggler.ResourceHealthiness {
	provider := obj.(*crossplanev1.Provider)

	installed := provider.GetCondition(crossplanev1.TypeInstalled)
	if installed.Status != corev1.ConditionTrue {
		return juggler.ResourceHealthiness{
			Healthy: false,
			Message: fmt.Sprintf("Provider installation is pending (%s). %s", installed.Reason, installed.Message),
		}
	}

	healthy := provider.GetCondition(crossplanev1.TypeHealthy)
	return juggler.ResourceHealthiness{
		Healthy: healthy.Status == corev1.ConditionTrue,
		Message: fmt.Sprintf("%s: %s", healthy.Reason, healthy.Message),
	}
}

// GetNamespace implements TargetComponent.
func (c *CrossplaneProvider) GetNamespace() string {
	return CrossplaneNamespace
}

// IsInstallable implements Component.
func (c *CrossplaneProvider) IsInstallable(ctx context.Context) (bool, error) {
	rfn := rcontext.VersionResolver(ctx)
	if _, err := rfn(crossplane.ProviderNameForProviderConfig(c.Config), c.Config.Version); err != nil {
		return false, err
	}
	return true, nil
}

// GetName implements Component.
func (c *CrossplaneProvider) GetName() string {
	return formatProviderName(c.Config.Name)
}

// GetDependencies implements Component.
func (c *CrossplaneProvider) GetDependencies() []juggler.Component {
	return []juggler.Component{&Crossplane{}}
}

// IsEnabled implements Component.
func (c *CrossplaneProvider) IsEnabled() bool {
	return c.Enabled
}

// Hooks implements Component.
func (*CrossplaneProvider) Hooks() juggler.ComponentHooks {
	return juggler.ComponentHooks{
		PreInstall: crossplane.CheckIfPolicyIsInstalled(crossplane.Providers),
		PreUpdate:  crossplane.CheckIfPolicyIsInstalled(crossplane.Providers),
	}
}

func formatProviderName(providerName string) string {
	providerName = crossplane.AddProviderPrefix(providerName)
	parts := strings.Split(providerName, "-")
	for i, part := range parts {
		parts[i] = cases.Title(language.English).String(part)
	}
	return strings.Join(parts, "")
}
