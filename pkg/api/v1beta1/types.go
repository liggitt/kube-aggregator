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

type APIServerConditionType string

const (
	APIServerAvailable APIServerConditionType = "Available"
)

type APIServerCondition struct {
	Type               APIServerConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=PodConditionType"`
	Status             kapi.ConditionStatus   `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	LastProbeTime      unversioned.Time       `json:"lastProbeTime,omitempty" protobuf:"bytes,3,opt,name=lastProbeTime"`
	LastTransitionTime unversioned.Time       `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`
	Reason             string                 `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	Message            string                 `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

type APIServerStatus struct {
	Group     string        `json:"group,omitempty" protobuf:"bytes,1,opt,name=group"`
	Version   string        `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
	Resources []APIResource `json:"resources" protobuf:"bytes,3,rep,name=resources"`

	Conditions []APIServerCondition `json:"conditions" protobuf:"bytes,4,rep,name=conditions"`
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

// APIServer is a logical top-level container for a set of origin resources
type APIServer struct {
	unversioned.TypeMeta `json:",inline"`
	kapi.ObjectMeta      `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   APIServerSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status APIServerStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}
