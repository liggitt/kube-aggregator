package v1beta1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	kapi "k8s.io/kubernetes/pkg/api/v1"
)

// APIServerList is a list of APIServer objects.
type APIServerList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []APIServer `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type APIServerSpec struct {
	Group        string `json:"group,omitempty" protobuf:"bytes,1,opt,name=group"`
	Version      string `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
	InternalHost string `json:"internalHost,omitempty" protobuf:"bytes,3,opt,name=internalHost"`
	Prefix       string `json:"prefix,omitempty" protobuf:"bytes,4,opt,name=prefix"`

	InsecureSkipTLSVerify bool   `json:"insecureSkipTLSVerify,omitempty" protobuf:"varint,5,opt,name=insecureSkipTLSVerify"`
	CABundle              []byte `json:"caBundle" protobuf:"bytes,6,opt,name=caBundle"`

	Priority int64 `json:"priority" protobuf:"varint,7,opt,name=priority"`
}

type APIServerStatus struct {
}

type APIResource struct {
	Name         string           `json:"name" protobuf:"bytes,1,opt,name=name"`
	Namespaced   bool             `json:"namespaced" protobuf:"varint,2,opt,name=namespaced"`
	Kind         string           `json:"kind" protobuf:"bytes,3,opt,name=kind"`
	SubResources []APISubResource `json:"subresources" protobuf:"bytes,4,rep,name=subresources"`
}

type APISubResource struct {
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	Kind string `json:"kind" protobuf:"bytes,2,opt,name=kind"`
}

// +genclient=true
// +nonNamespaced=true

// APIServer is a logical top-level container for a set of origin resources
type APIServer struct {
	unversioned.TypeMeta `json:",inline"`
	kapi.ObjectMeta      `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   APIServerSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status APIServerStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}
