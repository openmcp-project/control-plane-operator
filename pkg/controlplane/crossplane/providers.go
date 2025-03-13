package crossplane

import (
	"fmt"
	"strings"

	crossplanev1 "github.com/crossplane/crossplane/apis/pkg/v1"
	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/types"
)

const (
	providerPrefix = "provider-"
)

// EmptyFromConfig converts a CrossplaneProviderConfig to a Crossplane Provider resource.
func EmptyFromConfig(c v1beta1.CrossplaneProviderConfig) (*crossplanev1.Provider, types.NamespacedName) {
	return &crossplanev1.Provider{}, types.NamespacedName{
		Name: ProviderNameForProviderConfig(&c),
	}
}

func ReconcileProvider(provider *crossplanev1.Provider, config v1beta1.CrossplaneProviderConfig) error {
	utils.SetManagedBy(provider)
	provider.Spec.Package = config.Package
	provider.Spec.PackagePullPolicy = config.PackagePullPolicy
	provider.Spec.PackagePullSecrets = config.PackagePullSecrets
	// Set the CrossplaneConfig to use a DeploymentRuntimeConfig with the same name.
	// The corresponding DeploymentRuntimeConfig is generated in crossplanedeploymentruntimeconfig_component.go
	provider.Spec.RuntimeConfigReference = &crossplanev1.RuntimeConfigReference{
		Name: DeploymentRuntimeNameForProviderConfig(&config)}
	return nil
}

func AddProviderPrefix(providerName string) string {
	if strings.HasPrefix(providerName, providerPrefix) {
		return providerName
	}
	return fmt.Sprintf("%s%s", providerPrefix, providerName)
}

func TrimProviderPrefix(providerName string) string {
	return strings.TrimPrefix(providerName, providerPrefix)
}

// ProviderNameForProviderConfig returns the name of a Provider crossplane manifest for a ProviderConfig.
// It consists of the name of the provider with a prefix.
func ProviderNameForProviderConfig(p *v1beta1.CrossplaneProviderConfig) string {
	return AddProviderPrefix(p.Name)
}

// DeploymentRuntimeNameForProviderConfig returns the name of a DeploymentRuntimeConfig manifest for a ProviderConfig.
// Currently the name is the same as the name of the provider.
func DeploymentRuntimeNameForProviderConfig(p *v1beta1.CrossplaneProviderConfig) string {
	return ProviderNameForProviderConfig(p)
}
