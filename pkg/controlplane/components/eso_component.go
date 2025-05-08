package components

import (
	"context"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/fluxcd"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/hooks"
	"github.com/openmcp-project/control-plane-operator/pkg/utils/rcontext"
)

const (
	esoRelease       = "external-secrets"
	esoNamespace     = "external-secrets"
	ComponentNameESO = "ExternalSecretsOperator"
)

var _ fluxcd.FluxComponent = &ExternalSecretsOperator{}
var _ TargetComponent = &ExternalSecretsOperator{}
var _ PolicyRulesComponent = &ExternalSecretsOperator{}

// ExternalSecretsOperator is the add-on for https://github.com/external-secrets/external-secrets.
type ExternalSecretsOperator struct {
	Config *v1beta1.ExternalSecretsOperatorConfig
}

// GetPolicyRules implements PolicyRulesComponent.
func (e *ExternalSecretsOperator) GetPolicyRules() PolicyRules {
	return PolicyRules{
		Admin: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"external-secrets.io"},
				Resources: []string{
					"externalsecrets",
					"secretstores",
					"clustersecretstores",
					"pushsecrets",
				},
				Verbs: VerbsAdmin,
			},
			{
				APIGroups: []string{"generators.external-secrets.io"},
				Resources: []string{
					"vaultdynamicsecrets",
				},
				Verbs: VerbsAdmin,
			},
		},
		View: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"external-secrets.io"},
				Resources: []string{
					"externalsecrets",
					"secretstores",
					"clustersecretstores",
					"pushsecrets",
				},
				Verbs: VerbsView,
			},
			{
				APIGroups: []string{"generators.external-secrets.io"},
				Resources: []string{
					"vaultdynamicsecrets",
				},
				Verbs: VerbsView,
			},
		},
	}
}

// GetNamespace implements TargetComponent.
func (e *ExternalSecretsOperator) GetNamespace() string {
	return esoNamespace
}

func (e *ExternalSecretsOperator) IsInstallable(ctx context.Context) (bool, error) {
	rfn := rcontext.VersionResolver(ctx)
	if _, err := rfn(esoRelease, e.Config.Version); err != nil {
		return false, err
	}
	return true, nil
}

func (e *ExternalSecretsOperator) BuildSourceRepository(ctx context.Context) (fluxcd.SourceAdapter, error) {
	rfn := rcontext.VersionResolver(ctx)
	e.applyDefaultChartSpec(rfn)

	repo := &sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ComponentNameESO),
			Namespace: rcontext.TenantNamespace(ctx),
		},
		Spec: sourcev1.HelmRepositorySpec{
			URL: e.Config.Chart.Repository,
		},
	}

	adapter := &fluxcd.HelmRepositoryAdapter{Source: repo}
	adapter.ApplyDefaults()
	return adapter, nil
}

func (e *ExternalSecretsOperator) BuildManifesto(ctx context.Context) (fluxcd.Manifesto, error) {
	release := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ComponentNameESO),
			Namespace: rcontext.TenantNamespace(ctx),
		},
		Spec: helmv2.HelmReleaseSpec{
			Chart: &helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart:   e.Config.Chart.Name,
					Version: e.Config.Chart.Version,
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind: "HelmRepository",
						Name: strings.ToLower(ComponentNameESO),
					},
				},
			},
			ReleaseName:      esoRelease,
			TargetNamespace:  esoNamespace,
			StorageNamespace: esoNamespace,
			KubeConfig:       rcontext.FluxKubeconfigRef(ctx),
			Values:           e.Config.Values,
		},
	}

	adapter := &fluxcd.HelmReleaseManifesto{Manifest: release}
	adapter.ApplyDefaults()
	return adapter, nil
}

func (e *ExternalSecretsOperator) GetName() string {
	return ComponentNameESO
}

func (e *ExternalSecretsOperator) GetDependencies() []juggler.Component {
	// No dependencies
	return []juggler.Component{}
}

func (e *ExternalSecretsOperator) IsEnabled() bool {
	return e.Config != nil && e.Config.Version != ""
}

func (e *ExternalSecretsOperator) applyDefaultChartSpec(rfn v1beta1.VersionResolverFn) {
	if e.Config == nil {
		e.Config = &v1beta1.ExternalSecretsOperatorConfig{}
	}

	comp, _ := rfn(esoRelease, e.Config.Version)

	if e.Config.Chart == nil {
		e.Config.Chart = &v1beta1.ChartSpec{
			Repository: "https://charts.external-secrets.io",
			Name:       "external-secrets",
			Version:    comp.Version,
		}
	}
}

// Hooks implements Component.
func (e *ExternalSecretsOperator) Hooks() juggler.ComponentHooks {
	return juggler.ComponentHooks{
		PreUninstall: hooks.PreventOrphanedResources([]schema.GroupVersionKind{
			{Group: "external-secrets.io", Version: "v1beta1", Kind: "ExternalSecret"},
			{Group: "external-secrets.io", Version: "v1beta1", Kind: "SecretStore"},
			{Group: "external-secrets.io", Version: "v1beta1", Kind: "ClusterSecretStore"},
		}),
	}
}
