package crds

import (
	"bytes"
	"context"
	"embed"
	"path"

	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/components"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterAsComponents(jug *juggler.Juggler, crdFiles embed.FS, enabled bool, names ...string) error {
	allCRDs, err := readAllCRDs(crdFiles)
	if err != nil {
		return err
	}

	for _, crd := range filterCRDs(allCRDs, names...) {
		comp := &components.GenericObjectComponent{
			NamespacedName: types.NamespacedName{
				Name: crd.Name,
			},
			NameOverride:  "CRD" + crd.Spec.Names.Kind,
			Type:          &apiextv1.CustomResourceDefinition{},
			Enabled:       enabled,
			KeepInstalled: true,
			IsObjectHealthyFunc: func(obj client.Object) juggler.ResourceHealthiness {
				actual := obj.(*apiextv1.CustomResourceDefinition)
				return juggler.ResourceHealthiness{
					// TODO: Choose something more meaningful?
					Healthy: len(actual.Status.StoredVersions) > 0,
				}
			},
			ReconcileObjectFunc: func(ctx context.Context, obj client.Object) error {
				actual := obj.(*apiextv1.CustomResourceDefinition)
				actual.Spec = crd.Spec
				return nil
			},
		}
		jug.RegisterComponent(comp)
	}
	return nil
}

func filterCRDs(input []*apiextv1.CustomResourceDefinition, names ...string) []*apiextv1.CustomResourceDefinition {
	out := []*apiextv1.CustomResourceDefinition{}
	for _, crd := range input {
		for _, name := range names {
			if crd.Name == name {
				out = append(out, crd)
			}
		}
	}
	return out
}

func readAllCRDs(crdFiles embed.FS) ([]*apiextv1.CustomResourceDefinition, error) {
	crds := []*apiextv1.CustomResourceDefinition{}

	yamls, err := readAllFiles(crdFiles, ".")
	if err != nil {
		return nil, err
	}
	for _, yaml := range yamls {
		crd, err := decodeCRD(yaml)
		if err != nil {
			return nil, err
		}
		crds = append(crds, crd)
	}

	return crds, nil
}

func decodeCRD(yamlBytes []byte) (*apiextv1.CustomResourceDefinition, error) {
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlBytes), 100)
	crd := &apiextv1.CustomResourceDefinition{}
	return crd, decoder.Decode(crd)
}

func readAllFiles(fs embed.FS, dir string) ([][]byte, error) {
	fileContents := [][]byte{}

	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		fullpath := path.Join(dir, entry.Name())
		if entry.IsDir() {
			subdirContents, err := readAllFiles(fs, fullpath)
			if err != nil {
				return nil, err
			}
			fileContents = append(fileContents, subdirContents...)
			continue
		}

		content, err := fs.ReadFile(fullpath)
		if err != nil {
			return nil, err
		}
		fileContents = append(fileContents, content)
	}

	return fileContents, nil
}
