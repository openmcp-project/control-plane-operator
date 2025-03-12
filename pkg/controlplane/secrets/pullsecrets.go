package secrets

import (
	"context"

	"github.com/openmcp-project/control-plane-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AvailablePullSecrets returns a list of secrets that are labeled with `constants.LabelCopyToCP`
// and are of type `kubernetes.io/dockerconfigjson`.
func AvailablePullSecrets(ctx context.Context, c client.Client) ([]types.NamespacedName, error) {
	pullSecrets := []types.NamespacedName{}

	matchingLabels := client.MatchingLabels{
		constants.LabelCopyToCP: "true",
	}
	secretList := &corev1.SecretList{}
	err := c.List(ctx, secretList, matchingLabels)
	if err != nil {
		return nil, err
	}

	for _, secret := range secretList.Items {
		if secret.Type == corev1.SecretTypeDockerConfigJson {
			pullSecrets = append(pullSecrets, client.ObjectKeyFromObject(&secret))
		}
	}

	return pullSecrets, nil
}
