package components

import (
	"context"
	"encoding/json"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/fluxcd"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/hooks"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
	"github.com/openmcp-project/control-plane-operator/pkg/utils/rcontext"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	certManagerRelease       = "cert-manager"
	certManagerNamespace     = "cert-manager"
	ComponentNameCertManager = "CertManager"
)

var _ fluxcd.FluxComponent = &CertManager{}
var _ TargetComponent = &CertManager{}

type CertManager struct {
	Config *v1beta1.CertManagerConfig
}

// GetNamespace implements TargetComponent.
func (c *CertManager) GetNamespace() string {
	return certManagerNamespace
}

func (c *CertManager) IsInstallable(ctx context.Context) (bool, error) {
	rfn := rcontext.VersionResolver(ctx)
	if _, err := rfn(certManagerRelease, c.Config.Version); err != nil {
		return false, err
	}
	return true, nil
}

func (c *CertManager) BuildSourceRepository(ctx context.Context) (fluxcd.SourceAdapter, error) {
	rfn := rcontext.VersionResolver(ctx)
	c.applyDefaultChartSpec(rfn)

	repo := &sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ComponentNameCertManager),
			Namespace: rcontext.TenantNamespace(ctx),
		},
		Spec: sourcev1.HelmRepositorySpec{
			URL: c.Config.Chart.Repository,
		},
	}

	adapter := &fluxcd.HelmRepositoryAdapter{Source: repo}
	adapter.ApplyDefaults()
	return adapter, nil
}

//nolint:dupl
func (c *CertManager) BuildManifesto(ctx context.Context) (fluxcd.Manifesto, error) {
	if err := c.applyDefaultValues(); err != nil {
		return nil, err
	}

	release := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ComponentNameCertManager),
			Namespace: rcontext.TenantNamespace(ctx),
		},
		Spec: helmv2.HelmReleaseSpec{
			Chart: &helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart:   c.Config.Chart.Name,
					Version: c.Config.Chart.Version,
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind: "HelmRepository",
						Name: strings.ToLower(ComponentNameCertManager),
					},
				},
			},
			ReleaseName:      certManagerRelease,
			TargetNamespace:  certManagerNamespace,
			StorageNamespace: certManagerNamespace,
			KubeConfig:       rcontext.FluxKubeconfigRef(ctx),
			Values:           c.Config.Values,
		},
	}

	adapter := &fluxcd.HelmReleaseManifesto{Manifest: release}
	adapter.ApplyDefaults()
	return adapter, nil
}

// GetDependencies implements Component.
func (*CertManager) GetDependencies() []juggler.Component {
	return []juggler.Component{}
}

// GetName implements Component.
func (*CertManager) GetName() string {
	return ComponentNameCertManager
}

// IsEnabled implements Component.
func (c *CertManager) IsEnabled() bool {
	return c.Config != nil && c.Config.Version != ""
}

func (c *CertManager) applyDefaultChartSpec(rfn v1beta1.VersionResolverFn) {
	if c.Config == nil {
		c.Config = &v1beta1.CertManagerConfig{}
	}

	comp, _ := rfn(certManagerRelease, c.Config.Version)

	if c.Config.Chart == nil {
		c.Config.Chart = &v1beta1.ChartSpec{
			Repository: "https://charts.jetstack.io",
			Name:       "cert-manager",
			Version:    comp.Version,
		}
	}
}

// Hooks implements Component.
func (*CertManager) Hooks() juggler.ComponentHooks {
	return juggler.ComponentHooks{
		PreUninstall: hooks.PreventOrphanedResources([]schema.GroupVersionKind{
			{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"},
			{Group: "cert-manager.io", Version: "v1", Kind: "Issuer"},
			{Group: "cert-manager.io", Version: "v1", Kind: "ClusterIssuer"},
		}),
	}
}

func (c *CertManager) applyDefaultValues() error {
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
	if err := utils.SetNestedDefault(values, true, "installCRDs"); err != nil {
		return err
	}

	// Write updated values
	encoded, err := json.Marshal(values)
	c.Config.Values = &apiextensionsv1.JSON{Raw: encoded}
	return err
}
