package components

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/pkg/constants"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/object"
	"github.com/openmcp-project/control-plane-operator/pkg/utils"
)

var (
	ErrSecretTargetNamespaceEmpty = errors.New("secret target namespace must not be empty")
)

var _ object.ObjectComponent = &Secret{}
var _ object.OrphanedObjectsDetector = &Secret{}
var _ TargetComponent = &Secret{}
var _ juggler.StatusVisibility = &Secret{}

type Secret struct {
	SourceClient   client.Client
	Source, Target types.NamespacedName
	Enabled        bool
}

// BuildObjectToReconcile implements object.ObjectComponent.
func (s *Secret) BuildObjectToReconcile(ctx context.Context) (client.Object, types.NamespacedName, error) {
	if s.Target.Namespace == "" {
		return nil, types.NamespacedName{}, ErrSecretTargetNamespaceEmpty
	}

	return &corev1.Secret{}, types.NamespacedName{
		Name:      s.Target.Name,
		Namespace: s.Target.Namespace,
	}, nil
}

// ReconcileObject implements object.ObjectComponent.
func (s *Secret) ReconcileObject(ctx context.Context, obj client.Object) error {
	sourceSecret := &corev1.Secret{}
	// If secret is not enabled (= should be deleted), then we don't need to get it from the API server.
	if s.Enabled {
		if err := s.SourceClient.Get(ctx, s.Source, sourceSecret); err != nil {
			return err
		}
	}

	objSecret := obj.(*corev1.Secret)

	metav1.SetMetaDataLabel(&objSecret.ObjectMeta, constants.LabelCopySourceName, s.Source.Name)
	metav1.SetMetaDataLabel(&objSecret.ObjectMeta, constants.LabelCopySourceNamespace, s.Source.Namespace)

	objSecret.Type = sourceSecret.Type
	objSecret.Data = sourceSecret.Data
	return nil
}

// OrphanDetectorContext implements object.OrphanedObjectsDetector.
func (*Secret) OrphanDetectorContext() object.DetectorContext {
	return object.DetectorContext{
		ListType: &corev1.SecretList{},
		FilterCriteria: object.FilterCriteria{
			utils.IsManaged(),
			object.HasComponentLabel(),
		},
		ConvertFunc: func(list client.ObjectList) []juggler.Component {
			secrets := []juggler.Component{}
			for _, secret := range (list.(*corev1.SecretList)).Items {
				secrets = append(secrets, &Secret{Target: client.ObjectKeyFromObject(&secret)})
			}
			return secrets
		},
		SameFunc: func(configured, detected juggler.Component) bool {
			configuredS := configured.(*Secret)
			detectedS := detected.(*Secret)
			return configuredS.Target == detectedS.Target
		},
	}
}

// GetDependencies implements object.ObjectComponent.
func (s *Secret) GetDependencies() []juggler.Component {
	return []juggler.Component{}
}

// GetName implements object.ObjectComponent.
func (s *Secret) GetName() string {
	return formatSecretName(s.Target.Name)
}

// Hooks implements object.ObjectComponent.
func (s *Secret) Hooks() juggler.ComponentHooks {
	return juggler.ComponentHooks{}
}

func (s *Secret) IsInstallable(_ context.Context) (bool, error) {
	return true, nil
}

// IsEnabled implements object.ObjectComponent.
func (s *Secret) IsEnabled() bool {
	return s.Enabled
}

// IsObjectHealthy implements object.ObjectComponent.
func (s *Secret) IsObjectHealthy(obj client.Object) juggler.ResourceHealthiness {
	return juggler.ResourceHealthiness{
		// Secret has no status field.
		Healthy: obj.GetDeletionTimestamp() == nil,
	}
}

// GetNamespace implements TargetComponent.
func (s *Secret) GetNamespace() string {
	return s.Target.Namespace
}

func formatSecretName(name string) string {
	parts := strings.Split(name, "-")
	for i, part := range parts {
		parts[i] = cases.Title(language.English).String(part)
	}
	parts = append([]string{"Secret"}, parts...)
	return strings.Join(parts, "")
}

// IsStatusInternal implements StatusVisibility interface.
func (s *Secret) IsStatusInternal() bool {
	return true
}
