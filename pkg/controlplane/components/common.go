package components

import (
	"errors"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

var ErrVersionResolverNotConfigured = errors.New("version resolver is not configured in context")

// TargetComponent is a component that should be installed on the Target (remote/workload) cluster.
type TargetComponent interface {
	GetNamespace() string
}

// PolicyRulesComponent is a component that provides rules which will be added to ClusterRoles.
type PolicyRulesComponent interface {
	GetPolicyRules() PolicyRules
}

type PolicyRules struct {
	Admin []rbacv1.PolicyRule
	View  []rbacv1.PolicyRule
}

var (
	VerbsAdmin  = []string{rbacv1.VerbAll}
	VerbsView   = []string{"get", "watch", "list"}
	VerbsModify = []string{"get", "watch", "list", "update", "patch"}
)

func AggregatePolicyRules(components []juggler.Component) PolicyRules {
	result := PolicyRules{}
	for _, c := range components {
		if prComp, ok := c.(PolicyRulesComponent); ok {
			compRules := prComp.GetPolicyRules()
			result.Admin = append(result.Admin, compRules.Admin...)
			result.View = append(result.View, compRules.View...)
		}
	}
	return result
}
