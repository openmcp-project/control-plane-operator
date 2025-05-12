package controller

import (
	"github.com/openmcp-project/controller-utils/pkg/clientconfig"
	"k8s.io/client-go/rest"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

type RemoteConfigBuilder func(target v1beta1.Target) (*rest.Config, clientconfig.ReloadFunc, error)

func NewRemoteConfigBuilder() RemoteConfigBuilder {
	return func(target v1beta1.Target) (*rest.Config, clientconfig.ReloadFunc, error) {
		return clientconfig.New(target.Target).GetRESTConfig()
	}
}
