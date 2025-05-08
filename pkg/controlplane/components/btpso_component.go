package components

import (
	"context"
	"encoding/json"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/fluxcd"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/hooks"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
	"github.com/openmcp-project/control-plane-operator/pkg/utils/rcontext"
)

const (
	btpServiceOperatorNamespace = "sap-btp-service-operator"
	btpServiceOperatorRelease   = "sap-btp-service-operator"
	ComponentNameBTPSO          = "BTPServiceOperator"
)

var _ fluxcd.FluxComponent = &BTPServiceOperator{}
var _ TargetComponent = &BTPServiceOperator{}
var _ PolicyRulesComponent = &BTPServiceOperator{}

// BTPServiceOperator is the add-on for https://github.com/SAP/sap-btp-service-operator.
type BTPServiceOperator struct {
	Config *v1beta1.BTPServiceOperatorConfig
}

// GetPolicyRules implements PolicyRulesComponent.
func (btp *BTPServiceOperator) GetPolicyRules() PolicyRules {
	return PolicyRules{
		Admin: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"services.cloud.sap.com"},
				Resources: []string{
					"servicebindings",
					"serviceinstances",
				},
				Verbs: VerbsAdmin,
			},
		},
		View: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"services.cloud.sap.com"},
				Resources: []string{
					"servicebindings",
					"serviceinstances",
				},
				Verbs: VerbsView,
			},
		},
	}
}

// GetNamespace implements TargetComponent.
func (btp *BTPServiceOperator) GetNamespace() string {
	return btpServiceOperatorNamespace
}

func (btp *BTPServiceOperator) IsInstallable(ctx context.Context) (bool, error) {
	rfn := rcontext.VersionResolver(ctx)
	if _, err := rfn(btpServiceOperatorRelease, btp.Config.Version); err != nil {
		return false, err
	}
	return true, nil
}

func (btp *BTPServiceOperator) BuildSourceRepository(ctx context.Context) (fluxcd.SourceAdapter, error) {
	rfn := rcontext.VersionResolver(ctx)
	btp.applyDefaultChartSpec(rfn)

	repo := &sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ComponentNameBTPSO),
			Namespace: rcontext.TenantNamespace(ctx),
		},
		Spec: sourcev1.HelmRepositorySpec{
			URL: btp.Config.Chart.Repository,
		},
	}

	adapter := &fluxcd.HelmRepositoryAdapter{Source: repo}
	adapter.ApplyDefaults()
	return adapter, nil
}

//nolint:dupl
func (btp *BTPServiceOperator) BuildManifesto(ctx context.Context) (fluxcd.Manifesto, error) {
	if err := btp.applyDefaultValues(); err != nil {
		return nil, err
	}

	release := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(ComponentNameBTPSO),
			Namespace: rcontext.TenantNamespace(ctx),
		},
		Spec: helmv2.HelmReleaseSpec{
			Chart: &helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart:   btp.Config.Chart.Name,
					Version: btp.Config.Chart.Version,
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind: "HelmRepository",
						Name: strings.ToLower(ComponentNameBTPSO), // repo name
					},
				},
			},
			ReleaseName:      btpServiceOperatorRelease,
			TargetNamespace:  btpServiceOperatorNamespace,
			StorageNamespace: btpServiceOperatorNamespace,
			KubeConfig:       rcontext.FluxKubeconfigRef(ctx),
			Values:           btp.Config.Values,
		},
	}

	adapter := &fluxcd.HelmReleaseManifesto{Manifest: release}
	adapter.ApplyDefaults()
	return adapter, nil
}

// GetName implements Component.
func (btp *BTPServiceOperator) GetName() string {
	return ComponentNameBTPSO
}

// GetDependencies implements Component.
func (btp *BTPServiceOperator) GetDependencies() []juggler.Component {
	return []juggler.Component{&CertManager{}}
}

// IsEnabled implements Component.
func (btp *BTPServiceOperator) IsEnabled() bool {
	return btp.Config != nil && btp.Config.Version != ""
}

func (btp *BTPServiceOperator) applyDefaultChartSpec(rfn v1beta1.VersionResolverFn) {
	if btp.Config == nil {
		btp.Config = &v1beta1.BTPServiceOperatorConfig{}
	}

	comp, _ := rfn(btpServiceOperatorRelease, btp.Config.Version)

	if btp.Config.Chart == nil {
		btp.Config.Chart = &v1beta1.ChartSpec{
			Repository: "https://sap.github.io/sap-btp-service-operator",
			Name:       "sap-btp-operator",
			Version:    comp.Version,
		}
	}
}

// Hooks implements Component.
func (btp *BTPServiceOperator) Hooks() juggler.ComponentHooks {
	return juggler.ComponentHooks{
		PreUninstall: hooks.PreventOrphanedResources([]schema.GroupVersionKind{
			{Group: "services.cloud.sap.com", Version: "v1", Kind: "ServiceBinding"},
			{Group: "services.cloud.sap.com", Version: "v1", Kind: "ServiceInstance"},
		}),
	}
}

func (btp *BTPServiceOperator) applyDefaultValues() error {
	if btp.Config == nil {
		return nil
	}

	// Read user-provided values
	values := map[string]any{}
	if btp.Config.Values != nil {
		if err := json.Unmarshal(btp.Config.Values.Raw, &values); err != nil {
			return err
		}
	}

	// Apply defaults
	if err := utils.SetNestedDefault(values, "sap-btp-service-operator", "cluster", "id"); err != nil {
		return err
	}
	if err := utils.SetNestedDefault(values, 1, "manager", "replica_count"); err != nil {
		return err
	}

	// Write updated values
	encoded, err := json.Marshal(values)
	btp.Config.Values = &apiextensionsv1.JSON{Raw: encoded}
	return err
}
