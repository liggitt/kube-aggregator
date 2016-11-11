/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package genericapiserver

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/spec"
	"github.com/golang/glog"
	"github.com/pborman/uuid"
	"gopkg.in/natefinch/lumberjack.v2"

	"k8s.io/kubernetes/pkg/admission"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	apiserverauthenticator "k8s.io/kubernetes/pkg/apiserver/authenticator"
	apiserverfilters "k8s.io/kubernetes/pkg/apiserver/filters"
	apiserveropenapi "k8s.io/kubernetes/pkg/apiserver/openapi"
	"k8s.io/kubernetes/pkg/apiserver/request"
	"k8s.io/kubernetes/pkg/auth/authenticator"
	"k8s.io/kubernetes/pkg/auth/authorizer"
	authorizerunion "k8s.io/kubernetes/pkg/auth/authorizer/union"
	authhandlers "k8s.io/kubernetes/pkg/auth/handlers"
	"k8s.io/kubernetes/pkg/auth/user"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/cloudprovider"
	apiserverauthorizer "k8s.io/kubernetes/pkg/genericapiserver/authorizer"
	genericfilters "k8s.io/kubernetes/pkg/genericapiserver/filters"
	"k8s.io/kubernetes/pkg/genericapiserver/mux"
	"k8s.io/kubernetes/pkg/genericapiserver/openapi/common"
	"k8s.io/kubernetes/pkg/genericapiserver/options"
	"k8s.io/kubernetes/pkg/genericapiserver/routes"
	genericvalidation "k8s.io/kubernetes/pkg/genericapiserver/validation"
	"k8s.io/kubernetes/pkg/runtime"
	certutil "k8s.io/kubernetes/pkg/util/cert"
	"k8s.io/kubernetes/pkg/util/sets"
	"k8s.io/kubernetes/pkg/version"
	authenticatorunion "k8s.io/kubernetes/plugin/pkg/auth/authenticator/request/union"
)

const (
	// DefaultLegacyAPIPrefix is where the the legacy APIs will be located.
	DefaultLegacyAPIPrefix = "/api"

	// APIGroupPrefix is where non-legacy API group will be located.
	APIGroupPrefix = "/apis"
)

// Config is a structure used to configure a GenericAPIServer.
type Config struct {
	// Destination for audit logs
	AuditWriter io.Writer
	// Allow downstream consumers to disable swagger.
	// This includes returning the generated swagger spec at /swaggerapi and swagger ui at /swagger-ui.
	EnableSwaggerSupport bool
	// Allow downstream consumers to disable swagger ui.
	// Note that this is ignored if EnableSwaggerSupport is false
	EnableSwaggerUI bool
	// Allows api group versions or specific resources to be conditionally enabled/disabled.
	APIResourceConfigSource APIResourceConfigSource
	// allow downstream consumers to disable the index route
	EnableIndex             bool
	EnableProfiling         bool
	EnableMetrics           bool
	EnableGarbageCollection bool

	Version               *version.Info
	CorsAllowedOriginList []string
	Authenticator         authenticator.Request
	// TODO(roberthbailey): Remove once the server no longer supports http basic auth.
	SupportsBasicAuth bool
	Authorizer        authorizer.Authorizer
	AdmissionControl  admission.Interface
	// TODO(ericchiang): Determine if policy escalation checks should be an admission controller.
	AuthorizerRBACSuperUser string

	// LoopbackClientConfig is a config for a privileged loopback connection to the API server
	LoopbackClientConfig *restclient.Config

	// Map requests to contexts. Exported so downstream consumers can provider their own mappers
	RequestContextMapper api.RequestContextMapper

	// Required, the interface for serializing and converting objects to and from the wire
	Serializer runtime.NegotiatedSerializer

	// If specified, requests will be allocated a random timeout between this value, and twice this value.
	// Note that it is up to the request handlers to ignore or honor this timeout. In seconds.
	MinRequestTimeout int

	SecureServingInfo   *SecureServingInfo
	InsecureServingInfo *ServingInfo

	// DiscoveryAddresses is used to build the IPs pass to discovery.  If nil, the ExternalAddress is
	// always reported
	DiscoveryAddresses DiscoveryAddresses

	// The port on PublicAddress where a read-write server will be installed.
	// Defaults to 6443 if not set.
	ReadWritePort int

	// ExternalAddress is the host name to use for external (public internet) facing URLs (e.g. Swagger)
	ExternalAddress string

	// PublicAddress is the IP address where members of the cluster (kubelet,
	// kube-proxy, services, etc.) can reach the GenericAPIServer.
	// If nil or 0.0.0.0, the host's default interface will be used.
	PublicAddress net.IP

	// EnableOpenAPISupport enables OpenAPI support. Allow downstream customers to disable OpenAPI spec.
	EnableOpenAPISupport bool

	// OpenAPIConfig will be used in generating OpenAPI spec.
	OpenAPIConfig *common.Config

	// MaxRequestsInFlight is the maximum number of parallel non-long-running requests. Every further
	// request has to wait.
	MaxRequestsInFlight int

	// Predicate which is true for paths of long-running http requests
	LongRunningFunc genericfilters.LongRunningRequestCheck

	// Build the handler chains by decorating the apiHandler.
	BuildHandlerChainsFunc func(apiHandler http.Handler, c *Config) (secure, insecure http.Handler)

	// LegacyAPIGroupPrefixes is used to set up URL parsing for authorization and for validating requests
	// to InstallLegacyAPIGroup
	LegacyAPIGroupPrefixes sets.String
}

type ServingInfo struct {
	// BindAddress is the ip:port to serve on
	BindAddress string
	// BindNetwork is the type of network to bind to - defaults to "tcp", accepts "tcp",
	// "tcp4", and "tcp6".
	BindNetwork string
}

type SecureServingInfo struct {
	ServingInfo

	// ServerCert is the TLS cert info for serving secure traffic
	ServerCert GeneratableKeyCert
	// SNICerts are named CertKeys for serving secure traffic with SNI support.
	SNICerts []NamedCertKey
	// ClientCA is the certificate bundle for all the signers that you'll recognize for incoming client certificates
	ClientCA string
}

type CertKey struct {
	// CertFile is a file containing a PEM-encoded certificate
	CertFile string
	// KeyFile is a file containing a PEM-encoded private key for the certificate specified by CertFile
	KeyFile string
}

type NamedCertKey struct {
	CertKey

	// Names is a list of domain patterns: fully qualified domain names, possibly prefixed with
	// wildcard segments.
	Names []string
}

type GeneratableKeyCert struct {
	CertKey
	// Generate indicates that the cert/key pair should be generated if its not present.
	Generate bool
}

// NewConfig returns a Config struct with the default values
func NewConfig() *Config {
	longRunningRE := regexp.MustCompile(options.DefaultLongRunningRequestRE)

	config := &Config{
		Serializer:             api.Codecs,
		ReadWritePort:          6443,
		RequestContextMapper:   api.NewRequestContextMapper(),
		BuildHandlerChainsFunc: DefaultBuildHandlerChain,
		LegacyAPIGroupPrefixes: sets.NewString(DefaultLegacyAPIPrefix),

		EnableIndex:          true,
		EnableSwaggerSupport: true,
		OpenAPIConfig: &common.Config{
			ProtocolList:   []string{"https"},
			IgnorePrefixes: []string{"/swaggerapi"},
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title:   "Generic API Server",
					Version: "unversioned",
				},
			},
			DefaultResponse: &spec.Response{
				ResponseProps: spec.ResponseProps{
					Description: "Default Response.",
				},
			},
			GetOperationIDAndTags: apiserveropenapi.GetOperationIDAndTags,
		},
		LongRunningFunc: genericfilters.BasicLongRunningRequestCheck(longRunningRE, map[string]string{"watch": "true"}),
	}

	// this keeps the defaults in sync
	defaultOptions := options.NewServerRunOptions()
	// unset fields that can be overridden to avoid setting values so that we won't end up with lingering values.
	// TODO we probably want to run the defaults the other way.  A default here drives it in the CLI flags
	defaultOptions.AuditLogPath = ""
	return config.ApplyOptions(defaultOptions)
}

func (c *Config) ApplySecureServingOptions(secureServing *options.SecureServingOptions) *Config {
	if secureServing == nil || secureServing.ServingOptions.BindPort <= 0 {
		return c
	}

	secureServingInfo := &SecureServingInfo{
		ServingInfo: ServingInfo{
			BindAddress: net.JoinHostPort(secureServing.ServingOptions.BindAddress.String(), strconv.Itoa(secureServing.ServingOptions.BindPort)),
		},
		ServerCert: GeneratableKeyCert{
			CertKey: CertKey{
				CertFile: secureServing.ServerCert.CertKey.CertFile,
				KeyFile:  secureServing.ServerCert.CertKey.KeyFile,
			},
		},
		SNICerts: []NamedCertKey{},
		ClientCA: secureServing.ClientCA,
	}
	if secureServing.ServerCert.CertKey.CertFile == "" && secureServing.ServerCert.CertKey.KeyFile == "" {
		secureServingInfo.ServerCert.Generate = true
		secureServingInfo.ServerCert.CertFile = path.Join(secureServing.ServerCert.CertDirectory, secureServing.ServerCert.PairName+".crt")
		secureServingInfo.ServerCert.KeyFile = path.Join(secureServing.ServerCert.CertDirectory, secureServing.ServerCert.PairName+".key")
	}

	secureServingInfo.SNICerts = nil
	for _, nkc := range secureServing.SNICertKeys {
		secureServingInfo.SNICerts = append(secureServingInfo.SNICerts, NamedCertKey{
			CertKey: CertKey{
				KeyFile:  nkc.KeyFile,
				CertFile: nkc.CertFile,
			},
			Names: nkc.Names,
		})
	}

	c.SecureServingInfo = secureServingInfo
	c.ReadWritePort = secureServing.ServingOptions.BindPort

	return c
}

func (c *Config) ApplyInsecureServingOptions(insecureServing *options.ServingOptions) *Config {
	if insecureServing == nil || insecureServing.BindPort <= 0 {
		return c
	}

	c.InsecureServingInfo = &ServingInfo{
		BindAddress: net.JoinHostPort(insecureServing.BindAddress.String(), strconv.Itoa(insecureServing.BindPort)),
	}

	return c
}

func (c *Config) ApplyAuthenticationOptions(o *options.BuiltInAuthenticationOptions) *Config {
	if o == nil || o.PasswordFile == nil {
		return c
	}

	c.SupportsBasicAuth = len(o.PasswordFile.BasicAuthFile) > 0
	return c
}

func (c *Config) ApplyRBACSuperUser(rbacSuperUser string) *Config {
	c.AuthorizerRBACSuperUser = rbacSuperUser
	return c
}

// ApplyOptions applies the run options to the method receiver and returns self
func (c *Config) ApplyOptions(options *options.ServerRunOptions) *Config {
	if len(options.AuditLogPath) != 0 {
		c.AuditWriter = &lumberjack.Logger{
			Filename:   options.AuditLogPath,
			MaxAge:     options.AuditLogMaxAge,
			MaxBackups: options.AuditLogMaxBackups,
			MaxSize:    options.AuditLogMaxSize,
		}
	}

	c.CorsAllowedOriginList = options.CorsAllowedOriginList
	c.EnableGarbageCollection = options.EnableGarbageCollection
	c.EnableProfiling = options.EnableProfiling
	c.EnableSwaggerUI = options.EnableSwaggerUI
	c.ExternalAddress = options.ExternalHost
	c.MaxRequestsInFlight = options.MaxRequestsInFlight
	c.MinRequestTimeout = options.MinRequestTimeout
	c.PublicAddress = options.AdvertiseAddress

	return c
}

type completedConfig struct {
	*Config
}

// Complete fills in any fields not set that are required to have valid data and can be derived
// from other fields.  If you're going to `ApplyOptions`, do that first.  It's mutating the receiver.
func (c *Config) Complete() completedConfig {
	if len(c.ExternalAddress) == 0 && c.PublicAddress != nil {
		hostAndPort := c.PublicAddress.String()
		if c.ReadWritePort != 0 {
			hostAndPort = net.JoinHostPort(hostAndPort, strconv.Itoa(c.ReadWritePort))
		}
		c.ExternalAddress = hostAndPort
	}
	// All APIs will have the same authentication for now.
	if c.OpenAPIConfig != nil && c.OpenAPIConfig.SecurityDefinitions != nil {
		c.OpenAPIConfig.DefaultSecurity = []map[string][]string{}
		keys := []string{}
		for k := range *c.OpenAPIConfig.SecurityDefinitions {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			c.OpenAPIConfig.DefaultSecurity = append(c.OpenAPIConfig.DefaultSecurity, map[string][]string{k: {}})
		}
		if c.OpenAPIConfig.CommonResponses == nil {
			c.OpenAPIConfig.CommonResponses = map[int]spec.Response{}
		}
		if _, exists := c.OpenAPIConfig.CommonResponses[http.StatusUnauthorized]; !exists {
			c.OpenAPIConfig.CommonResponses[http.StatusUnauthorized] = spec.Response{
				ResponseProps: spec.ResponseProps{
					Description: "Unauthorized",
				},
			}
		}
	}
	if c.DiscoveryAddresses == nil {
		c.DiscoveryAddresses = DefaultDiscoveryAddresses{DefaultAddress: c.ExternalAddress}
	}

	// If the loopbackclientconfig is specified AND it has a token for use against the API server
	// wrap the authenticator and authorizer in loopback authentication logic
	if c.Authenticator != nil && c.Authorizer != nil && c.LoopbackClientConfig != nil && len(c.LoopbackClientConfig.BearerToken) > 0 {
		privilegedLoopbackToken := c.LoopbackClientConfig.BearerToken
		var uid = uuid.NewRandom().String()
		tokens := make(map[string]*user.DefaultInfo)
		tokens[privilegedLoopbackToken] = &user.DefaultInfo{
			Name:   user.APIServerUser,
			UID:    uid,
			Groups: []string{user.SystemPrivilegedGroup},
		}

		tokenAuthenticator := apiserverauthenticator.NewAuthenticatorFromTokens(tokens)
		c.Authenticator = authenticatorunion.New(tokenAuthenticator, c.Authenticator)

		tokenAuthorizer := apiserverauthorizer.NewPrivilegedGroups(user.SystemPrivilegedGroup)
		c.Authorizer = authorizerunion.New(tokenAuthorizer, c.Authorizer)
	}

	return completedConfig{c}
}

// SkipComplete provides a way to construct a server instance without config completion.
func (c *Config) SkipComplete() completedConfig {
	return completedConfig{c}
}

// New returns a new instance of GenericAPIServer from the given config.
// Certain config fields will be set to a default value if unset,
// including:
//   ServiceClusterIPRange
//   ServiceNodePortRange
//   MasterCount
//   ReadWritePort
//   PublicAddress
// Public fields:
//   Handler -- The returned GenericAPIServer has a field TopHandler which is an
//   http.Handler which handles all the endpoints provided by the GenericAPIServer,
//   including the API, the UI, and miscellaneous debugging endpoints.  All
//   these are subject to authorization and authentication.
//   InsecureHandler -- an http.Handler which handles all the same
//   endpoints as Handler, but no authorization and authentication is done.
// Public methods:
//   HandleWithAuth -- Allows caller to add an http.Handler for an endpoint
//   that uses the same authentication and authorization (if any is configured)
//   as the GenericAPIServer's built-in endpoints.
//   If the caller wants to add additional endpoints not using the GenericAPIServer's
//   auth, then the caller should create a handler for those endpoints, which delegates the
//   any unhandled paths to "Handler".
func (c completedConfig) New() (*GenericAPIServer, error) {
	if c.Serializer == nil {
		return nil, fmt.Errorf("Genericapiserver.New() called with config.Serializer == nil")
	}

	s := &GenericAPIServer{
		discoveryAddresses:     c.DiscoveryAddresses,
		LoopbackClientConfig:   c.LoopbackClientConfig,
		legacyAPIGroupPrefixes: c.LegacyAPIGroupPrefixes,
		admissionControl:       c.AdmissionControl,
		requestContextMapper:   c.RequestContextMapper,
		Serializer:             c.Serializer,

		minRequestTimeout:    time.Duration(c.MinRequestTimeout) * time.Second,
		enableSwaggerSupport: c.EnableSwaggerSupport,

		SecureServingInfo:   c.SecureServingInfo,
		InsecureServingInfo: c.InsecureServingInfo,
		ExternalAddress:     c.ExternalAddress,

		apiGroupsForDiscovery: map[string]unversioned.APIGroup{},

		enableOpenAPISupport: c.EnableOpenAPISupport,
		openAPIConfig:        c.OpenAPIConfig,

		postStartHooks: map[string]postStartHookEntry{},
	}

	s.HandlerContainer = mux.NewAPIContainer(http.NewServeMux(), c.Serializer)

	s.installAPI(c.Config)

	s.Handler, s.InsecureHandler = c.BuildHandlerChainsFunc(s.HandlerContainer.ServeMux, c.Config)

	return s, nil
}

// MaybeGenerateServingCerts generates serving certificates if requested and needed.
func (c *Config) MaybeGenerateServingCerts(alternateIPs ...net.IP) error {
	// It would be nice to set a fqdn subject alt name, but only the kubelets know, the apiserver is clueless
	// alternateDNS = append(alternateDNS, "kubernetes.default.svc.CLUSTER.DNS.NAME")
	if c.SecureServingInfo != nil && c.SecureServingInfo.ServerCert.Generate && !certutil.CanReadCertOrKey(c.SecureServingInfo.ServerCert.CertFile, c.SecureServingInfo.ServerCert.KeyFile) {
		// TODO (cjcullen): Is ClusterIP the right address to sign a cert with?
		alternateDNS := []string{"kubernetes.default.svc", "kubernetes.default", "kubernetes", "localhost"}

		if cert, key, err := certutil.GenerateSelfSignedCertKey(c.PublicAddress.String(), alternateIPs, alternateDNS); err != nil {
			return fmt.Errorf("unable to generate self signed cert: %v", err)
		} else {
			if err := certutil.WriteCert(c.SecureServingInfo.ServerCert.CertFile, cert); err != nil {
				return err
			}

			if err := certutil.WriteKey(c.SecureServingInfo.ServerCert.KeyFile, key); err != nil {
				return err
			}
			glog.Infof("Generated self-signed cert (%s, %s)", c.SecureServingInfo.ServerCert.CertFile, c.SecureServingInfo.ServerCert.KeyFile)
		}
	}

	return nil
}

func DefaultBuildHandlerChain(apiHandler http.Handler, c *Config) (secure, insecure http.Handler) {
	attributeGetter := apiserverfilters.NewRequestAttributeGetter(c.RequestContextMapper)

	generic := func(handler http.Handler) http.Handler {
		handler = genericfilters.WithCORS(handler, c.CorsAllowedOriginList, nil, nil, nil, "true")
		handler = genericfilters.WithPanicRecovery(handler, c.RequestContextMapper)
		handler = apiserverfilters.WithRequestInfo(handler, NewRequestInfoResolver(c), c.RequestContextMapper)
		handler = api.WithRequestContext(handler, c.RequestContextMapper)
		handler = genericfilters.WithTimeoutForNonLongRunningRequests(handler, c.LongRunningFunc)
		handler = genericfilters.WithMaxInFlightLimit(handler, c.MaxRequestsInFlight, c.LongRunningFunc)
		return handler
	}
	audit := func(handler http.Handler) http.Handler {
		return apiserverfilters.WithAudit(handler, attributeGetter, c.AuditWriter)
	}
	protect := func(handler http.Handler) http.Handler {
		handler = apiserverfilters.WithAuthorization(handler, attributeGetter, c.Authorizer)
		handler = apiserverfilters.WithImpersonation(handler, c.RequestContextMapper, c.Authorizer)
		handler = audit(handler) // before impersonation to read original user
		handler = authhandlers.WithAuthentication(handler, c.RequestContextMapper, c.Authenticator, authhandlers.Unauthorized(c.SupportsBasicAuth))
		return handler
	}

	return generic(protect(apiHandler)), generic(audit(apiHandler))
}

func (s *GenericAPIServer) installAPI(c *Config) {
	if c.EnableIndex {
		routes.Index{}.Install(s.HandlerContainer)
	}
	if c.EnableSwaggerSupport && c.EnableSwaggerUI {
		routes.SwaggerUI{}.Install(s.HandlerContainer)
	}
	if c.EnableProfiling {
		routes.Profiling{}.Install(s.HandlerContainer)
	}
	if c.EnableMetrics {
		if c.EnableProfiling {
			routes.MetricsWithReset{}.Install(s.HandlerContainer)
		} else {
			routes.DefaultMetrics{}.Install(s.HandlerContainer)
		}
	}
	routes.Version{Version: c.Version}.Install(s.HandlerContainer)
	s.HandlerContainer.Add(s.DynamicApisDiscovery())
}

func DefaultAndValidateRunOptions(options *options.ServerRunOptions) {
	genericvalidation.ValidateRunOptions(options)

	glog.Infof("Will report %v as public IP address.", options.AdvertiseAddress)

	// Set default value for ExternalAddress if not specified.
	if len(options.ExternalHost) == 0 {
		// TODO: extend for other providers
		if options.CloudProvider == "gce" || options.CloudProvider == "aws" {
			cloud, err := cloudprovider.InitCloudProvider(options.CloudProvider, options.CloudConfigFile)
			if err != nil {
				glog.Fatalf("Cloud provider could not be initialized: %v", err)
			}
			instances, supported := cloud.Instances()
			if !supported {
				glog.Fatalf("%q cloud provider has no instances.  this shouldn't happen. exiting.", options.CloudProvider)
			}
			hostname, err := os.Hostname()
			if err != nil {
				glog.Fatalf("Failed to get hostname: %v", err)
			}
			nodeName, err := instances.CurrentNodeName(hostname)
			if err != nil {
				glog.Fatalf("Failed to get NodeName: %v", err)
			}
			addrs, err := instances.NodeAddresses(nodeName)
			if err != nil {
				glog.Warningf("Unable to obtain external host address from cloud provider: %v", err)
			} else {
				for _, addr := range addrs {
					if addr.Type == api.NodeExternalIP {
						options.ExternalHost = addr.Address
					}
				}
			}
		}
	}
}

func NewRequestInfoResolver(c *Config) *request.RequestInfoFactory {
	apiPrefixes := sets.NewString(strings.Trim(APIGroupPrefix, "/")) // all possible API prefixes
	legacyAPIPrefixes := sets.String{}                               // APIPrefixes that won't have groups (legacy)
	for legacyAPIPrefix := range c.LegacyAPIGroupPrefixes {
		apiPrefixes.Insert(strings.Trim(legacyAPIPrefix, "/"))
		legacyAPIPrefixes.Insert(strings.Trim(legacyAPIPrefix, "/"))
	}

	return &request.RequestInfoFactory{
		APIPrefixes:          apiPrefixes,
		GrouplessAPIPrefixes: legacyAPIPrefixes,
	}
}
