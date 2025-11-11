package components

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/fluxcd"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/hooks"
	"github.com/openmcp-project/control-plane-operator/pkg/utils/rcontext"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	kyvernoRelease       = "kyverno"
	kyvernoNamespace     = "kyverno-system"
	ComponentNameKyverno = "Kyverno"

	EnvEnableKyvernoDefaultValues = "ENABLE_KYVERNO_DEFAULT_VALUES"
)

var _ fluxcd.FluxComponent = &Kyverno{}
var _ TargetComponent = &Kyverno{}
var _ PolicyRulesComponent = &Kyverno{}

type Kyverno struct {
	Config *v1beta1.KyvernoConfig
}

// GetPolicyRules implements PolicyRulesComponent.
func (k *Kyverno) GetPolicyRules() PolicyRules {
	return PolicyRules{
		Admin: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"kyverno.io"},
				Resources: []string{
					"cleanuppolicies",
					"clustercleanuppolicies",
					"policies",
					"clusterpolicies",
					"admissionreports",
					"clusteradmissionreports",
					"backgroundscanreports",
					"clusterbackgroundscanreports",
					"updaterequests",
				},
				Verbs: VerbsAdmin,
			},
			{
				APIGroups: []string{"wgpolicyk8s.io"},
				Resources: []string{
					"policyreports",
					"clusterpolicyreports",
				},
				Verbs: VerbsAdmin,
			},
			{
				APIGroups: []string{"reports.kyverno.io"},
				Resources: []string{
					"ephemeralreports",
					"clusterephemeralreports",
				},
				Verbs: VerbsAdmin,
			},
		},
		View: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"kyverno.io"},
				Resources: []string{
					"cleanuppolicies",
					"clustercleanuppolicies",
					"policies",
					"clusterpolicies",
					"admissionreports",
					"clusteradmissionreports",
					"backgroundscanreports",
					"clusterbackgroundscanreports",
					"updaterequests",
				},
				Verbs: VerbsView,
			},
			{
				APIGroups: []string{"wgpolicyk8s.io"},
				Resources: []string{
					"policyreports",
					"clusterpolicyreports",
				},
				Verbs: VerbsView,
			},
			{
				APIGroups: []string{"reports.kyverno.io"},
				Resources: []string{
					"ephemeralreports",
					"clusterephemeralreports",
				},
				Verbs: VerbsView,
			},
		},
	}
}

// GetNamespace implements TargetComponent.
func (k *Kyverno) GetNamespace() string {
	return kyvernoNamespace
}

func (k *Kyverno) GetName() string {
	return ComponentNameKyverno
}

func (k *Kyverno) GetDependencies() []juggler.Component {
	return []juggler.Component{}
}

func (k *Kyverno) IsEnabled() bool {
	return k.Config != nil && k.Config.Version != ""
}

func (k *Kyverno) Hooks() juggler.ComponentHooks {
	return juggler.ComponentHooks{
		PreUninstall: hooks.PreventOrphanedResources([]schema.GroupVersionKind{
			{Group: "kyverno.io", Version: "v1", Kind: "ClusterPolicy"},
			{Group: "kyverno.io", Version: "v1", Kind: "Policy"},
		}),
	}
}

func (k *Kyverno) IsInstallable(ctx context.Context) (bool, error) {
	rfn := rcontext.VersionResolver(ctx)
	if _, err := rfn(kyvernoRelease, k.Config.Version); err != nil {
		return false, err
	}
	return true, nil
}

func (k *Kyverno) BuildSourceRepository(ctx context.Context) (fluxcd.SourceAdapter, error) {
	rfn := rcontext.VersionResolver(ctx)
	k.applyDefaultChartSpec(rfn)

	repo := &sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ComponentNameKyverno),
			Namespace: rcontext.TenantNamespace(ctx),
		},
		Spec: sourcev1.HelmRepositorySpec{
			URL: k.Config.Chart.Repository,
		},
	}

	adapter := &fluxcd.HelmRepositoryAdapter{Source: repo}
	adapter.ApplyDefaults()
	return adapter, nil
}

func (k *Kyverno) applyDefaultChartSpec(rfn v1beta1.VersionResolverFn) {
	if k.Config == nil {
		k.Config = &v1beta1.KyvernoConfig{}
	}

	comp, _ := rfn(kyvernoRelease, k.Config.Version)

	if k.Config.Chart == nil {
		k.Config.Chart = &v1beta1.ChartSpec{
			Repository: "https://kyverno.github.io/kyverno",
			Name:       "kyverno",
			Version:    comp.Version,
		}
	}
}

func (k *Kyverno) BuildManifesto(ctx context.Context) (fluxcd.Manifesto, error) {
	values := k.Config.Values

	// If default values are enabled, use them instead
	if os.Getenv(EnvEnableKyvernoDefaultValues) == "true" {
		defaultValues, err := k.getDefaultValues()
		if err != nil {
			return nil, fmt.Errorf("failed to get default Kyverno values: %w", err)
		}
		values = defaultValues
	}

	release := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ComponentNameKyverno),
			Namespace: rcontext.TenantNamespace(ctx),
		},
		Spec: helmv2.HelmReleaseSpec{
			Chart: &helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart:   k.Config.Chart.Name,
					Version: k.Config.Chart.Version,
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind: "HelmRepository",
						Name: strings.ToLower(ComponentNameKyverno),
					},
				},
			},
			ReleaseName:      kyvernoRelease,
			TargetNamespace:  kyvernoNamespace,
			StorageNamespace: kyvernoNamespace,
			KubeConfig:       rcontext.FluxKubeconfigRef(ctx),
			Values:           values,
		},
	}

	adapter := &fluxcd.HelmReleaseManifesto{Manifest: release}
	adapter.ApplyDefaults()
	return adapter, nil
}

func (k *Kyverno) getDefaultValues() (*apiextensionsv1.JSON, error) {
	defaults := map[string]interface{}{
		"config": map[string]interface{}{
			"preserve": false,
			"resourceFilters": []string{
				"[*/*,kyverno,*]",
				"[*/*,istio-system,*]",
				"[*/*,kyma-system,*]",
				"[*/*,kube-system,*]",
				"[*/*,kube-public,*]",
				"[*/*,neo-core,*]",
			},
			"updateRequestThreshold": 5000,
			"excludeGroups": []string{
				"system:nodes",
			},
			"webhooks": map[string]interface{}{
				"namespaceSelector": map[string]interface{}{
					"matchExpressions": []map[string]interface{}{
						{
							"key":      "kubernetes.io/metadata.name",
							"operator": "NotIn",
							"values": []string{
								"kube-system",
								"kyverno",
								"istio-system",
								"kube-public",
								"kyma-system",
								"neo-core",
							},
						},
					},
				},
			},
		},
	}

	raw, err := json.Marshal(defaults)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal default values: %w", err)
	}
	return &apiextensionsv1.JSON{Raw: raw}, nil
}
