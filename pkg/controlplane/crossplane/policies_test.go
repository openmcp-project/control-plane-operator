package crossplane

import (
	"bufio"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	arv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var (
	errFake = errors.New("fake")
)

func Test_ReconcilePolicy(t *testing.T) {
	policy := &arv1.ValidatingAdmissionPolicy{}
	err := ReconcilePolicy(Providers, policy)
	assert.NoError(t, err)

	assert.Len(t, policy.Spec.MatchConstraints.ResourceRules, 1)
	assert.Len(t, policy.Spec.Variables, 4)
	for _, v := range policy.Spec.Variables {
		assert.Contains(t, v.Expression, Providers)
		assert.NotContains(t, v.Expression, Functions)
		assert.NotContains(t, v.Expression, Configurations)
	}
	assert.Len(t, policy.Spec.Validations, 1)
	assert.Equal(t, policy.Spec.FailurePolicy, ptr.To(arv1.Fail))
}

func Test_ReconcilePolicyBinding(t *testing.T) {
	binding := &arv1.ValidatingAdmissionPolicyBinding{}
	err := ReconcilePolicyBinding("cpr1", "policy1", binding)
	assert.NoError(t, err)

	assert.Equal(t, "cpr1", binding.Spec.ParamRef.Name)
	assert.Equal(t, ptr.To(arv1.DenyAction), binding.Spec.ParamRef.ParameterNotFoundAction)
	assert.Equal(t, "policy1", binding.Spec.PolicyName)
	assert.Contains(t, binding.Spec.ValidationActions, arv1.Deny)
}

func Test_CheckIfPolicyIsInstalled(t *testing.T) {
	testCases := []struct {
		desc             string
		pt               PackageType
		expected         error
		interceptorFuncs interceptor.Funcs
		initObjs         []client.Object
	}{
		{
			desc:     "should fail when validating admission policy is missing",
			pt:       Providers,
			expected: ErrPolicyNotFound,
		},
		{
			desc:     "should fail when validating admission policy is unhealthy",
			pt:       Providers,
			expected: ErrPolicyUnhealthy,
			initObjs: []client.Object{
				&arv1.ValidatingAdmissionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: GetPolicyName(Providers),
					},
				},
			},
		},
		{
			desc:     "should fail when validating admission policy binding is missing",
			pt:       Providers,
			expected: ErrPolicyBindingNotFound,
			initObjs: []client.Object{
				&arv1.ValidatingAdmissionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:       GetPolicyName(Providers),
						Generation: 1,
					},
					Status: arv1.ValidatingAdmissionPolicyStatus{
						ObservedGeneration: 1,
					},
				},
			},
		},
		{
			desc:     "should fail when validating admission policy binding is unhealthy",
			pt:       Providers,
			expected: ErrPolicyBindingUnhealthy,
			initObjs: []client.Object{
				&arv1.ValidatingAdmissionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:       GetPolicyName(Providers),
						Generation: 1,
					},
					Status: arv1.ValidatingAdmissionPolicyStatus{
						ObservedGeneration: 1,
					},
				},
				&arv1.ValidatingAdmissionPolicyBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: GetPolicyName(Providers),
					},
				},
			},
		},
		{
			desc:     "should pass when validating admission policy and binding are present",
			pt:       Providers,
			expected: nil,
			initObjs: []client.Object{
				&arv1.ValidatingAdmissionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:       GetPolicyName(Providers),
						Generation: 1,
					},
					Status: arv1.ValidatingAdmissionPolicyStatus{
						ObservedGeneration: 1,
					},
				},
				&arv1.ValidatingAdmissionPolicyBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:       GetPolicyName(Providers),
						Generation: 1,
					},
				},
			},
		},
		{
			desc:     "should fail when fetching validating admission policy fails",
			pt:       Providers,
			expected: ErrFailedToGetPolicy,
			interceptorFuncs: interceptor.Funcs{
				//nolint:lll
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if _, ok := obj.(*arv1.ValidatingAdmissionPolicy); ok {
						return errFake
					}
					return client.Get(ctx, key, obj, opts...)
				},
			},
		},
		{
			desc:     "should fail when fetching validating admission policy binding fails",
			pt:       Providers,
			expected: ErrFailedToGetPolicyBinding,
			initObjs: []client.Object{
				&arv1.ValidatingAdmissionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:       GetPolicyName(Providers),
						Generation: 1,
					},
					Status: arv1.ValidatingAdmissionPolicyStatus{
						ObservedGeneration: 1,
					},
				},
			},
			interceptorFuncs: interceptor.Funcs{
				//nolint:lll
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if _, ok := obj.(*arv1.ValidatingAdmissionPolicyBinding); ok {
						return errFake
					}
					return client.Get(ctx, key, obj, opts...)
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			c := fake.NewClientBuilder().WithInterceptorFuncs(tC.interceptorFuncs).WithObjects(tC.initObjs...).Build()
			fn := CheckIfPolicyIsInstalled(tC.pt)
			assert.NotNil(t, fn)
			actual := fn(context.Background(), c)
			assert.ErrorIs(t, actual, tC.expected)
		})
	}
}

func TestOldObjectCompareValidationBuilder(t *testing.T) {
	testCases := []struct {
		desc   string
		gf     []groupedFields
		addVal []arv1.Validation
		expOut []string
	}{
		{
			desc: "no additional validations",
			gf: []groupedFields{
				{
					PrefixPath: "test",
					Fields:     []string{"myfield1", "myfield2"},
				},
				{
					PrefixPath: "test.nested",
					Fields:     []string{"myfield1", "myfield2"},
				},
			},
			expOut: []string{
				"    - expression: \"oldObject.?test.?myfield1 == object.?test.?myfield1\"",
				"    - expression: \"oldObject.?test.?myfield2 == object.?test.?myfield2\"",
				"    - expression: \"oldObject.?test.?nested.?myfield1 == object.?test.?nested.?myfield1\"",
				"    - expression: \"oldObject.?test.?nested.?myfield2 == object.?test.?nested.?myfield2\"",
			},
		},
		{
			desc: "place additional validations on top",
			gf: []groupedFields{
				{
					PrefixPath: "test",
					Fields:     []string{"myfield1", "myfield2"},
				},
				{
					PrefixPath: "test.nested",
					Fields:     []string{"myfield1", "myfield2"},
				},
			},
			addVal: []arv1.Validation{
				{
					Expression: "custom == customTest",
				},
			},
			expOut: []string{
				"    - expression: \"custom == customTest\"",
				"    - expression: \"oldObject.?test.?myfield1 == object.?test.?myfield1\"",
				"    - expression: \"oldObject.?test.?myfield2 == object.?test.?myfield2\"",
				"    - expression: \"oldObject.?test.?nested.?myfield1 == object.?test.?nested.?myfield1\"",
				"    - expression: \"oldObject.?test.?nested.?myfield2 == object.?test.?nested.?myfield2\"",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			b := &oldObjectCompareValidationBuilder{}
			b.GroupedFields = tc.gf
			b.AdditionalValidations = tc.addVal

			res := b.String()
			scanner := bufio.NewScanner(strings.NewReader(res))
			l := 0
			for scanner.Scan() {
				got := scanner.Text()
				exp := tc.expOut[l]
				if got != exp {
					t.Errorf("Err in line %d: exp %q, got %q", l, exp, got)
				}
				l++
			}
		})
	}
}

func TestGetDeploymentRuntimeConfigExpressionBuilder(t *testing.T) {
	allowedFields := []string{
		"spec.deploymentTemplate.spec.template.spec.containers.args",
		"spec.serviceAccountTemplate.metadata.name",
	}
	for i, f := range allowedFields {
		allowedFields[i] = strings.ReplaceAll(f, ".", ".?")
	}

	b := getDeploymentRuntimeConfigExpressionBuilder()

	// make sure that the allowed fields are not part of the expressions returned from the builder
	for _, val := range b.Build() {
		for _, af := range allowedFields {
			if strings.Contains(val.Expression, af) {
				t.Errorf("Expected field %q, not to be part of a validation deny expr, but got %q", af, val.Expression)
			}
		}
	}

}

func TestReconcileDeploymentConfigRuntimeProtectionPolicy(t *testing.T) {
	expExprLen := 63
	policy := &arv1.ValidatingAdmissionPolicy{}
	ReconcileDeploymentConfigRuntimeProtectionPolicy(policy)

	assert.Len(t, policy.Spec.Validations, expExprLen)
	assert.Equal(t, policy.Spec.FailurePolicy, ptr.To(arv1.Fail))
}

func TestReconcileDeploymentConfigRuntimeProtectionPolicyBinding(t *testing.T) {
	polName := "testpolicy"
	binding := &arv1.ValidatingAdmissionPolicyBinding{}
	ReconcileDeploymentConfigRuntimeProtectionPolicyBinding(polName, binding)

	assert.Equal(t, polName, binding.Spec.PolicyName)
	assert.Contains(t, binding.Spec.ValidationActions, arv1.Deny)
}
