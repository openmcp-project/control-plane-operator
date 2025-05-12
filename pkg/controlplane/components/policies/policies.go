package policies

import (
	"context"

	arv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/components"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/crossplane"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

const (
	CrossplanePackageRestrictionName = "default"
)

func RegisterAsComponents(jug *juggler.Juggler, sourceClient client.Client, enabled bool) error {
	cpr := &components.GenericObjectComponent{
		NamespacedName: types.NamespacedName{
			Name: CrossplanePackageRestrictionName,
		},
		Enabled: enabled,
		Type:    &v1beta1.CrossplanePackageRestriction{},
		ReconcileObjectFunc: func(ctx context.Context, obj client.Object) error {
			sourceCPR := &v1beta1.CrossplanePackageRestriction{}
			if err := sourceClient.Get(ctx, types.NamespacedName{
				Name: CrossplanePackageRestrictionName,
			}, sourceCPR); err != nil {
				return err
			}

			objCPR := obj.(*v1beta1.CrossplanePackageRestriction)
			objCPR.Spec = sourceCPR.Spec

			return nil
		},
		IsObjectHealthyFunc: func(obj client.Object) juggler.ResourceHealthiness {
			return juggler.ResourceHealthiness{
				// CrossplanePackageRestriction has no status field.
				Healthy: obj.GetDeletionTimestamp() == nil,
			}
		},
	}
	jug.RegisterComponent(cpr)

	for _, pt := range crossplane.PackageTypes {
		policy := &components.GenericObjectComponent{
			NamespacedName: types.NamespacedName{
				Name: crossplane.GetPolicyName(pt),
			},
			Enabled:          enabled,
			Type:             &arv1.ValidatingAdmissionPolicy{},
			TypeNameOverride: "Policy",
			ReconcileObjectFunc: func(ctx context.Context, obj client.Object) error {
				vap := obj.(*arv1.ValidatingAdmissionPolicy)
				return crossplane.ReconcilePolicy(pt, vap)
			},
			IsObjectHealthyFunc: func(obj client.Object) juggler.ResourceHealthiness {
				vap := obj.(*arv1.ValidatingAdmissionPolicy)
				return juggler.ResourceHealthiness{
					Healthy: crossplane.IsPolicyHealthy(vap),
				}
			},
		}
		jug.RegisterComponent(policy)

		policyBinding := &components.GenericObjectComponent{
			NamespacedName: types.NamespacedName{
				Name: crossplane.GetPolicyName(pt),
			},
			Enabled:          enabled,
			Type:             &arv1.ValidatingAdmissionPolicyBinding{},
			TypeNameOverride: "PolicyBinding",
			ReconcileObjectFunc: func(ctx context.Context, obj client.Object) error {
				vapb := obj.(*arv1.ValidatingAdmissionPolicyBinding)
				return crossplane.ReconcilePolicyBinding(cpr.Name, policy.Name, vapb)
			},
			IsObjectHealthyFunc: func(obj client.Object) juggler.ResourceHealthiness {
				vapb := obj.(*arv1.ValidatingAdmissionPolicyBinding)
				return juggler.ResourceHealthiness{
					Healthy: crossplane.IsPolicyBindingHealthy(vapb),
				}
			},
		}
		jug.RegisterComponent(policyBinding)
	}
	return nil
}

// RegisterDeploymentRuntimeConfigProtection adds a ValidatingAdmissionPolicy,
// which only allows certain fields of DeploymentRuntimeConfig to be edited and
// which excludes the ServiceAccount of the controlplane-operator
func RegisterDeploymentRuntimeConfigProtection(jug *juggler.Juggler, sourceClient client.Client, enabled bool) error {
	policy := &components.GenericObjectComponent{
		NamespacedName: types.NamespacedName{
			Name: "restrict-crossplane-deploymentruntimeconfig",
		},
		Enabled:          enabled,
		Type:             &arv1.ValidatingAdmissionPolicy{},
		TypeNameOverride: "Policy",
		ReconcileObjectFunc: func(ctx context.Context, obj client.Object) error {
			vap := obj.(*arv1.ValidatingAdmissionPolicy)
			crossplane.ReconcileDeploymentConfigRuntimeProtectionPolicy(vap)
			return nil
		},
		IsObjectHealthyFunc: func(obj client.Object) juggler.ResourceHealthiness {
			vap := obj.(*arv1.ValidatingAdmissionPolicy)
			return juggler.ResourceHealthiness{
				Healthy: crossplane.IsPolicyHealthy(vap),
			}
		},
	}
	jug.RegisterComponent(policy)

	policyBinding := &components.GenericObjectComponent{
		NamespacedName: types.NamespacedName{
			Name: "restrict-crossplane-deploymentruntimeconfig",
		},
		Enabled:          enabled,
		Type:             &arv1.ValidatingAdmissionPolicyBinding{},
		TypeNameOverride: "PolicyBinding",
		ReconcileObjectFunc: func(ctx context.Context, obj client.Object) error {
			vapd := obj.(*arv1.ValidatingAdmissionPolicyBinding)
			crossplane.ReconcileDeploymentConfigRuntimeProtectionPolicyBinding(policy.Name, vapd)
			return nil
		},
		IsObjectHealthyFunc: func(obj client.Object) juggler.ResourceHealthiness {
			vapb := obj.(*arv1.ValidatingAdmissionPolicyBinding)
			return juggler.ResourceHealthiness{
				Healthy: crossplane.IsPolicyBindingHealthy(vapb),
			}
		},
	}
	jug.RegisterComponent(policyBinding)

	return nil
}
