package server

import (
	"fmt"
	"io"
	"net"
	"path"
	"time"

	"github.com/golang/glog"
	"github.com/pborman/uuid"
	"github.com/spf13/cobra"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apimachinery/registered"
	apiserverauthenticator "k8s.io/kubernetes/pkg/apiserver/authenticator"
	"k8s.io/kubernetes/pkg/auth/authenticator"
	"k8s.io/kubernetes/pkg/auth/authenticator/bearertoken"
	"k8s.io/kubernetes/pkg/auth/group"
	"k8s.io/kubernetes/pkg/auth/user"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/genericapiserver"
	genericoptions "k8s.io/kubernetes/pkg/genericapiserver/options"
	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/registry/generic/registry"
	"k8s.io/kubernetes/pkg/storage/storagebackend"
	certutil "k8s.io/kubernetes/pkg/util/cert"
	utilwait "k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/plugin/pkg/auth/authenticator/request/anonymous"
	authenticationunion "k8s.io/kubernetes/plugin/pkg/auth/authenticator/request/union"
	"k8s.io/kubernetes/plugin/pkg/auth/authenticator/request/x509"
	authenticationwebhook "k8s.io/kubernetes/plugin/pkg/auth/authenticator/token/webhook"
	authorizationwebhook "k8s.io/kubernetes/plugin/pkg/auth/authorizer/webhook"

	discoveryapi "github.com/openshift/kube-aggregator/pkg/api"
	"github.com/openshift/kube-aggregator/pkg/apiserver"
)

const defaultConfigDir = "openshift.local.config/kube-aggregator"
const defaultEtcdPathPrefix = "/registry/openshift.io/kube-aggregator"

type DiscoveryServerOptions struct {
	StdOut io.Writer

	KubeConfig string
	ClientCA   string

	// we're only going to use the etcd options.  Well, at least to start.
	// once we refactor, we'll start making it easy to choose subsets of the options
	EtcdOptions *genericoptions.ServerRunOptions
}

const startLong = `Start an API server hosting the project.openshift.io API.`

// NewCommandStartMaster provides a CLI handler for 'start master' command
func NewCommandStartDiscoveryServer(out io.Writer) *cobra.Command {
	o := &DiscoveryServerOptions{
		StdOut:      out,
		EtcdOptions: genericoptions.NewServerRunOptions().WithEtcdOptions(),
	}
	o.EtcdOptions.StorageConfig.Prefix = defaultEtcdPathPrefix
	o.EtcdOptions.StorageConfig.Codec = kapi.Codecs.LegacyCodec(registered.EnabledVersionsForGroup(discoveryapi.GroupName)...)

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
	flags.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "Location of the master configuration file to run from. When running from a configuration file, all other command-line arguments are ignored.")
	flags.StringVar(&o.ClientCA, "client-ca-file", o.ClientCA, "If set, any request presenting a client certificate signed by one of the authorities in the client-ca-file is authenticated with an identity corresponding to the CommonName of the client certificate.")
	o.EtcdOptions.AddEtcdStorageFlags(flags)

	// autocompletion hints
	cmd.MarkFlagFilename("write-config")
	cmd.MarkFlagFilename("config", "yaml", "yml")

	GLog(cmd.PersistentFlags())

	return cmd
}

func (o DiscoveryServerOptions) Validate(args []string) error {
	return nil
}

func (o *DiscoveryServerOptions) Complete() error {
	return nil
}

// RunServer will eventually take the options and:
// 1.  Creates certs if needed
// 2.  Reads fully specified master config OR builds a fully specified master config from the args
// 3.  Writes the fully specified master config and exits if needed
// 4.  Starts the master based on the fully specified config
func (o DiscoveryServerOptions) RunDiscoveryServer() error {
	secureServingInfo := genericapiserver.SecureServingInfo{
		ServingInfo: genericapiserver.ServingInfo{
			BindAddress: net.JoinHostPort("0.0.0.0", "8444"),
		},
		ServerCert: genericapiserver.GeneratableKeyCert{
			Generate: true,
			CertKey: genericapiserver.CertKey{
				CertFile: path.Join(defaultConfigDir, "apiserver.crt"),
				KeyFile:  path.Join(defaultConfigDir, "apiserver.key"),
			},
		},
		ClientCA: o.ClientCA,
	}

	m := &DiscoveryServer{
		servingInfo:   secureServingInfo,
		kubeConfig:    o.KubeConfig,
		storageConfig: o.EtcdOptions.StorageConfig,
	}
	return m.Start()
}

// DiscoveryServer encapsulates starting the components of the master
type DiscoveryServer struct {
	// this should be part of the serializeable config
	servingInfo genericapiserver.SecureServingInfo
	kubeConfig  string

	storageConfig storagebackend.Config
}

// Start launches a master. It will error if possible, but some background processes may still
// be running and the process should exit after it finishes.
func (s *DiscoveryServer) Start() error {
	genericAPIServerConfig := genericapiserver.NewConfig().Complete()
	genericAPIServerConfig.SecureServingInfo = &s.servingInfo
	if err := genericAPIServerConfig.MaybeGenerateServingCerts(); err != nil {
		return err
	}

	privilegedLoopbackToken := uuid.NewRandom().String()
	selfClientConfig, err := newSelfClientConfig(s.servingInfo, privilegedLoopbackToken)
	if err != nil {
		glog.Fatalf("Failed to create clientset: %v", err)
	}
	genericAPIServerConfig.LoopbackClientConfig = selfClientConfig

	kubeClientConfig, err := clientcmd.
		NewNonInteractiveDeferredLoadingClientConfig(&clientcmd.ClientConfigLoadingRules{ExplicitPath: s.kubeConfig}, &clientcmd.ConfigOverrides{}).
		ClientConfig()
	if err != nil {
		return err
	}
	kubeclientset, err := internalclientset.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}
	genericAPIServerConfig.Authenticator, err = NewAuthenticator(privilegedLoopbackToken, s.servingInfo.ClientCA, kubeclientset)
	if err != nil {
		return err
	}
	genericAPIServerConfig.Authorizer, err = authorizationwebhook.NewFromInterface(kubeclientset.Authorization().SubjectAccessReviews(), 30*time.Second, 30*time.Second)
	if err != nil {
		return err
	}

	if err != nil {
		glog.Fatalf("error in initializing storage factory: %s", err)
	}

	config := apiserver.Config{
		GenericConfig:     genericAPIServerConfig.Config,
		RESTOptionsGetter: restOptionsFactory{storageConfig: &s.storageConfig},
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}
	server.GenericAPIServer.PrepareRun().Run(utilwait.NeverStop)
	return nil
}

func newSelfClientConfig(servingInfo genericapiserver.SecureServingInfo, token string) (*restclient.Config, error) {
	clientConfig := &restclient.Config{
		// Increase QPS limits. The client is currently passed to all admission plugins,
		// and those can be throttled in case of higher load on apiserver - see #22340 and #22422
		// for more details. Once #22422 is fixed, we may want to remove it.
		QPS:   50,
		Burst: 100,
	}

	host, port, err := net.SplitHostPort(servingInfo.ServingInfo.BindAddress)
	if err != nil {
		return nil, err
	}
	// TODO there's a way to get this via parsing
	if host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	clientConfig.Host = "https://" + net.JoinHostPort(host, port)
	// TODO this is incorrect, only works for self-signed
	clientConfig.CAFile = servingInfo.ServerCert.CertFile
	clientConfig.BearerToken = token

	return clientConfig, nil
}

func NewAuthenticator(privilegedLoopbackToken, clientCAFile string, clientset internalclientset.Interface) (authenticator.Request, error) {
	var uid = uuid.NewRandom().String()
	tokens := make(map[string]*user.DefaultInfo)
	tokens[privilegedLoopbackToken] = &user.DefaultInfo{
		Name:   user.APIServerUser,
		UID:    uid,
		Groups: []string{user.SystemPrivilegedGroup},
	}
	loopbackTokenAuthenticator := apiserverauthenticator.NewAuthenticatorFromTokens(tokens)

	certAuth, err := newAuthenticatorFromClientCAFile(clientCAFile)
	if err != nil {
		return nil, err
	}

	tokenChecker, err := authenticationwebhook.NewFromInterface(clientset.Authentication().TokenReviews(), 5*time.Minute)
	if err != nil {
		return nil, err
	}

	authenticator := authenticationunion.New(loopbackTokenAuthenticator, certAuth, bearertoken.New(tokenChecker))
	authenticator = group.NewGroupAdder(authenticator, []string{user.AllAuthenticated})

	// If the authenticator chain returns an error, return an error (don't consider a bad bearer token anonymous).
	authenticator = authenticationunion.NewFailOnError(authenticator, anonymous.NewAuthenticator())

	return authenticator, nil

}

func newAuthenticatorFromClientCAFile(clientCAFile string) (authenticator.Request, error) {
	roots, err := certutil.NewPool(clientCAFile)
	if err != nil {
		return nil, err
	}

	opts := x509.DefaultVerifyOptions()
	opts.Roots = roots

	return x509.New(opts, x509.CommonNameUserConversion), nil
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
