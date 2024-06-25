package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TrinoBackend is the CRD structure
type TrinoBackend struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TrinoBackendSpec `json:"spec"`
}

// TrinoBackendSpec defines the desired state of TrinoBackend
type TrinoBackendSpec struct {
	Name         string `json:"name"`
	ProxyTo      string `json:"proxyTo"`
	Active       bool   `json:"active"`
	RoutingGroup string `json:"routingGroup"`
	ExternalUrl  string `json:"externalUrl,omitempty"`
}

// TrinoBackendList contains a list of TrinoBackend
type TrinoBackendList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrinoBackend `json:"items"`
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TrinoBackend) DeepCopyInto(out *TrinoBackend) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrinoBackend.
func (in *TrinoBackend) DeepCopy() *TrinoBackend {
	if in == nil {
		return nil
	}
	out := new(TrinoBackend)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TrinoBackend) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TrinoBackendList) DeepCopyInto(out *TrinoBackendList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TrinoBackend, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrinoBackendList.
func (in *TrinoBackendList) DeepCopy() *TrinoBackendList {
	if in == nil {
		return nil
	}
	out := new(TrinoBackendList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TrinoBackendList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
