package crossplane

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	crossplanev1 "github.com/crossplane/crossplane/apis/pkg/v1"
	crossplanev1beta "github.com/crossplane/crossplane/apis/pkg/v1beta1"
	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	arv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PackageType string

const (
	Providers      PackageType = "providers"
	Configurations PackageType = "configurations"
	Functions      PackageType = "functions"
)

var (
	PackageTypes = []PackageType{
		Providers,
		Configurations,
		Functions,
	}

	ErrPolicyNotFound           = errors.New("policy is not installed yet")
	ErrFailedToGetPolicy        = errors.New("failed to get policy")
	ErrPolicyUnhealthy          = errors.New("policy is unhealthy")
	ErrPolicyBindingNotFound    = errors.New("policy binding is not installed yet")
	ErrFailedToGetPolicyBinding = errors.New("failed to get policy binding")
	ErrPolicyBindingUnhealthy   = errors.New("policy binding is unhealthy")
)

//nolint:lll
func ReconcilePolicy(pt PackageType, policy *arv1.ValidatingAdmissionPolicy) error {
	// Reconcile ValidatingAdmissionPolicy
	policy.Spec.FailurePolicy = ptr.To(arv1.Fail)
	policy.Spec.ParamKind = &arv1.ParamKind{
		APIVersion: v1beta1.GroupVersion.String(),
		Kind:       "CrossplanePackageRestriction",
	}
	policy.Spec.MatchConstraints = &arv1.MatchResources{
		ResourceRules: []arv1.NamedRuleWithOperations{
			{
				RuleWithOperations: arv1.RuleWithOperations{
					Operations: []arv1.OperationType{
						arv1.Create,
						arv1.Update,
					},
					Rule: arv1.Rule{
						APIGroups:   []string{crossplanev1.Group},
						APIVersions: []string{"*"},
						Resources:   []string{strings.ToLower(string(pt))},
					},
				},
			},
		},
	}
	policy.Spec.Variables = []arv1.Variable{
		{
			Name: "allPackagesAllowed",
			// Check if "packages" contains an asterisk.
			Expression: fmt.Sprintf("'*' in params.spec.%s.packages", pt),
		},
		{
			Name: "allRegistriesAllowed",
			// Check if "registries" contains an asterisk.
			Expression: fmt.Sprintf("'*' in params.spec.%s.registries", pt),
		},
		{
			Name: "packageAllowed",
			// Check if "packages" contains "package" (including version => only specific version allowed) or "packages" contains "package" (excluding version => any version allowed).
			Expression: fmt.Sprintf("object.spec.package in params.spec.%s.packages || object.spec.package.split(':', 2)[0] in params.spec.%s.packages", pt, pt),
		},
		{
			Name: "registryAllowed",
			// Check if "package" starts with any of the allowed registries.
			Expression: fmt.Sprintf("params.spec.%s.registries.exists(registry, object.spec.package.startsWith(registry + '/'))", pt),
		},
	}
	policy.Spec.Validations = []arv1.Validation{
		{
			Expression: "variables.allPackagesAllowed || variables.allRegistriesAllowed || variables.packageAllowed || variables.registryAllowed",
			Message:    "Package or registry not allowed",
			Reason:     ptr.To(metav1.StatusReasonForbidden),
		},
	}

	// All good
	return nil
}

func ReconcilePolicyBinding(cprName, policyName string, binding *arv1.ValidatingAdmissionPolicyBinding) error {
	// Reconcile ValidatingAdmissionPolicyBinding
	binding.Spec.PolicyName = policyName
	binding.Spec.ParamRef = &arv1.ParamRef{
		Name:                    cprName,
		ParameterNotFoundAction: ptr.To(arv1.DenyAction),
	}
	binding.Spec.ValidationActions = []arv1.ValidationAction{arv1.Deny}

	// All good
	return nil
}

func IsPolicyHealthy(policy *arv1.ValidatingAdmissionPolicy) bool {
	// Generation and ObservedGeneration must be the same and greater than zero.
	return policy.Status.ObservedGeneration > 0 &&
		policy.Generation > 0 &&
		policy.Generation == policy.Status.ObservedGeneration &&
		policy.DeletionTimestamp == nil
}

func IsPolicyBindingHealthy(binding *arv1.ValidatingAdmissionPolicyBinding) bool {
	// ValidatingAdmissionPolicyBinding has no status field.
	return binding.Generation > 0 &&
		binding.DeletionTimestamp == nil
}

func GetPolicyName(pt PackageType) string {
	return fmt.Sprintf("restrict-crossplane-%s", strings.ToLower(string(pt)))
}

// CheckIfPolicyIsInstalled checks if the policy and binding for the given PackageType is installed.
// If not, we cannot safely install or update the package and the hook will fail.
func CheckIfPolicyIsInstalled(pt PackageType) func(ctx context.Context, c client.Client) error {
	key := types.NamespacedName{
		Name: GetPolicyName(pt),
	}

	return func(ctx context.Context, c client.Client) error {
		// Check if policy exists.
		policy := &arv1.ValidatingAdmissionPolicy{}
		err := c.Get(ctx, key, policy)
		if apierrors.IsNotFound(err) {
			return ErrPolicyNotFound
		}
		if err != nil {
			// some unknown error occurred.
			return errors.Join(ErrFailedToGetPolicy, err)
		}
		if !IsPolicyHealthy(policy) {
			return ErrPolicyUnhealthy
		}

		// Check if policy binding exists.
		policyBinding := &arv1.ValidatingAdmissionPolicyBinding{}
		err = c.Get(ctx, key, policyBinding)
		if apierrors.IsNotFound(err) {
			return ErrPolicyBindingNotFound
		}
		if err != nil {
			// some unknown error occurred.
			return errors.Join(ErrFailedToGetPolicyBinding, err)
		}
		if !IsPolicyBindingHealthy(policyBinding) {
			return ErrPolicyBindingUnhealthy
		}

		// All good.
		return nil
	}
}

func ReconcileDeploymentConfigRuntimeProtectionPolicy(avp *arv1.ValidatingAdmissionPolicy) {
	serviceAccountName := fmt.Sprintf(
		"system:serviceaccount:%s:%s",
		os.Getenv("POD_NAMESPACE"),
		os.Getenv("POD_SERVICE_ACCOUNT"),
	)

	avp.Spec = arv1.ValidatingAdmissionPolicySpec{
		FailurePolicy: ptr.To(arv1.Fail),
		MatchConditions: []arv1.MatchCondition{
			{
				Name:       "exclude-co-operator",
				Expression: fmt.Sprintf("request.userInfo.username != %q", serviceAccountName),
			},
		},
		MatchConstraints: &arv1.MatchResources{
			ResourceRules: []arv1.NamedRuleWithOperations{
				{
					RuleWithOperations: arv1.RuleWithOperations{
						Operations: []arv1.OperationType{
							arv1.Update,
						},
						Rule: arv1.Rule{
							APIGroups:   []string{crossplanev1beta.Group},
							APIVersions: []string{crossplanev1beta.Version},
							Resources:   []string{"deploymentruntimeconfigs"}, // need lower-case plural here, so we need to hardcode it
						},
					},
				},
			},
		},
		Validations: getDeploymentRuntimeConfigExpressionBuilder().Build(),
	}
}

func ReconcileDeploymentConfigRuntimeProtectionPolicyBinding(policyName string,
	avpd *arv1.ValidatingAdmissionPolicyBinding) {

	avpd.ObjectMeta.Name = policyName

	avpd.Spec = arv1.ValidatingAdmissionPolicyBindingSpec{
		ValidationActions: []arv1.ValidationAction{arv1.Deny},
		PolicyName:        policyName,
	}

}

// groupedFields represents a group of fields which all share
// the same prefix. E.g ".spec.serviceAccountTemplate.metadata" and ".spec.serviceAccountTemplate.annotations"
type groupedFields struct {
	// PrefixPath defines the path which all fields should be prefixed by.
	// Separate by dots. E.g. ".spec.serviceAccountTemplate.metadata"
	// Should not include "object." or "oldObject."
	PrefixPath string

	// Fields describes the names of the fields which should be used
	Fields []string
}

// oldObjectCompareValidationBuilder builds expressions to compare all fields with their old
// version of that field. Syntactically, it automatically generates expressions with the
// following syntax for you:
// oldObject.?<path>.?<field> == object.?<path>.?<field>
type oldObjectCompareValidationBuilder struct {
	// GroupedFields describes for which groups of fields the validations should be build
	GroupedFields []groupedFields
	// AdditionalValidations allows to add additional custom validations. These will be
	// added to Build() output before the generated validations
	AdditionalValidations []arv1.Validation
}

func (om *oldObjectCompareValidationBuilder) Build() []arv1.Validation {
	ret := []arv1.Validation{}

	if om.AdditionalValidations != nil {
		ret = append(ret, om.AdditionalValidations...)
	}

	for _, expr := range om.GroupedFields {
		// turn any paths into optional paths
		prefix := strings.ReplaceAll(expr.PrefixPath, ".", ".?")

		for _, field := range expr.Fields {
			exp := fmt.Sprintf("oldObject.?%s.?%s == object.?%s.?%s", prefix, field, prefix, field)
			ret = append(ret, arv1.Validation{
				Expression: exp,
				Message:    fmt.Sprintf("field \"%s.%s\" is not allowed to be modified", expr.PrefixPath, field),
			})
		}
	}
	return ret
}

// String prints out all expressions as a nice list, so they can directly be copied
// into a ValidatingAdmissionPolicy manifest for local testing.
func (om *oldObjectCompareValidationBuilder) String() string {
	var sb strings.Builder
	vs := om.Build()

	for _, v := range vs {
		sb.WriteString(fmt.Sprintf("    - expression: %q\n", v.Expression))
	}
	return sb.String()
}

func getDeploymentRuntimeConfigExpressionBuilder() *oldObjectCompareValidationBuilder {
	b := &oldObjectCompareValidationBuilder{}

	b.AdditionalValidations = []arv1.Validation{
		// exit early if another container has been added.
		// This also allows us to use .spec.containers[0] below as we can guarantee that there
		// is only one container.
		{
			Expression: "size(object.spec.deploymentTemplate.spec.template.spec.containers) == 1",
			Message:    "The number of containers is not allowed to be changed",
		},
	}

	b.GroupedFields = append(b.GroupedFields, groupedFields{
		PrefixPath: "spec.serviceAccountTemplate.metadata",
		Fields: []string{
			"annotations",
			"labels",
			// "name", name is allowed to be edited
		},
	})

	b.GroupedFields = append(b.GroupedFields, groupedFields{
		PrefixPath: "spec.deploymentTemplate.spec",
		Fields: []string{
			"selector",
			"replicas",
			"minReadySeconds",
			"strategy",
			"revisionHistoryLimit",
			"progressDeadlineSeconds",
			"paused",
		},
	})

	b.GroupedFields = append(b.GroupedFields, groupedFields{
		PrefixPath: "spec.deploymentTemplate.spec.template",
		Fields: []string{
			"metadata",
		},
	})

	b.GroupedFields = append(b.GroupedFields, groupedFields{
		PrefixPath: "spec.deploymentTemplate.spec.template.spec",
		Fields: []string{
			// "args", args is allowed to be edited
			"volumes",
			"nodeSelector",
			"nodeName",
			"affinity",
			"tolerations",
			"schedulerName",
			"runtimeClassName",
			"priorityClassName",
			"priority",
			"preemptionPolicy",
			"topologySpreadConstraints",
			"overhead",
			"restartPolicy",
			"terminationGracePeriodSeconds",
			"activeDeadlineSeconds",
			"readinessGates",
			"hostname",
			"setHostnameAsFQDN",
			"subdomain",
			"hostAliases",
			"dnsConfig",
			"dnsPolicy",
			"hostNetwork",
			"hostPID",
			"hostIPC",
			"shareProcessNamespace",
			"serviceAccountName",
			"automountServiceAccountToken",
			"securityContext",
		},
	})

	b.GroupedFields = append(b.GroupedFields, groupedFields{
		PrefixPath: "spec.deploymentTemplate.spec.template.spec.containers[0]",
		Fields: []string{
			"command",
			"env",
			"envFrom",
			"image",
			"imagePullPolicy",
			"lifecycle",
			"livenessProbe",
			"name",
			"ports",
			"readinessProbe",
			"resizePolicy",
			"resources",
			"restartPolicy",
			"securityContext",
			"startupProbe",
			"stdin",
			"stdinOnce",
			"terminationMessagePath",
			"terminationMessagePolicy",
			"tty",
			"volumeDevices",
			"volumeMounts",
			"workingDir",
		},
	})

	return b
}
