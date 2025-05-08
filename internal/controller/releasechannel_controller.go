package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ocmlib "ocm.software/ocm/api/ocm"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/internal/ocm"
)

type ReleaseChannelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ReleaseChannelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var releasechannel v1beta1.ReleaseChannel
	if err := r.Get(ctx, req.NamespacedName, &releasechannel); err != nil {
		log.Error(err, "unable to fetch ReleaseChannel")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Reconciling ReleaseChannel")

	var repo ocmlib.Repository
	var componentNames []string
	if releasechannel.Spec.OcmRegistryUrl != "" {
		// Get the secret using the PullSecretRef in the ReleaseChannel
		var secret corev1.Secret
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: releasechannel.Spec.PullSecretRef.Namespace,
			Name:      releasechannel.Spec.PullSecretRef.Name,
		}, &secret); err != nil {
			log.Error(err, "unable to fetch Secret")
			return ctrl.Result{
				RequeueAfter: requeueAfterError,
			}, fmt.Errorf("unable to fetch Secret: %w", err)
		}

		var err error
		repo, componentNames, err = ocm.GetOCMRemoteRepo(releasechannel.Spec.OcmRegistryUrl, secret, releasechannel.Spec.PrefixFilter)
		if err != nil {
			log.Error(err, "unable to get components from remote OCM")
			return ctrl.Result{
				RequeueAfter: requeueAfterError,
			}, fmt.Errorf("unable to get components from OCM: %w", err)
		}
	} else if releasechannel.Spec.OcmRegistrySecretRef.Name != "" && releasechannel.Spec.OcmRegistrySecretKey != "" {
		var err error
		// Get data from secret
		var secret corev1.Secret
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: releasechannel.Spec.OcmRegistrySecretRef.Namespace,
			Name:      releasechannel.Spec.OcmRegistrySecretRef.Name,
		}, &secret); err != nil {
			log.Error(err, "unable to fetch Secret")
			return ctrl.Result{
				RequeueAfter: requeueAfterError,
			}, fmt.Errorf("unable to fetch Secret: %w", err)
		}

		data, ok := secret.Data[releasechannel.Spec.OcmRegistrySecretKey]
		if !ok {
			err := fmt.Errorf("key %s not found in secret %s", releasechannel.Spec.OcmRegistrySecretKey, releasechannel.Spec.OcmRegistrySecretRef.Name)
			log.Error(err, "unable to get data from secret")
			return ctrl.Result{
				RequeueAfter: requeueAfterError,
			}, err
		}

		repo, componentNames, err = ocm.GetOCMLocalRepo(data, releasechannel.Spec.PrefixFilter)
		if err != nil {
			log.Error(err, "unable to get components from local OCM")
			return ctrl.Result{
				RequeueAfter: requeueAfterError,
			}, fmt.Errorf("unable to get components from local OCM: %w", err)
		}
	} else {
		return ctrl.Result{
			RequeueAfter: requeueAfterError,
		}, fmt.Errorf("either 'OcmRegistryUrl' or 'OcmRegistrySecretRef & OcmRegistrySecretKey' must be set")
	}

	components, err := ocm.GetOCMComponentsWithVersions(repo, componentNames, releasechannel.Spec.PrefixFilter)
	if err != nil {
		log.Error(err, "unable to get components from OCM")
		return ctrl.Result{
			RequeueAfter: requeueAfterError,
		}, fmt.Errorf("unable to get components from OCM: %w", err)
	}

	releasechannel.Status.Components = components

	err = r.Status().Update(ctx, &releasechannel)
	if err != nil {
		log.Error(err, "unable to update ReleaseChannel status")
		return ctrl.Result{
			RequeueAfter: requeueAfterError,
		}, fmt.Errorf("unable to update ReleaseChannel status: %w", err)
	}

	log.Info("Finish Reconciling ReleaseChannel")

	return ctrl.Result{
		RequeueAfter: releasechannel.Spec.Interval.Duration,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseChannelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	updatePred := predicate.Funcs{
		// Only allow updates when the spec.size of the Busybox resource changes
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj := e.ObjectOld.(*v1beta1.ReleaseChannel)
			newObj := e.ObjectNew.(*v1beta1.ReleaseChannel)

			return oldObj.Spec != newObj.Spec
		},
		// Allow create events
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},

		// Allow delete events
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},

		// Allow generic events (e.g., external triggers)
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.ReleaseChannel{}, builder.WithPredicates(updatePred)).
		Complete(r)
}
