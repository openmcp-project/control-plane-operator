package components

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/fluxcd"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/hooks"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
	"github.com/openmcp-project/control-plane-operator/pkg/utils/rcontext"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	crossplaneRelease       = "crossplane"
	CrossplaneNamespace     = "crossplane-system"
	ComponentNameCrossplane = "Crossplane"
)

var _ fluxcd.FluxComponent = &Crossplane{}
var _ TargetComponent = &Crossplane{}
var _ PolicyRulesComponent = &Crossplane{}

type Crossplane struct {
	Config *v1beta1.CrossplaneConfig
}

// GetPolicyRules implements PolicyRulesComponent.
func (c *Crossplane) GetPolicyRules() PolicyRules {
	rules := PolicyRules{
		Admin: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"pkg.crossplane.io"},
				Resources: []string{
					"configurations",
					"functions",
					"providers",
				},
				Verbs: VerbsAdmin,
			},
			{
				APIGroups: []string{"apiextensions.crossplane.io"},
				Resources: []string{
					"compositeresourcedefinitions",
					"compositions",
					"environmentconfigs",
				},
				Verbs: VerbsAdmin,
			},
			{
				APIGroups: []string{"pkg.crossplane.io"},
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     VerbsView,
			},
			{
				APIGroups: []string{"apiextensions.crossplane.io"},
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     VerbsView,
			},
		},
		View: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"pkg.crossplane.io"},
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     VerbsView,
			},
			{
				APIGroups: []string{"apiextensions.crossplane.io"},
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     VerbsView,
			},
		},
	}

	rules.Admin = append(rules.Admin, rbacv1.PolicyRule{
		APIGroups: []string{"pkg.crossplane.io"},
		Resources: []string{
			"deploymentruntimeconfigs",
		},
		Verbs: VerbsModify,
	})

	return rules
}

// GetNamespace implements TargetComponent.
func (c *Crossplane) GetNamespace() string {
	return CrossplaneNamespace
}

func (c *Crossplane) IsInstallable(ctx context.Context) (bool, error) {
	rfn := rcontext.VersionResolver(ctx)
	if _, err := rfn(crossplaneRelease, c.Config.Version); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Crossplane) BuildSourceRepository(ctx context.Context) (fluxcd.SourceAdapter, error) {
	rfn := rcontext.VersionResolver(ctx)
	c.applyDefaultChartSpec(rfn)

	repo := &sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ComponentNameCrossplane),
			Namespace: rcontext.TenantNamespace(ctx),
		},
		Spec: sourcev1.HelmRepositorySpec{
			URL:     c.Config.Chart.Repository,
			Timeout: &metav1.Duration{Duration: 1 * time.Minute},
		},
	}

	adapter := &fluxcd.HelmRepositoryAdapter{Source: repo}
	adapter.ApplyDefaults()
	return adapter, nil
}

//nolint:dupl
func (c *Crossplane) BuildManifesto(ctx context.Context) (fluxcd.Manifesto, error) {
	if err := c.applyDefaultValues(); err != nil {
		return nil, err
	}

	release := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ComponentNameCrossplane),
			Namespace: rcontext.TenantNamespace(ctx),
		},
		Spec: helmv2.HelmReleaseSpec{
			Chart: &helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart:   c.Config.Chart.Name,
					Version: c.Config.Chart.Version,
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind: "HelmRepository",
						Name: strings.ToLower(ComponentNameCrossplane),
					},
				},
			},
			ReleaseName:      crossplaneRelease,
			TargetNamespace:  CrossplaneNamespace,
			StorageNamespace: CrossplaneNamespace,
			KubeConfig:       rcontext.FluxKubeconfigRef(ctx),
			Values:           c.Config.Values,
		},
	}

	adapter := &fluxcd.HelmReleaseManifesto{Manifest: release}
	adapter.ApplyDefaults()
	return adapter, nil
}

// GetName implements Component.
func (*Crossplane) GetName() string {
	return ComponentNameCrossplane
}

// GetDependencies implements Component.
func (*Crossplane) GetDependencies() []juggler.Component {
	// No dependencies
	return []juggler.Component{}
}

// IsEnabled implements Component.
func (c *Crossplane) IsEnabled() bool {
	return c.Config != nil && c.Config.Version != ""
}

func (c *Crossplane) applyDefaultChartSpec(rfn v1beta1.VersionResolverFn) {
	if c.Config == nil {
		c.Config = &v1beta1.CrossplaneConfig{}
	}

	comp, _ := rfn(crossplaneRelease, c.Config.Version)

	if c.Config.Chart == nil {
		c.Config.Chart = &v1beta1.ChartSpec{
			Repository: comp.HelmRepo,
			Name:       comp.HelmChart,
			Version:    comp.Version,
		}
	}
}

func (c *Crossplane) applyDefaultValues() error {
	if c.Config == nil {
		return nil
	}

	// Read user-provided values
	values := map[string]any{}
	if c.Config.Values != nil {
		if err := json.Unmarshal(c.Config.Values.Raw, &values); err != nil {
			return err
		}
	}

	// Apply defaults
	if err := utils.SetNestedDefault(values, true, "rbacManager", "skipAggregatedClusterRoles"); err != nil {
		return err
	}

	// Write updated values
	encoded, err := json.Marshal(values)
	c.Config.Values = &apiextensionsv1.JSON{Raw: encoded}
	return err
}

// Hooks implements Component.
func (*Crossplane) Hooks() juggler.ComponentHooks {
	return juggler.ComponentHooks{
		PreUninstall: hooks.PreventOrphanedResources([]schema.GroupVersionKind{
			{Group: "pkg.crossplane.io", Version: "v1", Kind: "Provider"},
			{Group: "pkg.crossplane.io", Version: "v1", Kind: "ProviderRevision"},
		}),
	}
}
