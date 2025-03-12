package utils

import (
	"os"
)

const localOCMRegistryTestDataPath = "../../test/testdata/ocm_registry.tgz"

type Path string

const (
	LocalOCMRepositoryPathValid Path = localOCMRegistryTestDataPath
	RepositoryPathInvalid       Path = "invalid/path"
	OCMRepositoryPathKey             = "LOCAL_OCM_REPOSITORY_PATH"
)

func SetEnvironmentVariableForLocalOCMTar(path Path) error {
	return os.Setenv(OCMRepositoryPathKey, string(path))
}
