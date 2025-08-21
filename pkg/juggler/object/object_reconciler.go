package object

import (
	"context"
	"errors"
	"reflect"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
)

var (
	errNotObjectComponent = errors.New("not an object component")
)

var _ juggler.ComponentReconciler = &ObjectReconciler{}
var _ juggler.OrphanedComponentsDetector = &ObjectReconciler{}

func NewReconciler(logger logr.Logger, remoteClient client.Client, labelComponentName string) *ObjectReconciler {
	return &ObjectReconciler{
		logger:             logger,
		remoteClient:       remoteClient,
		knownTypes:         sets.Set[reflect.Type]{},
		labelComponentName: labelComponentName,
	}
}

type ObjectReconciler struct {
	logger             logr.Logger
	remoteClient       client.Client
	knownTypes         sets.Set[reflect.Type]
	labelComponentName string
}

// DetectOrphanedComponents implements juggler.OrphanedComponentsDetector.
func (r *ObjectReconciler) DetectOrphanedComponents(
	ctx context.Context,
	configuredComponents []juggler.Component,
) ([]juggler.Component, error) {
	orphaned := []juggler.Component{}
	for _, kt := range r.KnownTypes() {
		newComp := reflect.New(kt).Elem().Interface()
		if ood, ok := newComp.(OrphanedObjectsDetector); ok {
			dc := ood.OrphanDetectorContext()
			filtered, err := r.filterOrphanedObjects(ctx, configuredComponents, dc)
			if err != nil {
				return nil, err
			}
			orphaned = append(orphaned, filtered...)
		}
	}
	return orphaned, nil
}

// filterOrphanedObjects uses the `DetectorContext` to search for possibly orphaned objects, convert them
// to components and compare those to configured components to decide which of them are orphaned.
func (r *ObjectReconciler) filterOrphanedObjects(
	ctx context.Context,
	configuredComponents []juggler.Component,
	dc DetectorContext,
) ([]juggler.Component, error) {
	err := r.remoteClient.List(ctx, dc.ListType, dc.FilterCriteria...)
	if utils.IsCRDNotFound(err) {
		// CRD not installed, so there can't be any orphaned resources of this type.
		return []juggler.Component{}, nil
	}
	if err != nil {
		return nil, err
	}
	possiblyOrphanedComps := dc.ConvertFunc(dc.ListType)
	orphaned := []juggler.Component{}
	for _, possiblyOrphaned := range possiblyOrphanedComps {
		found := false
		for _, configured := range configuredComponents {
			// only try to match component of the same type e.g. CrossplaneProvider <=> CrossplaneProvider
			if reflect.TypeOf(configured) != reflect.TypeOf(possiblyOrphaned) {
				continue
			}
			if dc.SameFunc(configured, possiblyOrphaned) {
				found = true
			}
		}
		if !found {
			orphaned = append(orphaned, possiblyOrphaned)
		}
	}
	return orphaned, nil
}

// KnownTypes implements juggler.ComponentReconciler.
func (r *ObjectReconciler) KnownTypes() []reflect.Type {
	return r.knownTypes.UnsortedList()
}

func (r *ObjectReconciler) RegisterType(comps ...ObjectComponent) {
	for _, c := range comps {
		cType := reflect.TypeOf(c)
		r.knownTypes.Insert(cType)
	}
}

// Install implements ComponentReconciler.
func (r *ObjectReconciler) Install(ctx context.Context, component juggler.Component) error {
	return r.applyObject(ctx, component)
}

// Observe implements ComponentReconciler.
func (r *ObjectReconciler) Observe(ctx context.Context, comp juggler.Component) (juggler.ComponentObservation, error) {
	objectComponent, ok := comp.(ObjectComponent)
	if !ok {
		return juggler.ComponentObservation{}, errNotObjectComponent
	}

	obj, key, err := objectComponent.BuildObjectToReconcile(ctx)
	if err != nil {
		return juggler.ComponentObservation{}, err
	}

	err = r.remoteClient.Get(ctx, key, obj)
	if apierrors.IsNotFound(err) || utils.IsCRDNotFound(err) {
		r.logger.Info("Object not found")
		return juggler.ComponentObservation{ResourceExists: false}, nil
	}
	if err != nil {
		return juggler.ComponentObservation{}, err
	}

	return juggler.ComponentObservation{
		ResourceExists:      true,
		ResourceHealthiness: objectComponent.IsObjectHealthy(obj),
	}, nil
}

// PreUninstall implements ComponentReconciler.
func (r *ObjectReconciler) PreUninstall(ctx context.Context, component juggler.Component) error {
	if component.Hooks().PreUninstall != nil {
		return component.Hooks().PreUninstall(ctx, r.remoteClient)
	}
	return nil
}

// PreInstall implements juggler.ComponentReconciler.
func (r *ObjectReconciler) PreInstall(ctx context.Context, component juggler.Component) error {
	if component.Hooks().PreInstall != nil {
		return component.Hooks().PreInstall(ctx, r.remoteClient)
	}
	return nil
}

// PreUpdate implements juggler.ComponentReconciler.
func (r *ObjectReconciler) PreUpdate(ctx context.Context, component juggler.Component) error {
	if component.Hooks().PreUpdate != nil {
		return component.Hooks().PreUpdate(ctx, r.remoteClient)
	}
	return nil
}

// Uninstall implements ComponentReconciler.
func (r *ObjectReconciler) Uninstall(ctx context.Context, component juggler.Component) error {
	objectComponent, ok := component.(ObjectComponent)
	if !ok {
		return errNotObjectComponent
	}

	obj, key, err := objectComponent.BuildObjectToReconcile(ctx)
	if err != nil {
		return err
	}
	obj.SetName(key.Name)
	obj.SetNamespace(key.Namespace)

	return client.IgnoreNotFound(r.remoteClient.Delete(ctx, obj))
}

// Update implements ComponentReconciler.
func (r *ObjectReconciler) Update(ctx context.Context, component juggler.Component) error {
	return r.applyObject(ctx, component)
}

func (r *ObjectReconciler) applyObject(ctx context.Context, component juggler.Component) error {
	objectComponent, ok := component.(ObjectComponent)
	if !ok {
		return errNotObjectComponent
	}

	obj, key, err := objectComponent.BuildObjectToReconcile(ctx)
	if err != nil {
		return err
	}
	obj.SetName(key.Name)
	obj.SetNamespace(key.Namespace)

	_, err = controllerutil.CreateOrUpdate(ctx, r.remoteClient, obj, func() error {
		utils.SetManagedBy(obj)
		utils.SetLabel(obj, r.labelComponentName, component.GetName())
		return objectComponent.ReconcileObject(ctx, obj)
	})
	return err
}
