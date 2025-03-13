package fluxcd

import (
	"errors"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errNotAHelmReleaseManifesto = errors.New("FluxResource is not a HelmReleaseManifesto")
)

var _ Manifesto = &HelmReleaseManifesto{}

type HelmReleaseManifesto struct {
	Manifest *helmv2.HelmRelease
}

// Reconcile implements Manifesto.
func (h *HelmReleaseManifesto) Reconcile(desired FluxResource) error {
	desiredManifesto, ok := desired.(*HelmReleaseManifesto)
	if !ok {
		return errNotAHelmReleaseManifesto
	}

	preserved := h.Manifest.Spec.DeepCopy()
	h.Manifest.Spec = desiredManifesto.Manifest.Spec
	// Give suspension precedence
	h.Manifest.Spec.Suspend = preserved.Suspend
	return nil
}

// Empty implements Manifesto.
func (h *HelmReleaseManifesto) Empty() Manifesto {
	return &HelmReleaseManifesto{&helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.Manifest.Name,
			Namespace: h.Manifest.Namespace,
		},
	}}
}

// GetHealthiness implements Manifesto.
func (h *HelmReleaseManifesto) GetHealthiness() juggler.ResourceHealthiness {
	cond := apimeta.FindStatusCondition(h.Manifest.Status.Conditions, fluxmeta.ReadyCondition)
	if cond == nil {
		return juggler.ResourceHealthiness{
			Healthy: false,
			Message: msgReadyNotPresent,
		}
	}
	return juggler.ResourceHealthiness{
		Healthy: cond.Status == metav1.ConditionTrue,
		Message: cond.Message,
	}
}

func (h *HelmReleaseManifesto) GetObjectKey() client.ObjectKey {
	return client.ObjectKey{
		Namespace: h.Manifest.Namespace,
		Name:      h.Manifest.Name,
	}
}

func (h *HelmReleaseManifesto) GetObject() client.Object {
	return h.Manifest
}

func (h *HelmReleaseManifesto) ApplyDefaults() {
	// This usually does nothing but we can keep it here in case Flux resources will have
	// a defaulting func in the future.
	scheme.Default(h.Manifest)

	h.Manifest.Spec.Interval = metav1.Duration{Duration: 5 * time.Minute}

	if h.Manifest.Spec.Install == nil {
		h.Manifest.Spec.Install = &helmv2.Install{}
	}

	h.Manifest.Spec.Install.CreateNamespace = true

	if h.Manifest.Spec.Install.Remediation == nil {
		h.Manifest.Spec.Install.Remediation = &helmv2.InstallRemediation{}
	}

	h.Manifest.Spec.Install.Remediation.Retries = -1

	if h.Manifest.Spec.Upgrade == nil {
		h.Manifest.Spec.Upgrade = &helmv2.Upgrade{}
	}

	if h.Manifest.Spec.Upgrade.Remediation == nil {
		h.Manifest.Spec.Upgrade.Remediation = &helmv2.UpgradeRemediation{}
	}

	h.Manifest.Spec.Upgrade.Remediation.Retries = -1
}
