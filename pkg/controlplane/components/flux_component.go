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
	fluxRelease       = "flux"
	fluxNamespace     = "flux-system"
	ComponentNameFlux = "Flux"
)

var _ fluxcd.FluxComponent = &Flux{}
var _ TargetComponent = &Flux{}
var _ PolicyRulesComponent = &Flux{}

type Flux struct {
	Config *v1beta1.FluxConfig
}

func (f *Flux) GetPolicyRules() PolicyRules {
	return PolicyRules{
		Admin: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"notification.toolkit.fluxcd.io",
					"source.toolkit.fluxcd.io",
					"helm.toolkit.fluxcd.io",
					"image.toolkit.fluxcd.io",
					"kustomize.toolkit.fluxcd.io",
				},
				Resources: []string{
					"*",
				},
				Verbs: VerbsAdmin,
			},
		},
		View: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"notification.toolkit.fluxcd.io",
					"source.toolkit.fluxcd.io",
					"helm.toolkit.fluxcd.io",
					"image.toolkit.fluxcd.io",
					"kustomize.toolkit.fluxcd.io",
				},
				Resources: []string{
					"*",
				},
				Verbs: VerbsView,
			},
		},
	}
}

func (f *Flux) GetNamespace() string {
	return fluxNamespace
}

func (f *Flux) GetName() string {
	return ComponentNameFlux
}

func (f *Flux) GetDependencies() []juggler.Component {
	return []juggler.Component{}
}

func (f *Flux) IsEnabled() bool {
	return f.Config != nil && f.Config.Version != ""
}

func (f *Flux) Hooks() juggler.ComponentHooks {
	return juggler.ComponentHooks{
		PreUninstall: hooks.PreventOrphanedResources([]schema.GroupVersionKind{
			{Group: "helm.toolkit.fluxcd.io", Version: "v2", Kind: "HelmRelease"},
			{Group: "kustomize.toolkit.fluxcd.io", Version: "v1", Kind: "Kustomization"},
		}),
	}
}

func (f *Flux) IsInstallable(ctx context.Context) (bool, error) {
	rfn := rcontext.VersionResolver(ctx)
	if _, err := rfn(fluxRelease, f.Config.Version); err != nil {
		return false, err
	}
	return true, nil
}

func (f *Flux) applyDefaultChartSpec(rfn v1beta1.VersionResolverFn) {
	if f.Config == nil {
		f.Config = &v1beta1.FluxConfig{}
	}

	comp, _ := rfn(fluxRelease, f.Config.Version)

	if f.Config.Chart == nil {
		f.Config.Chart = &v1beta1.ChartSpec{
			Repository: "https://fluxcd-community.github.io/helm-charts",
			Name:       "flux2",
			Version:    comp.Version,
		}
	}
}

func (f *Flux) BuildSourceRepository(ctx context.Context) (fluxcd.SourceAdapter, error) {
	rfn := rcontext.VersionResolver(ctx)
	f.applyDefaultChartSpec(rfn)

	repo := &sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ComponentNameFlux),
			Namespace: rcontext.TenantNamespace(ctx),
		},
		Spec: sourcev1.HelmRepositorySpec{
			URL: f.Config.Chart.Repository,
		},
	}

	adapter := &fluxcd.HelmRepositoryAdapter{Source: repo}
	adapter.ApplyDefaults()
	return adapter, nil
}

func (f *Flux) BuildManifesto(ctx context.Context) (fluxcd.Manifesto, error) {
	release := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ComponentNameFlux),
			Namespace: rcontext.TenantNamespace(ctx),
		},
		Spec: helmv2.HelmReleaseSpec{
			Chart: &helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart:   f.Config.Chart.Name,
					Version: f.Config.Chart.Version,
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind: "HelmRepository",
						Name: strings.ToLower(ComponentNameFlux),
					},
				},
			},
			ReleaseName:      fluxRelease,
			TargetNamespace:  fluxNamespace,
			StorageNamespace: fluxNamespace,
			KubeConfig:       rcontext.FluxKubeconfigRef(ctx),
			Values:           f.Config.Values,
		},
	}

	adapter := &fluxcd.HelmReleaseManifesto{Manifest: release}
	adapter.ApplyDefaults()
	return adapter, nil
}
