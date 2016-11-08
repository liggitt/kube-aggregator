package apiserver

import (
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/genericapiserver"
	"k8s.io/kubernetes/pkg/registry/generic"

	discoveryapi "github.com/openshift/kube-aggregator/pkg/api"
	discoveryapiv1beta1 "github.com/openshift/kube-aggregator/pkg/api/v1beta1"
	apiserverstorage "github.com/openshift/kube-aggregator/pkg/registry/apiserver"
)

type Config struct {
	GenericConfig *genericapiserver.Config
}

// APIDiscoveryServer contains state for a Kubernetes cluster master/api server.
type APIDiscoveryServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

type completedConfig struct {
	*Config
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *Config) Complete() completedConfig {
	c.GenericConfig.Complete()

	return completedConfig{c}
}

// SkipComplete provides a way to construct a server instance without config completion.
func (c *Config) SkipComplete() completedConfig {
	return completedConfig{c}
}

// New returns a new instance of APIDiscoveryServer from the given config.
func (c completedConfig) New() (*APIDiscoveryServer, error) {
	s, err := c.Config.GenericConfig.SkipComplete().New() // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}

	m := &APIDiscoveryServer{
		GenericAPIServer: s,
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(discoveryapi.GroupName)
	apiGroupInfo.GroupMeta.GroupVersion = discoveryapiv1beta1.SchemeGroupVersion

	v1beta1storage := map[string]rest.Storage{}
	v1beta1storage["apiservers"] = apiserverstorage.NewREST(generic.RESTOptions{})

	apiGroupInfo.VersionedResourcesStorageMap[discoveryapiv1beta1.SchemeGroupVersion.Version] = v1beta1storage

	if err := m.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	// m.GenericAPIServer.AddPostStartHook("start-informers", func(context genericapiserver.PostStartHookContext) error {
	// 	informerFactory.Start(wait.NeverStop)
	// 	return nil
	// })

	return m, nil
}
