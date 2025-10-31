package fluxcd

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
)

var (
	errNotFluxComponent = errors.New("not a flux component")
)

var _ juggler.ComponentReconciler = &FluxReconciler{}

func NewFluxReconciler(logger logr.Logger, localClient client.Client, remoteClient client.Client, labelComponentName string) *FluxReconciler {
	return &FluxReconciler{
		logger:       logger,
		localClient:  localClient,
		remoteClient: remoteClient,
		knownTypes:   sets.Set[reflect.Type]{},
		labelFunc:    juggler.DefaultLabelFunc(labelComponentName),
	}
}

func (r *FluxReconciler) WithLabelFunc(fn juggler.LabelFunc) *FluxReconciler {
	if fn != nil {
		r.labelFunc = fn
	}
	return r
}

type FluxReconciler struct {
	localClient  client.Client
	remoteClient client.Client
	logger       logr.Logger
	knownTypes   sets.Set[reflect.Type]
	labelFunc    juggler.LabelFunc
}

// KnownTypes implements juggler.ComponentReconciler.
func (r *FluxReconciler) KnownTypes() []reflect.Type {
	return r.knownTypes.UnsortedList()
}

func (r *FluxReconciler) RegisterType(comps ...FluxComponent) {
	for _, c := range comps {
		cType := reflect.TypeOf(c)
		r.knownTypes.Insert(cType)
	}
}

//nolint:lll
func (r *FluxReconciler) Observe(ctx context.Context, component juggler.Component) (juggler.ComponentObservation, error) {
	fluxComponent, ok := component.(FluxComponent)
	if !ok {
		return juggler.ComponentObservation{}, errNotFluxComponent
	}

	sourceObservation, errSource := r.observeSource(ctx, fluxComponent)
	if errSource != nil {
		return juggler.ComponentObservation{}, errSource
	}

	manifestoObservation, errManifest := r.observeManifesto(ctx, fluxComponent)
	if errManifest != nil {
		return juggler.ComponentObservation{}, errManifest
	}

	return juggler.ComponentObservation{
		ResourceExists:      sourceObservation.ResourceExists || manifestoObservation.ResourceExists,
		ResourceHealthiness: aggregateHealthiness(manifestoObservation.ResourceHealthiness, sourceObservation.ResourceHealthiness),
	}, nil
}

//nolint:lll
func (r *FluxReconciler) observeManifesto(ctx context.Context, fluxComponent FluxComponent) (juggler.ComponentObservation, error) {
	desiredManifesto, err := fluxComponent.BuildManifesto(ctx)
	if err != nil {
		return juggler.ComponentObservation{}, err
	}

	actualManifest := desiredManifesto.Empty()

	errGetM := r.localClient.Get(ctx, actualManifest.GetObjectKey(), actualManifest.GetObject())
	if apierrors.IsNotFound(errGetM) {
		// CAUTION: NotFound is not an error!!!
		return juggler.ComponentObservation{ResourceExists: false}, nil
	}

	return juggler.ComponentObservation{
		ResourceExists:      true,
		ResourceHealthiness: actualManifest.GetHealthiness(),
	}, nil
}

//nolint:lll
func (r *FluxReconciler) observeSource(ctx context.Context, fluxComponent FluxComponent) (juggler.ComponentObservation, error) {
	desiredSource, err := fluxComponent.BuildSourceRepository(ctx)
	if err != nil {
		return juggler.ComponentObservation{}, err
	}

	actualSource := desiredSource.Empty()

	errGetS := r.localClient.Get(ctx, actualSource.GetObjectKey(), actualSource.GetObject())
	if apierrors.IsNotFound(errGetS) {
		// CAUTION: NotFound is not an error!!!
		return juggler.ComponentObservation{ResourceExists: false}, nil
	}

	// resource exists, check if it is healthy, return observation
	return juggler.ComponentObservation{
		ResourceExists:      true,
		ResourceHealthiness: actualSource.GetHealthiness(),
	}, nil
}

func (r *FluxReconciler) Uninstall(ctx context.Context, component juggler.Component) error {
	fluxComponent, ok := component.(FluxComponent)
	if !ok {
		return errNotFluxComponent
	}

	errSource := r.deleteSource(ctx, fluxComponent)
	if errSource != nil {
		return errSource
	}

	return r.deleteManifesto(ctx, fluxComponent)
}

func (r *FluxReconciler) deleteSource(ctx context.Context, fluxComponent FluxComponent) error {
	desiredRepository, err := fluxComponent.BuildSourceRepository(ctx)
	if err != nil {
		return err
	}

	resourceRepository := desiredRepository.GetObject()

	err = r.localClient.Delete(ctx, resourceRepository)
	return client.IgnoreNotFound(err)
}

func (r *FluxReconciler) deleteManifesto(ctx context.Context, fluxComponent FluxComponent) error {
	desiredManifesto, err := fluxComponent.BuildManifesto(ctx)
	if err != nil {
		return err
	}

	resourceManifesto := desiredManifesto.GetObject()

	err = r.localClient.Delete(ctx, resourceManifesto)
	return client.IgnoreNotFound(err)
}

func (r *FluxReconciler) Update(ctx context.Context, component juggler.Component) error {
	return r.installOrUpdate(ctx, component)
}

func (r *FluxReconciler) Install(ctx context.Context, component juggler.Component) error {
	return r.installOrUpdate(ctx, component)
}

func (r *FluxReconciler) PreUninstall(ctx context.Context, component juggler.Component) error {
	if component.Hooks().PreUninstall != nil {
		return component.Hooks().PreUninstall(ctx, r.remoteClient)
	}
	return nil
}

// PreInstall implements juggler.ComponentReconciler.
func (r *FluxReconciler) PreInstall(ctx context.Context, component juggler.Component) error {
	if component.Hooks().PreInstall != nil {
		return component.Hooks().PreInstall(ctx, r.remoteClient)
	}
	return nil
}

// PreUpdate implements juggler.ComponentReconciler.
func (r *FluxReconciler) PreUpdate(ctx context.Context, component juggler.Component) error {
	if component.Hooks().PreUpdate != nil {
		return component.Hooks().PreUpdate(ctx, r.remoteClient)
	}
	return nil
}

func (r *FluxReconciler) installOrUpdate(ctx context.Context, component juggler.Component) error {
	fluxComponent, ok := component.(FluxComponent)
	if !ok {
		return errNotFluxComponent
	}

	errSource := r.installOrUpdateSource(ctx, fluxComponent)
	if errSource != nil {
		return errSource
	}

	return r.installOrUpdateManifesto(ctx, fluxComponent)
}

func (r *FluxReconciler) installOrUpdateManifesto(ctx context.Context, fluxComponent FluxComponent) error {
	desired, errMan := fluxComponent.BuildManifesto(ctx)
	if errMan != nil {
		return errMan
	}

	actual := desired.Empty()
	obj := actual.GetObject()
	result, errCU := controllerutil.CreateOrUpdate(ctx, r.localClient, obj, func() error {
		if err := actual.Reconcile(desired); err != nil {
			return err
		}
		utils.SetLabels(obj, r.labelFunc(fluxComponent))
		return nil
	})

	if errCU != nil {
		return errCU
	}

	r.logger.Info(fmt.Sprintf("%T %s/%s %s", obj, obj.GetNamespace(), obj.GetName(), result))

	return nil
}

func (r *FluxReconciler) installOrUpdateSource(ctx context.Context, fluxComponent FluxComponent) error {
	desired, err := fluxComponent.BuildSourceRepository(ctx)
	if err != nil {
		return err
	}

	actual := desired.Empty()
	obj := actual.GetObject()
	result, errCU := controllerutil.CreateOrUpdate(ctx, r.localClient, obj, func() error {
		if err := actual.Reconcile(desired); err != nil {
			return err
		}
		utils.SetLabels(obj, r.labelFunc(fluxComponent))
		return nil
	})

	if errCU != nil {
		return errCU
	}

	r.logger.Info(fmt.Sprintf("%T %s/%s %s", obj, obj.GetNamespace(), obj.GetName(), result))

	return nil
}

func aggregateHealthiness(states ...juggler.ResourceHealthiness) juggler.ResourceHealthiness {
	result := juggler.ResourceHealthiness{Healthy: true}

	for _, rh := range states {
		if !rh.Healthy {
			result.Healthy = false
			result.Message = fmt.Sprintf("%s\n%s", result.Message, rh.Message)
		}
	}

	result.Message = strings.TrimSpace(result.Message)
	return result
}
