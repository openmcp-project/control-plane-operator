package clusterroles

import (
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/components"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

func RegisterAsComponents(jug *juggler.Juggler, cpComponents []juggler.Component, enabled bool) {
	policyRules := components.AggregatePolicyRules(cpComponents)
	jug.RegisterComponent(&components.ClusterRole{
		Name:    "Admin",
		Rules:   policyRules.Admin,
		Enabled: enabled,
		// Workaround to prevent users from losing access to resources
		// until other ControlPlane components are uninstalled.
		KeepInstalled: true,
	})
	jug.RegisterComponent(&components.ClusterRole{
		Name:    "View",
		Rules:   policyRules.View,
		Enabled: enabled,
		// Workaround to prevent users from losing access to resources
		// until other ControlPlane components are uninstalled.
		KeepInstalled: true,
	})
}
