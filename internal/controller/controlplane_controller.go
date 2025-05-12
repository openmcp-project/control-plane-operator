/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"embed"
	"errors"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openmcp-project/control-plane-operator/internal/ocm"

	"github.com/openmcp-project/control-plane-operator/cmd/options"
	"github.com/openmcp-project/control-plane-operator/internal/schemes"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/components/clusterroles"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/components/crds"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/components/policies"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/crossplane"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/secretresolver"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/secrets"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/fluxcd"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/object"
	"github.com/openmcp-project/control-plane-operator/pkg/utils/rcontext"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	condApi "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1beta1 "github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/components"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/kubeconfiggen"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/targetrbac"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
)

const (
	requeueAfter      = 1 * time.Minute
	requeueAfterError = 5 * time.Second

	cpNamespacePrefix = "cp-"
	cpNamespaceMaxLen = 63
)

var (
	errComponentRemaining           = errors.New("at least one component is still installed")
	errFailedToCreateCPNamespace    = errors.New("failed to create namespace for ControlPlane")
	errFailedToBuildRESTConfig      = errors.New("failed to build REST config from ControlPlane target")
	errFailedToRemoteClient         = errors.New("failed to build client for ControlPlane target")
	errFailedToEnsureFluxKubeconfig = errors.New("failed to generate or save Flux kubeconfig")
	errFailedToApplyFluxRBAC        = errors.New("failed to apply Flux RBAC")

	secretTargetNamespaces = []string{
		components.CrossplaneNamespace,
	}

	embeddedCRDsToInstall = []string{
		"crossplanepackagerestrictions.core.orchestrate.cloud.sap",
	}
)

// ControlPlaneReconciler reconciles a ControlPlane object
type ControlPlaneReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	Recorder            record.EventRecorder
	Kubeconfiggen       kubeconfiggen.Generator
	FluxSecretResolver  secretresolver.SecretResolver
	WebhookMiddleware   types.NamespacedName
	ReconcilePeriod     time.Duration
	RemoteConfigBuilder RemoteConfigBuilder
	EmbeddedCRDs        embed.FS
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	newConditions := []metav1.Condition{}

	cp := &corev1beta1.ControlPlane{}
	if err := r.Get(ctx, req.NamespacedName, cp); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ControlPlane not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch ControlPlane")
		return ctrl.Result{}, err
	}

	namespace, err := r.ensureNamespace(ctx, cp)
	if err != nil {
		return ctrl.Result{}, errors.Join(errFailedToCreateCPNamespace, err)
	}
	ctx = rcontext.WithTenantNamespace(ctx, namespace)

	resolverFn := r.getReleaseChannels(ctx)
	ctx = rcontext.WithVersionResolver(ctx, resolverFn)
	ctx = rcontext.WithSecretRefResolver(ctx, r.FluxSecretResolver.Resolve)

	// get a remote config for the target cluster
	remoteCfg, _, err := r.RemoteConfigBuilder(cp.Spec.Target)
	if err != nil {
		return ctrl.Result{}, errors.Join(errFailedToBuildRESTConfig, err)
	}

	// create a remote client
	remoteClient, err := client.New(remoteCfg, client.Options{Scheme: schemes.Remote})
	if err != nil {
		return ctrl.Result{}, errors.Join(errFailedToRemoteClient, err)
	}

	// Flux kubeconfig and RBAC
	if err := targetrbac.Apply(ctx, remoteClient, cp.Spec.Target.FluxServiceAccount); err != nil {
		return ctrl.Result{}, errors.Join(errFailedToApplyFluxRBAC, err)
	}

	fluxKubeconfig, err := r.ensureKubeconfig(ctx, remoteCfg, namespace, "flux-kubeconfig", cp.Spec.Target.FluxServiceAccount)
	if err != nil {
		return ctrl.Result{}, errors.Join(errFailedToEnsureFluxKubeconfig, err)
	}
	ctx = rcontext.WithFluxKubeconfigRef(ctx, fluxKubeconfig)

	// Always update status
	defer func() {
		utils.UpdateConditions(&cp.Status.Conditions, newConditions)
		if err := r.Status().Update(ctx, cp); err != nil {
			log.Error(err, "failed to update status")
		}
	}()

	if !cp.DeletionTimestamp.IsZero() {
		return r.deleteControlPlane(ctx, cp, remoteClient, &newConditions)
	}

	if err := r.ensureFinalizer(ctx, cp); err != nil {
		return ctrl.Result{}, err
	}

	// update ControlPlane v1beta1.ComponentConfig
	conditions, err := r.updateControlPlaneComponents(ctx, cp, remoteClient)
	if err != nil {
		return ctrl.Result{}, err
	}

	// append collected conditions to Status
	for _, c := range conditions {
		condApi.SetStatusCondition(&newConditions, c)
	}

	// set status conditions Available
	condApi.SetStatusCondition(&newConditions, corev1beta1.Available())

	cp.Status.Namespace = namespace

	return ctrl.Result{RequeueAfter: r.ReconcilePeriod}, nil
}

// getReleaseChannels returns a function that can be used to resolve the version of a component
func (r *ControlPlaneReconciler) getReleaseChannels(ctx context.Context) corev1beta1.VersionResolverFn {
	return func(componentName string, version string) (corev1beta1.ComponentVersion, error) {
		return ocm.GetOCMComponent(ctx, r.Client, componentName, version)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1beta1.ControlPlane{}).
		Complete(r)
}

// updateControlPlaneComponents is the reconcile method where the v1beta1.ControlPlane components get reconciled
// by the components.Juggler. This function will return a list of Kubernetes conditions for the particular components.
func (r *ControlPlaneReconciler) updateControlPlaneComponents(ctx context.Context, cp *corev1beta1.ControlPlane, remoteClient client.Client) ([]metav1.Condition, error) {
	j, err := r.newJuggler(ctx, cp, remoteClient)
	if err != nil {
		return nil, err
	}
	result := j.Reconcile(ctx)

	enabledComponents := 0
	healthyComponents := 0
	conditions := []metav1.Condition{}
	for _, componentResult := range result {
		if componentResult.Component.IsEnabled() {
			enabledComponents++
		}
		if componentResult.Result == juggler.StatusHealthy {
			healthyComponents++
		}

		if !componentResult.Component.IsEnabled() && componentResult.Result == juggler.StatusDisabled {
			// Component is not enabled and has been successfully uninstalled (or has never been installed).
			// Don't output a condition in this case.
			continue
		}
		conditions = append(conditions, componentResult.ToCondition())
	}

	cp.Status.ComponentsEnabled = enabledComponents
	cp.Status.ComponentsHealthy = healthyComponents

	return conditions, nil
}

func (r *ControlPlaneReconciler) deleteControlPlane(ctx context.Context, cp *corev1beta1.ControlPlane, remoteClient client.Client, newConditions *[]metav1.Condition) (ctrl.Result, error) {
	if !r.hasFinalizer(cp) {
		return ctrl.Result{}, nil
	}

	log := log.FromContext(ctx)

	conditions, err := r.deleteControlPlaneComponents(ctx, cp, remoteClient)
	// append collected conditions to Status
	for _, c := range conditions {
		condApi.SetStatusCondition(newConditions, c)
	}
	if errors.Is(err, errComponentRemaining) {
		log.Info(err.Error())
		return ctrl.Result{RequeueAfter: requeueAfterError}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := targetrbac.Delete(ctx, remoteClient); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.removeFinalizer(ctx, cp); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ControlPlaneReconciler) deleteControlPlaneComponents(ctx context.Context, cp *corev1beta1.ControlPlane, remoteClient client.Client) ([]metav1.Condition, error) {
	// disable all components
	cpCopy := cp.DeepCopy()
	cpCopy.Spec.ComponentsConfig = corev1beta1.ComponentsConfig{}

	j, err := r.newJuggler(ctx, cpCopy, remoteClient)
	if err != nil {
		return nil, err
	}
	result := j.Reconcile(ctx)

	anyComponentRemaining := false
	for _, cr := range result {
		// do not count components that are marked as "keep on uninstall".
		if kou, ok := cr.Component.(juggler.KeepOnUninstall); ok && kou.KeepOnUninstall() {
			continue
		}
		// status must be "Disabled", otherwise the component is counted as "Remaining".
		if cr.Result != juggler.StatusDisabled {
			anyComponentRemaining = true
		}
	}

	conditions := []metav1.Condition{}
	for _, componentResult := range result {
		conditions = append(conditions, componentResult.ToCondition())
	}

	if anyComponentRemaining {
		return conditions, errComponentRemaining
	}

	return conditions, nil
}

func (r *ControlPlaneReconciler) newJuggler(ctx context.Context, cp *corev1beta1.ControlPlane, remoteClient client.Client) (*juggler.Juggler, error) {
	logger := log.FromContext(ctx)
	juggler := juggler.NewJuggler(logger, juggler.NewEventRecorder(r.Recorder, cp))

	secretsToCopy, err := r.addPullSecrets(ctx, cp)
	if err != nil {
		return nil, err
	}
	juggler.RegisterComponent(secretsToCopy...)

	// register Components that get installed on the target cluster
	cpComponents := r.controlPlaneComponents(cp)
	juggler.RegisterComponent(cpComponents...)

	// register ClusterRoles
	clusterroles.RegisterAsComponents(juggler, cpComponents, !cp.WasDeleted())

	// register CRDs
	if err := crds.RegisterAsComponents(juggler, r.EmbeddedCRDs, !cp.WasDeleted(), embeddedCRDsToInstall...); err != nil {
		return nil, err
	}

	// register policies
	if err := policies.RegisterAsComponents(juggler, r.Client, !cp.WasDeleted()); err != nil {
		return nil, err
	}

	if err := policies.RegisterDeploymentRuntimeConfigProtection(juggler, r.Client, options.IsDeploymentRuntimeConfigProtectionEnabled() && !cp.WasDeleted()); err != nil {
		return nil, err
	}

	r.registerReconcilers(juggler, logger, remoteClient)

	if err := juggler.RegisterOrphanedComponents(ctx); err != nil {
		return nil, err
	}

	return juggler, nil
}

func (r *ControlPlaneReconciler) registerReconcilers(juggler *juggler.Juggler, logger logr.Logger, remoteClient client.Client) {
	fr := fluxcd.NewFluxReconciler(logger, r.Client, remoteClient)
	fr.RegisterType(
		&components.BTPServiceOperator{},
		&components.CertManager{},
		&components.Crossplane{},
		&components.ExternalSecretsOperator{},
		&components.Flux{},
		&components.Kyverno{},
	)
	juggler.RegisterReconciler(fr)

	or := object.NewReconciler(logger, remoteClient)
	or.RegisterType(
		&components.ClusterRole{},
		&components.CrossplaneProvider{},
		&components.CrossplaneDeploymentRuntimeConfig{},
		&components.Secret{},
		&components.GenericObjectComponent{},
	)
	juggler.RegisterReconciler(or)
}

func (r *ControlPlaneReconciler) addPullSecrets(ctx context.Context, cp *corev1beta1.ControlPlane) ([]juggler.Component, error) {
	pullSecrets, err := secrets.AvailablePullSecrets(ctx, r.Client)
	if err != nil {
		return nil, err
	}

	comps := []juggler.Component{}
	for _, ps := range pullSecrets {
		for _, ns := range secretTargetNamespaces {
			// Add Secret component for each pull secret in every target namespace.
			comps = append(comps, &components.Secret{
				Enabled:      !cp.WasDeleted(),
				SourceClient: r.Client,
				Source:       ps,
				Target: types.NamespacedName{
					Name:      ps.Name,
					Namespace: ns,
				},
			})
		}

		// If Crossplane is enabled, add secret ref to the list of pull secrets.
		if cp.Spec.Crossplane != nil {
			for _, provider := range cp.Spec.Crossplane.Providers {
				provider.PackagePullSecrets = append(provider.PackagePullSecrets, corev1.LocalObjectReference{
					Name: ps.Name,
				})
			}
		}
	}

	return comps, nil
}

// controlPlaneComponents will extract the components from the v1beta1.ControlPlane spec that will be installed in the target cluster,
// so that the Juggler can reconcile them.
func (r *ControlPlaneReconciler) controlPlaneComponents(cp *corev1beta1.ControlPlane) []juggler.Component {
	comps := []juggler.Component{}
	xp := &components.Crossplane{
		Config: cp.Spec.Crossplane,
	}
	comps = append(comps, xp)
	if cp.Spec.Crossplane != nil {
		for _, provider := range cp.Spec.Crossplane.Providers {
			comps = append(comps, &components.CrossplaneProvider{
				Config:  provider,
				Enabled: xp.IsEnabled(),
			})
			comps = append(comps, &components.CrossplaneDeploymentRuntimeConfig{
				Name:    crossplane.DeploymentRuntimeNameForProviderConfig(provider),
				Enabled: xp.IsEnabled(),
			})
		}
	}
	comps = append(comps, &components.CertManager{
		Config: cp.Spec.CertManager,
	})
	comps = append(comps, &components.BTPServiceOperator{
		Config: cp.Spec.BTPServiceOperator,
	})
	comps = append(comps, &components.ExternalSecretsOperator{
		Config: cp.Spec.ExternalSecretsOperator,
	})
	comps = append(comps, &components.Kyverno{
		Config: cp.Spec.Kyverno,
	})
	comps = append(comps, &components.Flux{
		Config: cp.Spec.Flux,
	})
	return comps
}

func (r *ControlPlaneReconciler) ensureNamespace(ctx context.Context, cp *corev1beta1.ControlPlane) (string, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: shortenToXCharacters(cpNamespacePrefix+cp.Name, cpNamespaceMaxLen),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
		utils.SetLabel(ns, corev1beta1.LabelControlPlane, cp.Name)
		utils.SetManagedBy(ns)
		return controllerutil.SetOwnerReference(cp, ns, r.Scheme)
	})
	return ns.Name, err
}

func (r *ControlPlaneReconciler) ensureFinalizer(ctx context.Context, cp *corev1beta1.ControlPlane) error {
	updated := controllerutil.AddFinalizer(cp, corev1beta1.Finalizer)
	if updated {
		return r.Update(ctx, cp)
	}
	return nil
}

func (r *ControlPlaneReconciler) removeFinalizer(ctx context.Context, cp *corev1beta1.ControlPlane) error {
	updated := controllerutil.RemoveFinalizer(cp, corev1beta1.Finalizer)
	if updated {
		return r.Update(ctx, cp)
	}
	return nil
}

func (r *ControlPlaneReconciler) hasFinalizer(cp *corev1beta1.ControlPlane) bool {
	return controllerutil.ContainsFinalizer(cp, corev1beta1.Finalizer)
}
