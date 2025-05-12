package ocm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/ocm/extensions/accessmethods/helm"
	"ocm.software/ocm/api/ocm/extensions/accessmethods/ociartifact"
	"ocm.software/ocm/api/ocm/extensions/repositories/ctf"
	"ocm.software/ocm/api/ocm/extensions/repositories/ocireg"
	"ocm.software/ocm/api/ocm/ocmutils"
	"ocm.software/ocm/api/tech/oci/identity"
	"ocm.software/ocm/api/utils/accessobj"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

// Create an ocm.Repository entity out of an url specified in the ocmRegistry parameter.
// With the secret parameter you provide the necessary username and password for accessing the ocm Repository.
//
// Additionally the method returns a list of components in the second return parameter which are part of the ocm Repository.
// This can be filtered using the prefixFilter parameter.
func GetOCMRemoteRepo(ocmRegistry string, secret corev1.Secret, prefixFilter string) (ocm.Repository, []string, error) {
	var secretData RegistryCredentials
	// Decode the secret to get the username and password
	username, ok := secret.Data["username"]
	if !ok {
		return nil, nil, errors.New("secret does not contain username")
	}
	password, ok := secret.Data["password"]
	if !ok {
		return nil, nil, errors.New("secret does not contain password")
	}

	secretData.Username = string(username)
	secretData.Password = string(password)

	octx := ocm.DefaultContext()
	creds := identity.SimpleCredentials(secretData.Username, secretData.Password)

	spec := ocireg.NewRepositorySpec(ocmRegistry, nil)
	repo, err := octx.RepositoryForSpec(spec, creds)
	if err != nil {
		return nil, nil, errors.New("failed to get ocm registry with provided credentials")
	}

	// Currently workaround, as OCM does not support listing components for oci registries
	// Hopefully there will be something in the future
	// For now, we first get a list of all repositories in the oci registry and then use them as components
	registryHost := strings.Split(ocmRegistry, "/")[0]
	path := strings.TrimPrefix(ocmRegistry, registryHost+"/")

	prefix := path + "/component-descriptors/"
	repositories, err := GetRepositoriesInOCIRegistry(registryHost, secretData, prefix, "https")
	if err != nil {
		return nil, nil, errors.New("can't get repositories in OCI registry")
	}

	var components []string
	for _, repository := range repositories {
		if prefixFilter == "" || strings.HasPrefix(repository, prefixFilter) {
			components = append(components, repository)
		}
	}

	return repo, components, nil
}

// Create an ocm.Repository entity out a byte array of a tar based ocm registry.
//
// Additionally the method returns a list of components in the second return parameter which are part of the ocm Repository.
// This can be filtered using the prefixFilter parameter.
func GetOCMLocalRepo(ocmRegistry []byte, prefixFilter string) (ocm.Repository, []string, error) {
	f, err := os.CreateTemp("", "ocm-registry")
	if err != nil {
		return nil, nil, err
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			fmt.Printf("Failed to remove file %s: %v\n", name, err)
		}
	}(f.Name())

	// Save the ocmRegistry to a file
	_, err = f.Write(ocmRegistry)
	if err != nil {
		return nil, nil, err
	}

	octx := ocm.DefaultContext()
	fileRepo, err := ctf.NewRepositorySpec(accessobj.ACC_READONLY, f.Name())
	if err != nil {
		return nil, nil, err
	}

	repo, err := octx.RepositoryForSpec(fileRepo, nil)
	if err != nil {
		return nil, nil, err
	}

	components, err := repo.ComponentLister().GetComponents(prefixFilter, true)
	if err != nil {
		return nil, nil, err
	}

	return repo, components, nil
}

// This method takes a ocm repository and a list of components to create our Component struct out of it
// In that process, the ocm Repository will be searched for the specificed component names and
// their available versions will be put into the resulting component array.
//
// The prefixFilter must be specified as it will be cut off every componentName in the end
// so the resulting Component names are without it.
func GetOCMComponentsWithVersions(repo ocm.Repository, components []string, prefixFilter string) ([]v1beta1.Component, error) {
	octx := ocm.DefaultContext()

	var componentList = make([]v1beta1.Component, 0, len(components))
	for _, componentName := range components {
		component, _ := repo.LookupComponent(componentName)
		versions, _ := component.ListVersions()

		formattedComponentName := strings.TrimPrefix(componentName, prefixFilter)
		formattedComponentName = strings.TrimPrefix(formattedComponentName, "/")
		comp := v1beta1.Component{
			Name:     formattedComponentName,
			Versions: make([]v1beta1.ComponentVersion, 0),
		}

		for _, version := range versions {
			cva, err := component.LookupVersion(version)
			if err != nil {
				return nil, err
			}

			resources := cva.GetResources()
			access, err := resources[0].Access()
			if err != nil {
				return nil, err
			}

			switch access.GetKind() {
			case ociartifact.Type:
				ref, err := ocmutils.GetOCIArtifactRef(octx, resources[0])
				if err != nil {
					return nil, err
				}

				comp.Versions = append(comp.Versions, v1beta1.ComponentVersion{
					Version:   version,
					DockerRef: ref,
				})
			case helm.Type:
				accessSpec, ok := access.(*helm.AccessSpec)
				if !ok {
					return nil, errors.New("failed to get Helm repository reference")
				}

				chartname := strings.Split(accessSpec.HelmChart, ":")[0]
				comp.Versions = append(comp.Versions, v1beta1.ComponentVersion{
					Version:   version,
					HelmRepo:  accessSpec.HelmRepository,
					HelmChart: chartname,
				})
			default:
				return nil, errors.New("unsupported access method")
			}
		}

		componentList = append(componentList, comp)
	}

	return componentList, nil
}

// GetOCMComponent takes a component name and a version as input and searches in an OCM registry if the component
// with version is available. The function returns a Component object with the repository and version of the component.
func GetOCMComponent(
	ctx context.Context,
	client client.Client,
	componentName string,
	version string,
) (v1beta1.ComponentVersion, error) {
	releasechannels := v1beta1.ReleaseChannelList{}
	err := client.List(ctx, &releasechannels)
	if err != nil {
		return v1beta1.ComponentVersion{}, err
	}

	for _, releasechannel := range releasechannels.Items {
		components := releasechannel.Status.Components
		for _, component := range components {
			if component.Name == componentName {
				for _, componentVersion := range component.Versions {
					if componentVersion.Version == version {
						return componentVersion, nil
					}
				}
			}
		}
	}

	return v1beta1.ComponentVersion{}, fmt.Errorf("component %s with version %s not found", componentName, version)
}
