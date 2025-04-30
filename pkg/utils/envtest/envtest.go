package envtest

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	errMakefileIsDir           = errors.New("expected Makefile to be a file but it is a directory")
	errFailedToGetWD           = errors.New("failed to get working directory")
	errFailedToFindMakefile    = errors.New("failed to find Makefile")
	errFailedToRunMake         = errors.New("failed to run make")
	errFailedToRunSetupEnvtest = errors.New("failed to run setup-envtest")
	errMakefileNotFound        = errors.New("reached fs root and did not find Makefile")
	errFailedToReadMakefile    = errors.New("failed to read Makefile")
	errK8sVersionNotFound      = errors.New("value of ENVTEST_K8S_VERSION not found")

	k8sVersionRegexp = regexp.MustCompile(`ENVTEST_K8S_VERSION\s*=\s*(.+)\n`)

	k8sVersionEnvName = "ENVTEST_K8S_VERSION"
)

// Install uses make to install the envtest dependencies and sets the
// KUBEBUILDER_ASSETS environment variable.
func Install() error {
	wd, err := os.Getwd()
	if err != nil {
		return errors.Join(errFailedToGetWD, err)
	}

	makefilePath, err := findMakefile(wd)
	if err != nil {
		return errors.Join(errFailedToFindMakefile, err)
	}
	repoDir := filepath.Dir(makefilePath)

	if err := runMakeEnvtest(repoDir); err != nil {
		return err
	}

	assetsDir, err := runSetupEnvtest(repoDir, makefilePath)
	if err != nil {
		return err
	}

	return os.Setenv("KUBEBUILDER_ASSETS", assetsDir)
}

func runMakeEnvtest(repoDir string) error {
	cmd := exec.Command("make", "envtest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		return errors.Join(errFailedToRunMake, err)
	}
	return nil
}

func runSetupEnvtest(repoDir, makefilePath string) (string, error) {
	k8sVersion, err := readK8sVersion(makefilePath)
	if err != nil {
		return "", err
	}

	binDir := filepath.Join(repoDir, "bin")
	binary := filepath.Join(binDir, "setup-envtest")
	cmd := exec.Command(binary, "use", k8sVersion, "--bin-dir", binDir, "-p", "path")
	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		return "", errors.Join(errFailedToRunSetupEnvtest, err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func findMakefile(root string) (string, error) {
	if !filepath.IsAbs(root) {
		var err error
		if root, err = filepath.Abs(root); err != nil {
			return "", err
		}
	}

	if root == "/" {
		return "", errMakefileNotFound
	}

	makefilePath := filepath.Join(root, "Makefile")
	finfo, err := os.Stat(makefilePath)
	if errors.Is(err, fs.ErrNotExist) {
		parent := filepath.Dir(root)
		return findMakefile(parent)
	}
	if err != nil {
		return "", err
	}
	if finfo.IsDir() {
		return "", errMakefileIsDir
	}

	return makefilePath, nil
}

func readK8sVersion(makefilePath string) (string, error) {
	// try to read environment variables first.
	version := os.Getenv("ENVTEST_K8S_VERSION")
	if len(version) != 0 {
		return version, nil
	}
	// fall back to reading the makefile!
	bytes, err := os.ReadFile(makefilePath)
	if err != nil {
		return "", errors.Join(errFailedToReadMakefile, err)
	}

	match := k8sVersionRegexp.FindSubmatch(bytes)
	if match == nil || len(match) != 2 {
		return "", errK8sVersionNotFound
	}

	return string(match[1]), nil
}
