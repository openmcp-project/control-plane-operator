package schemes

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"

	crossplanev1 "github.com/crossplane/crossplane/apis/pkg/v1"
	crossplanev1beta1 "github.com/crossplane/crossplane/apis/pkg/v1beta1"
	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

var (
	// Local is the Scheme used when communicating with the cluster where the controller is running (Core).
	Local = runtime.NewScheme()

	// Remote is the Scheme used when communicating with the cluster where the workload is running (Target).
	Remote = runtime.NewScheme()
)

// initLocal adds types to Local scheme.
func initLocal() {
	// Standard go client types
	utilruntime.Must(clientgoscheme.AddToScheme(Local))

	// Controller types
	utilruntime.Must(v1beta1.AddToScheme(Local))

	// Flux CD
	utilruntime.Must(helmv2.AddToScheme(Local))
	utilruntime.Must(kustomizev1.AddToScheme(Local))
	utilruntime.Must(sourcev1.AddToScheme(Local))
}

// initRemote adds types to Remote scheme.
func initRemote() {
	// Standard go client types
	utilruntime.Must(clientgoscheme.AddToScheme(Remote))
	utilruntime.Must(apiextensionsv1.AddToScheme(Remote))

	// Controller types
	utilruntime.Must(v1beta1.AddToScheme(Remote))

	// Crossplane
	utilruntime.Must(crossplanev1.AddToScheme(Remote))
	utilruntime.Must(crossplanev1beta1.AddToScheme(Remote))
}

func init() {
	initLocal()
	initRemote()
}
