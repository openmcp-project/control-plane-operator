//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/crossplane-contrib/xp-testing/pkg/envvar"
	"github.com/crossplane-contrib/xp-testing/pkg/logging"
	"github.com/crossplane-contrib/xp-testing/pkg/xpenvfuncs"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	crLog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

var (
	UutControllerKey = "openmcp-project/control-plane-operator"

	testEnv env.Environment
)

var (
	timeoutDeploymentsAvailable = time.Minute * 5
	timeoutDeploymentDeleted    = time.Minute * 4
)

func TestMain(m *testing.M) {
	var verbosity = 4
	logging.EnableVerboseLogging(&verbosity)
	crLog.SetLogger(klog.NewKlogr())

	// read environment variables
	uutImages := envvar.GetOrPanic("UUT_IMAGES")
	uutController := GetImagesFromJsonOrPanic(uutImages)

	pullSecretUser := envvar.GetOrPanic("PULL_SECRET_USER")
	pullSecretPassword := envvar.GetOrPanic("PULL_SECRET_PASSWORD")

	kindClusterName := envvar.GetOrDefault("CLUSTER_NAME", envconf.RandomName("mcp-e2e", 10))

	// create a new test environment and kind cluster
	testEnv = env.New()
	kindCluster := kind.NewCluster(kindClusterName)

	// Setup uses pre-defined funcs to create kind cluster
	testEnv.Setup(
		envfuncs.CreateCluster(kindCluster, kindClusterName),
		envfuncs.LoadDockerImageToCluster(kindClusterName, uutController),

		func(ctx context.Context, config *envconf.Config) (context.Context, error) {
			manager := helm.New(config.KubeconfigFile())

			// install the control plane operator via Helm
			err := manager.RunInstall(
				helm.WithName("flux2"),
				helm.WithChart("flux2"),
				helm.WithNamespace("flux-system"),
				helm.WithArgs("--create-namespace", "--repo", "https://fluxcd-community.github.io/helm-charts", "--version", "2.13.0"), //nolint:lll
			)
			if err != nil {
				_, errDump := xpenvfuncs.DumpLogs(kindClusterName, "flux-setup-err")(ctx, config)
				klog.Fatal(errDump)
			}

			return ctx, err
		},
		func(ctx context.Context, config *envconf.Config) (context.Context, error) {
			// create a new generic secret with pull secret for Flux
			err := exec.Command("kubectl", "create", "secret", "generic", "artifactory-readonly-basic", "--type=kubernetes.io/basic-auth", "--from-literal=username="+pullSecretUser, "--from-literal=password="+pullSecretPassword).Run() //nolint:lll
			if err != nil {
				return ctx, err
			}
			// pull secret has to be copied to every tenant namespace in order to get consumed by Flux
			err = exec.Command("kubectl", "label", "secret", "artifactory-readonly-basic", "core.orchestrate.cloud.sap/copy-to-cp-namespaces=true").Run() //nolint:lll
			if err != nil {
				return ctx, err
			}
			err = exec.Command("kubectl", "annotate", "secret", "artifactory-readonly-basic", "core.orchestrate.cloud.sap/credentials-for-url=https://common.repositories.cloud.sap/artifactory/api/helm/deploy-releases-hyperspace-helm").Run() //nolint:lll
			if err != nil {
				return ctx, err
			}

			return ctx, nil
		},
		func(ctx context.Context, config *envconf.Config) (context.Context, error) {
			manager := helm.New(config.KubeconfigFile())

			// split uutController in key and value
			uutControllerSplit := strings.Split(uutController, ":")

			err := exec.Command("kubectl", "create", "namespace", "co-system").Run() //nolint:lll
			if err != nil {
				return ctx, err
			}

			// Create new secret with ocm registry file for local overrides
			err = exec.Command("kubectl", "create", "secret", "generic", "ocm-registry", "-n", "co-system", "--from-file=ocm_registry.tgz=../testdata/ocm_registry.tgz").Run() //nolint:lll
			if err != nil {
				fmt.Println("Error creating secret ocm-registry")
				return ctx, err
			}

			// install the control plane operator via Helm
			err = manager.RunInstall(
				helm.WithName("co-control-plane-operator"),
				helm.WithChart("../../charts/control-plane-operator"),
				helm.WithNamespace("co-system"),
				helm.WithArgs(
					"--create-namespace",
					"--set", "image.repository="+uutControllerSplit[0],
					"--set", "image.tag="+uutControllerSplit[1],
					"--set", "image.pullPolicy=Never",
					"--set", "syncPeriod=10s", // Set the sync period here
					"-f", "testdata/values.yaml"), //nolint:lll
			)

			return ctx, err
		},

		waitForController("co-control-plane-operator", "co-system"),
		func(ctx context.Context, config *envconf.Config) (context.Context, error) {
			// Apply the releasechannel from the samples directory
			err := exec.Command("kubectl", "apply", "-f", "../../config/samples/releasechannel/local.yaml").Run()
			if err != nil {
				fmt.Println("Error applying releasechannel")
				return ctx, err
			}

			return ctx, nil
		},
	)

	testEnv.AfterEachTest(DumpLogsOnFail(kindClusterName))
	testEnv.Finish(xpenvfuncs.DumpLogs(kindClusterName, "post-test"))

	testEnv.Finish(envfuncs.DestroyCluster(kindClusterName))

	os.Exit(testEnv.Run(m))
}

func DumpLogsOnFail(kindClusterName string) func(context.Context, *envconf.Config, *testing.T) (context.Context, error) {
	return func(ctx context.Context, c *envconf.Config, t *testing.T) (context.Context, error) {
		if t.Failed() {
			xpenvfuncs.DumpLogs(kindClusterName, kindClusterName+t.Name())
		}
		return ctx, nil
	}
}

// GetImagesFromJsonOrPanic returns the UUT controller image from the UUT_IMAGES environment variable
func GetImagesFromJsonOrPanic(imagesJson string) string {
	imageMap := map[string]string{}

	err := json.Unmarshal([]byte(imagesJson), &imageMap)

	if err != nil {
		panic(errors.Wrap(err, "failed to unmarshal json from UUT_IMAGE"))
	}

	uutController := imageMap[UutControllerKey]

	return uutController
}

// waitForController waits for the controller to become available
func waitForController(name string, namespace string) env.Func {
	return func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		dep := v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		}
		// wait for the deployment to become at least 50%
		err := wait.For(conditions.New(c.Client().Resources()).ResourceMatch(&dep, func(object k8s.Object) bool {

			d := object.(*v1.Deployment)
			klog.Infof("Checking controller %s/%s to be available", namespace, namespace)
			return float64(d.Status.ReadyReplicas)/float64(*d.Spec.Replicas) >= 0.50
		}), wait.WithTimeout(timeoutDeploymentsAvailable))
		return ctx, err
	}
}
