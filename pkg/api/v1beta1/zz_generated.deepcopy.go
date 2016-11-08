// +build !ignore_autogenerated_openshift

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package v1beta1

import (
	v1 "k8s.io/kubernetes/pkg/api/v1"
	conversion "k8s.io/kubernetes/pkg/conversion"
	runtime "k8s.io/kubernetes/pkg/runtime"
	reflect "reflect"
)

func init() {
	SchemeBuilder.Register(RegisterDeepCopies)
}

// RegisterDeepCopies adds deep-copy functions to the given scheme. Public
// to allow building arbitrary schemes.
func RegisterDeepCopies(scheme *runtime.Scheme) error {
	return scheme.AddGeneratedDeepCopyFuncs(
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1beta1_APIServer, InType: reflect.TypeOf(&APIServer{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1beta1_APIServerList, InType: reflect.TypeOf(&APIServerList{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1beta1_APIServerSpec, InType: reflect.TypeOf(&APIServerSpec{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1beta1_APIServerStatus, InType: reflect.TypeOf(&APIServerStatus{})},
	)
}

func DeepCopy_v1beta1_APIServer(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*APIServer)
		out := out.(*APIServer)
		out.TypeMeta = in.TypeMeta
		if newVal, err := c.DeepCopy(&in.ObjectMeta); err != nil {
			return err
		} else {
			out.ObjectMeta = *newVal.(*v1.ObjectMeta)
		}
		out.Spec = in.Spec
		out.Status = in.Status
		return nil
	}
}

func DeepCopy_v1beta1_APIServerList(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*APIServerList)
		out := out.(*APIServerList)
		out.TypeMeta = in.TypeMeta
		out.ListMeta = in.ListMeta
		if in.Items != nil {
			in, out := &in.Items, &out.Items
			*out = make([]APIServer, len(*in))
			for i := range *in {
				if err := DeepCopy_v1beta1_APIServer(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		} else {
			out.Items = nil
		}
		return nil
	}
}

func DeepCopy_v1beta1_APIServerSpec(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*APIServerSpec)
		out := out.(*APIServerSpec)
		out.Group = in.Group
		out.Version = in.Version
		out.InternalHost = in.InternalHost
		out.Prefix = in.Prefix
		out.Priority = in.Priority
		return nil
	}
}

func DeepCopy_v1beta1_APIServerStatus(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*APIServerStatus)
		out := out.(*APIServerStatus)
		out.Group = in.Group
		out.Version = in.Version
		return nil
	}
}