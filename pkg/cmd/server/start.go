package server

import (
	"fmt"
	"io"

	"github.com/pborman/uuid"
	"github.com/spf13/cobra"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apimachinery/registered"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/genericapiserver"
	genericoptions "k8s.io/kubernetes/pkg/genericapiserver/options"
	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/registry/generic/registry"
	"k8s.io/kubernetes/pkg/storage/storagebackend"
	utilwait "k8s.io/kubernetes/pkg/util/wait"

	discoveryapi "github.com/openshift/kube-aggregator/pkg/apis/apifederation"
	"github.com/openshift/kube-aggregator/pkg/apiserver"
)

const defaultConfigDir = "openshift.local.config/kube-aggregator"
const defaultEtcdPathPrefix = "/registry/openshift.io/kube-aggregator"

type DiscoveryServerOptions struct {
	Etcd           *genericoptions.EtcdOptions
	SecureServing  *genericoptions.SecureServingOptions
	Authentication *genericoptions.DelegatingAuthenticationOptions
	Authorization  *genericoptions.DelegatingAuthorizationOptions

	AuthUser              string
	ProxyClientConfigFile string

	StdOut io.Writer
}

const startLong = `Start an API server hosting the apifederation.openshift.io API.`

// NewCommandStartMaster provides a CLI handler for 'start master' command
func NewCommandStartDiscoveryServer(out io.Writer) *cobra.Command {
	o := &DiscoveryServerOptions{
		Etcd:           genericoptions.NewEtcdOptions(),
		SecureServing:  genericoptions.NewSecureServingOptions(),
		Authentication: genericoptions.NewDelegatingAuthenticationOptions(),
		Authorization:  genericoptions.NewDelegatingAuthorizationOptions(),
		StdOut:         out,
	}
	o.Etcd.StorageConfig.Prefix = defaultEtcdPathPrefix
	o.Etcd.StorageConfig.Codec = kapi.Codecs.LegacyCodec(registered.EnabledVersionsForGroup(discoveryapi.GroupName)...)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Launch a discovery.openshift.io server",
		Long:  startLong,
		Run: func(c *cobra.Command, args []string) {
			fmt.Printf("Starting\n")

			kcmdutil.CheckErr(o.Complete())
			kcmdutil.CheckErr(o.Validate(args))
			kcmdutil.CheckErr(o.RunDiscoveryServer())
		},
	}

	flags := cmd.Flags()
	o.Etcd.AddFlags(flags)
	o.SecureServing.AddFlags(flags)
	o.Authentication.AddFlags(flags)
	o.Authorization.AddFlags(flags)
	flags.StringVar(&o.ProxyClientConfigFile, "proxy-kubeconfig", o.ProxyClientConfigFile, "kubeconfig file which will only use the `user` section for authenticating to backing servers.")
	flags.StringVar(&o.AuthUser, "auth-user", o.AuthUser, "username of the user used for delegating authentication and authorization.  Primes /bootstrap/rbac endpoint.")

	GLog(cmd.PersistentFlags())

	return cmd
}

func (o DiscoveryServerOptions) Validate(args []string) error {
	return nil
}

func (o *DiscoveryServerOptions) Complete() error {
	return nil
}

func (o DiscoveryServerOptions) RunDiscoveryServer() error {
	var err error
	genericAPIServerConfig := genericapiserver.NewConfig().ApplySecureServingOptions(o.SecureServing)
	if err := genericAPIServerConfig.MaybeGenerateServingCerts(); err != nil {
		return err
	}

	privilegedLoopbackToken := uuid.NewRandom().String()
	if genericAPIServerConfig.LoopbackClientConfig, err = genericoptions.NewSelfClientConfig(o.SecureServing, nil, privilegedLoopbackToken); err != nil {
		return err
	}

	authenticatorConfig, err := o.Authentication.ToAuthenticationConfig(o.SecureServing.ClientCA)
	if err != nil {
		return err
	}
	if genericAPIServerConfig.Authenticator, _, err = authenticatorConfig.New(); err != nil {
		return err
	}

	authorizerConfig, err := o.Authorization.ToAuthorizationConfig()
	if err != nil {
		return err
	}
	if genericAPIServerConfig.Authorizer, err = authorizerConfig.New(); err != nil {
		return err
	}

	// read the kubeconfig file to use for proxying requests
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = o.ProxyClientConfigFile
	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	clientConfig, err := loader.ClientConfig()
	if err != nil {
		return err
	}

	config := apiserver.Config{
		GenericConfig:     genericAPIServerConfig,
		RESTOptionsGetter: restOptionsFactory{storageConfig: &o.Etcd.StorageConfig},
		ProxyTLSConfig: restclient.TLSClientConfig{
			CertFile: clientConfig.TLSClientConfig.CertFile,
			KeyFile:  clientConfig.TLSClientConfig.KeyFile,
			CertData: clientConfig.TLSClientConfig.CertData,
			KeyData:  clientConfig.TLSClientConfig.KeyData,
		},
		AuthUser: o.AuthUser,
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}
	server.GenericAPIServer.PrepareRun().Run(utilwait.NeverStop)

	return nil
}

type restOptionsFactory struct {
	storageConfig *storagebackend.Config
}

func (f restOptionsFactory) NewFor(resource unversioned.GroupResource) generic.RESTOptions {
	return generic.RESTOptions{
		StorageConfig:           f.storageConfig,
		Decorator:               registry.StorageWithCacher,
		DeleteCollectionWorkers: 1,
		EnableGarbageCollection: false,
		ResourcePrefix:          f.storageConfig.Prefix + "/" + resource.Group + "/" + resource.Resource,
	}
}
