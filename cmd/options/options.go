package options

import "flag"

var (
	// enableDeploymentRuntimeConfigProtection is a flag to enable the
	// Crossplane DeploymentRuntimeConfig protection feature for all control planes.
	// When enabled, certain fields in the Crossplane DeploymentRuntimeConfig will be protected
	// from being modified by the user.
	// When disabled, users won't gain permissions to modify DeploymentRuntimeConfigs.
	// Default is disabled.
	enableDeploymentRuntimeConfigProtection = false
)

// SetEnableDeploymentRuntimeConfigProtection sets the enableDeploymentRuntimeConfigProtection flag.
func SetEnableDeploymentRuntimeConfigProtection(enable bool) {
	enableDeploymentRuntimeConfigProtection = enable
}

// IsDeploymentRuntimeConfigProtectionEnabled returns the value of the enableDeploymentRuntimeConfigProtection flag.
func IsDeploymentRuntimeConfigProtectionEnabled() bool {
	return enableDeploymentRuntimeConfigProtection
}

// AddOptions adds the options to the flag set.
func AddOptions() {
	flag.BoolVar(&enableDeploymentRuntimeConfigProtection, "enable-deploymentruntimeconfig-protection", false,
		"Enable DeploymentRuntimeConfig protection feature for all control planes.")
}
