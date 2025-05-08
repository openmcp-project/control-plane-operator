package fluxcd

import (
	"errors"
	"time"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

var (
	errNotAHelmRepositoryAdapter = errors.New("FluxResource is not a HelmRepositoryAdapter")
	errNotAGitRepositoryAdapter  = errors.New("FluxResource is not a GitRepositoryAdapter")
)

const (
	msgReadyNotPresent = "Unable to check healthiness. Ready condition is not present."
)

/// use strategy for  with adapter pattern
// adapter for third party types to conform to a common interface
// use strategy for the reconcilers on how each can be reconciled e.g. for the applyResources method in flux-reconciler

var _ SourceAdapter = &HelmRepositoryAdapter{}

// HelmRepositoryAdapter implements SourceAdapter
type HelmRepositoryAdapter struct {
	Source *sourcev1.HelmRepository
}

// Reconcile implements SourceAdapter.
func (h *HelmRepositoryAdapter) Reconcile(desired FluxResource) error {
	desiredAdapter, ok := desired.(*HelmRepositoryAdapter)
	if !ok {
		return errNotAHelmRepositoryAdapter
	}

	preserved := h.Source.Spec.DeepCopy()
	h.Source.Spec = desiredAdapter.Source.Spec
	// Give suspension precedence
	h.Source.Spec.Suspend = preserved.Suspend
	return nil
}

// Empty implements SourceAdapter.
func (h *HelmRepositoryAdapter) Empty() SourceAdapter {
	return &HelmRepositoryAdapter{&sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.Source.Name,
			Namespace: h.Source.Namespace,
		},
	}}
}

// GetHealthiness implements SourceAdapter.
func (h *HelmRepositoryAdapter) GetHealthiness() juggler.ResourceHealthiness {
	cond := apimeta.FindStatusCondition(h.Source.Status.Conditions, fluxmeta.ReadyCondition)
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

// GetObject returns the HelmRepository as a client.Object to be used with the client.Client or alike interfaces
func (h *HelmRepositoryAdapter) GetObject() client.Object {
	return h.Source
}

// GetObjectKey returns the HelmRepository as a client.ObjectKey to be used with client.Client.Get
func (h *HelmRepositoryAdapter) GetObjectKey() client.ObjectKey {
	return client.ObjectKey{
		Namespace: h.Source.Namespace,
		Name:      h.Source.Name,
	}
}

func (h *HelmRepositoryAdapter) ApplyDefaults() {
	// This usually does nothing but we can keep it here in case Flux resources will have
	// a defaulting func in the future.
	scheme.Default(h.Source)

	h.Source.Spec.Interval = metav1.Duration{Duration: 1 * time.Hour}
}

//
// -----------------------------------
//

var _ SourceAdapter = &GitRepositoryAdapter{}

// GitRepositoryAdapter implements SourceAdapter
type GitRepositoryAdapter struct {
	Source *sourcev1.GitRepository
}

// Reconcile implements SourceAdapter.
func (g *GitRepositoryAdapter) Reconcile(desired FluxResource) error {
	desiredAdapter, ok := desired.(*GitRepositoryAdapter)
	if !ok {
		return errNotAGitRepositoryAdapter
	}

	preserved := g.Source.Spec.DeepCopy()
	g.Source.Spec = desiredAdapter.Source.Spec
	// Give suspension precedence
	g.Source.Spec.Suspend = preserved.Suspend
	return nil
}

// Empty implements SourceAdapter.
func (g *GitRepositoryAdapter) Empty() SourceAdapter {
	return &GitRepositoryAdapter{&sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      g.Source.Name,
			Namespace: g.Source.Namespace,
		},
	}}
}

// GetHealthiness implements SourceAdapter.
func (g *GitRepositoryAdapter) GetHealthiness() juggler.ResourceHealthiness {
	cond := apimeta.FindStatusCondition(g.Source.Status.Conditions, fluxmeta.ReadyCondition)
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

// GetObject returns the GitRepository as a client.Object to be used with the client.Client or alike interfaces
func (g *GitRepositoryAdapter) GetObject() client.Object {
	return g.Source
}

// GetObjectKey returns the GitRepository as a client.ObjectKey to be used with client.Client.Get
func (g *GitRepositoryAdapter) GetObjectKey() client.ObjectKey {
	return client.ObjectKey{
		Namespace: g.Source.Namespace,
		Name:      g.Source.Name,
	}
}

func (g *GitRepositoryAdapter) ApplyDefaults() {
	// This usually does nothing but we can keep it here in case Flux resources will have
	// a defaulting func in the future.
	scheme.Default(g.Source)

	g.Source.Spec.Interval = metav1.Duration{Duration: 1 * time.Hour}
}
