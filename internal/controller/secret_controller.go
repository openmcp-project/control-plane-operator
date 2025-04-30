package controller

import (
	"context"
	"errors"
	"strings"

	"github.com/openmcp-project/control-plane-operator/pkg/constants"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1beta1 "github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
)

const (
	finalizerOrphan = corev1beta1.Finalizer + "/orphan"
)

var (
	errFailedToCopySecret = errors.New("failed to copy secret")
	errFailedToDeleteCopy = errors.New("failed to delete secret copy")
)

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	secret := &corev1.Secret{}
	if err := r.Get(ctx, req.NamespacedName, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Secret not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Secret")
		return ctrl.Result{}, err
	}

	// set labels to a non-nil value because we read from them multiple times
	if secret.Labels == nil {
		secret.Labels = map[string]string{}
	}

	if !r.shouldReconcile(secret) {
		return ctrl.Result{}, nil
	}

	if !secret.DeletionTimestamp.IsZero() || secret.Labels[constants.LabelCopyToCPNamespace] != "true" {
		return r.handleDeletion(ctx, secret)
	}

	if err := r.ensureFinalizer(ctx, secret); err != nil {
		return ctrl.Result{}, err
	}

	return r.handleSync(ctx, secret)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&corev1.Secret{},
			builder.WithPredicates(r.buildFilterPredicate()),
		).
		Complete(r)
}

func (r *SecretReconciler) handleDeletion(ctx context.Context, secret *corev1.Secret) (ctrl.Result, error) {
	matchingLabels := client.MatchingLabels{
		constants.LabelCopySourceName:      secret.Name,
		constants.LabelCopySourceNamespace: secret.Namespace,
	}

	copies := &corev1.SecretList{}
	if err := r.List(ctx, copies, matchingLabels); err != nil {
		return ctrl.Result{}, err
	}

	if len(copies.Items) == 0 {
		return ctrl.Result{}, r.removeFinalizer(ctx, secret)
	}

	for _, copy := range copies.Items {
		if err := r.Delete(ctx, &copy); err != nil {
			return ctrl.Result{}, errors.Join(errFailedToDeleteCopy, err)
		}
	}

	return ctrl.Result{RequeueAfter: requeueAfterError}, nil
}

func (r *SecretReconciler) handleSync(ctx context.Context, secret *corev1.Secret) (ctrl.Result, error) {
	namespaces := &corev1.NamespaceList{}
	if err := r.List(ctx, namespaces); err != nil {
		return ctrl.Result{}, err
	}

	for _, ns := range namespaces.Items {
		if !strings.HasPrefix(ns.Name, cpNamespacePrefix) {
			continue
		}

		copy := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret.Name,
				Namespace: ns.Name,
			},
		}

		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, copy, func() error {
			copy.Type = secret.Type
			copy.Data = secret.Data
			metav1.SetMetaDataLabel(&copy.ObjectMeta, constants.LabelCopySourceName, secret.Name)
			metav1.SetMetaDataLabel(&copy.ObjectMeta, constants.LabelCopySourceNamespace, secret.Namespace)
			utils.SetManagedBy(copy)
			return nil
		})
		if err != nil {
			return ctrl.Result{}, errors.Join(errFailedToCopySecret, err)
		}
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

func (r *SecretReconciler) shouldReconcile(o client.Object) bool {
	labels := o.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	_, hasLabel := labels[constants.LabelCopyToCPNamespace]

	return hasLabel || r.hasFinalizer(o)
}

func (r *SecretReconciler) ensureFinalizer(ctx context.Context, o client.Object) error {
	updated := controllerutil.AddFinalizer(o, finalizerOrphan)
	if updated {
		return r.Update(ctx, o)
	}
	return nil
}

func (r *SecretReconciler) removeFinalizer(ctx context.Context, o client.Object) error {
	updated := controllerutil.RemoveFinalizer(o, finalizerOrphan)
	if updated {
		return r.Update(ctx, o)
	}
	return nil
}

func (r *SecretReconciler) hasFinalizer(o client.Object) bool {
	return controllerutil.ContainsFinalizer(o, finalizerOrphan)
}

func (r *SecretReconciler) buildFilterPredicate() filterObjectPredicate {
	return filterObjectPredicate{filterFunc: r.shouldReconcile}
}
