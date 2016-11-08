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

	Priority int64 `json:"priority" protobuf:"varint,5,opt,name=priority"`
}

type APIServerStatus struct {
	Group   string `json:"group,omitempty" protobuf:"bytes,1,opt,name=group"`
	Version string `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
}

// +genclient=true

// APIServer is a logical top-level container for a set of origin resources
type APIServer struct {
	unversioned.TypeMeta `json:",inline"`
	kapi.ObjectMeta      `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   APIServerSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status APIServerStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}
